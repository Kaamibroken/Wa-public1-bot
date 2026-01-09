package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// --- 🧠 MEMORY SYSTEM ---
type MovieResult struct {
	Identifier string
	Title      string
	Year       string
	Downloads  int
}

// یوزر کی سرچ ہسٹری محفوظ کرنے کے لیے
var searchCache = make(map[string][]MovieResult)

// ⚠️ ہم نے نام تبدیل کر دیا تاکہ main.go والی cacheMutex سے ٹکراؤ نہ ہو
var movieMutex sync.Mutex 

// Archive API Response Structures (Flexible Types)
type IAHeader struct {
	Identifier string      `json:"identifier"`
	Title      string      `json:"title"`
	Year       interface{} `json:"year"`      // Can be string or int
	Downloads  interface{} `json:"downloads"` // Can be string or int
}

type IAResponse struct {
	Response struct {
		Docs []IAHeader `json:"docs"`
	} `json:"response"`
}

type IAMetadata struct {
	Files []struct {
		Name   string `json:"name"`
		Format string `json:"format"`
		Size   string `json:"size"` 
	} `json:"files"`
}

func handleArchive(client *whatsmeow.Client, v *events.Message, input string) {
	if input == "" { return }
	input = strings.TrimSpace(input)
	senderJID := v.Info.Sender.String()

	// --- 1️⃣ کیا یوزر نے نمبر سلیکٹ کیا ہے؟ (Selection Logic) ---
	if isNumber(input) {
		index, _ := strconv.Atoi(input)
		
		movieMutex.Lock()
		movies, exists := searchCache[senderJID]
		movieMutex.Unlock()

		if exists && index > 0 && index <= len(movies) {
			selectedMovie := movies[index-1]
			// یہاں ہم سلیکٹڈ مووی کو ڈاؤن لوڈ کریں گے
			react(client, v.Info.Chat, v.Info.ID, "💿")
			downloadFromIdentifier(client, v, selectedMovie)
			return
		}
	}

	// --- 2️⃣ کیا یہ ڈائریکٹ لنک ہے؟ (Direct Link Logic) ---
	if strings.HasPrefix(input, "http") {
		react(client, v.Info.Chat, v.Info.ID, "🔗")
		// پریمیم کارڈ ہٹا کر سادہ میسج
		replyMessage(client, v, "⏳ *Processing Direct Link...*")
		downloadFileDirectly(client, v, input, "Unknown_File")
		return
	}

	// --- 3️⃣ یہ سرچ کوئری ہے! (Search Logic) ---
	react(client, v.Info.Chat, v.Info.ID, "🔎")
	go performSearch(client, v, input, senderJID)
}

// --- 🔍 Helper: Search Engine (Fixed User-Agent) ---
func performSearch(client *whatsmeow.Client, v *events.Message, query string, senderJID string) {
	// Archive Advanced Search API
	encodedQuery := url.QueryEscape(fmt.Sprintf("title:(%s) AND mediatype:(movies)", query))
	apiURL := fmt.Sprintf("https://archive.org/advancedsearch.php?q=%s&fl[]=identifier&fl[]=title&fl[]=year&fl[]=downloads&sort[]=downloads+desc&output=json&rows=10", encodedQuery)

	// ✅ FIX: http.NewRequest استعمال کریں تاکہ ہیڈرز لگا سکیں
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	clientHttp := &http.Client{Timeout: 30 * time.Second}
	resp, err := clientHttp.Do(req)
	
	if err != nil {
		replyMessage(client, v, "❌ Search API Error.")
		return
	}
	defer resp.Body.Close()

	// ڈیبگنگ کے لیے: اگر سٹیٹس 200 نہیں ہے تو ایرر دیں
	if resp.StatusCode != 200 {
		replyMessage(client, v, fmt.Sprintf("❌ API Error: %d", resp.StatusCode))
		return
	}

	var result IAResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		replyMessage(client, v, "❌ Data Parse Error (Invalid JSON).")
		return
	}

	docs := result.Response.Docs
	if len(docs) == 0 {
		replyMessage(client, v, "🚫 No movies found for: *"+query+"*")
		return
	}

	// میموری میں محفوظ کریں
	var movieList []MovieResult
	msgText := fmt.Sprintf("🎬 *Archive Results for:* '%s'\n\n", query)

	for i, doc := range docs {
		// ✅ Safe Conversion (Interface to String/Int)
		yearStr := fmt.Sprintf("%v", doc.Year)
		
		dlCount := 0
		switch val := doc.Downloads.(type) {
		case float64:
			dlCount = int(val)
		case string:
			dlCount, _ = strconv.Atoi(val)
		}

		movieList = append(movieList, MovieResult{
			Identifier: doc.Identifier,
			Title:      doc.Title,
			Year:       yearStr,
			Downloads:  dlCount,
		})
		msgText += fmt.Sprintf("*%d.* %s (%s)\n", i+1, doc.Title, yearStr)
	}
	
	msgText += "\n👇 *Reply with a number to download.*"

	// گلوبل کیشے اپڈیٹ کریں
	movieMutex.Lock()
	searchCache[senderJID] = movieList
	movieMutex.Unlock()

	// لسٹ بھیجیں
	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(msgText),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      proto.String(v.Info.ID),
				Participant:   proto.String(v.Info.Sender.String()),
				QuotedMessage: v.Message,
			},
		},
	})
}

// --- 📥 Helper: Find Best Video & Download ---
func downloadFromIdentifier(client *whatsmeow.Client, v *events.Message, movie MovieResult) {
	// Metadata API سے فائلز کی لسٹ لیں
	metaURL := fmt.Sprintf("https://archive.org/metadata/%s", movie.Identifier)
	
	// ✅ FIX: Metadata Request میں بھی User-Agent لگائیں
	req, _ := http.NewRequest("GET", metaURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	
	clientHttp := &http.Client{Timeout: 30 * time.Second}
	resp, err := clientHttp.Do(req)
	
	if err != nil { return }
	defer resp.Body.Close()

	var meta IAMetadata
	json.NewDecoder(resp.Body).Decode(&meta)

	bestFile := ""
	maxSize := int64(0)

	for _, f := range meta.Files {
		// فارمیٹ کلیننگ
		fName := strings.ToLower(f.Name)
		// ہم mp4 اور mkv کو ترجیح دیں گے، لیکن ogv/avi کو چھوڑ دیں گے اگر ممکن ہو
		if strings.HasSuffix(fName, ".mp4") || strings.HasSuffix(fName, ".mkv") {
			s, _ := strconv.ParseInt(f.Size, 10, 64)
			if s > maxSize {
				maxSize = s
				bestFile = f.Name
			}
		}
	}

	if bestFile == "" {
		replyMessage(client, v, "❌ No suitable video file found.")
		return
	}

	finalURL := fmt.Sprintf("https://archive.org/download/%s/%s", movie.Identifier, url.PathEscape(bestFile))
	
	replyMessage(client, v, fmt.Sprintf("🚀 *Downloading:* %s\n📦 *Please wait...*", movie.Title))
	
	go downloadFileDirectly(client, v, finalURL, movie.Title)
}

// --- 🚀 Core Downloader ---
func downloadFileDirectly(client *whatsmeow.Client, v *events.Message, urlStr string, customTitle string) {
	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	
	clientHttp := &http.Client{}
	resp, err := clientHttp.Do(req)
	if err != nil || resp.StatusCode != 200 {
		replyMessage(client, v, "❌ Download Failed (Link Invalid).")
		return
	}
	defer resp.Body.Close()

	// نام نکالنا
	fileName := customTitle
	if fileName == "Unknown_File" {
		parts := strings.Split(urlStr, "/")
		fileName = parts[len(parts)-1]
	}
	if !strings.Contains(fileName, ".") { fileName += ".mp4" }

	// Temp File
	tempFile := fmt.Sprintf("temp_%d_%s", time.Now().UnixNano(), fileName)
	out, _ := os.Create(tempFile)
	io.Copy(out, resp.Body)
	out.Close()

	fileData, _ := os.ReadFile(tempFile)
	defer os.Remove(tempFile)

	up, err := client.Upload(context.Background(), fileData, whatsmeow.MediaDocument)
	if err != nil {
		replyMessage(client, v, "❌ Upload Failed.")
		return
	}

	// Send Logic (Simple Video Message)
	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			Mimetype:      proto.String("video/mp4"),
			Title:         proto.String(fileName),
			FileName:      proto.String(fileName),
			FileLength:    proto.Uint64(uint64(len(fileData))),
			FileSHA256:    up.FileSHA256,
			FileEncSHA256: up.FileEncSHA256,
			Caption:       proto.String("✅ " + fileName),
		},
	})
	react(client, v.Info.Chat, v.Info.ID, "✅")
}

// ✅ helper function
func isNumber(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
