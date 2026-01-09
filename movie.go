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

var searchCache = make(map[string][]MovieResult)
var movieMutex sync.Mutex 

// Archive API Response Structures
type IAHeader struct {
	Identifier string      `json:"identifier"`
	Title      string      `json:"title"`
	Year       interface{} `json:"year"`
	Downloads  interface{} `json:"downloads"`
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
			
			// 🔥 فورا ریسپانس تاکہ یوزر کو پتہ چلے بوٹ زندہ ہے
			react(client, v.Info.Chat, v.Info.ID, "🔄")
			replyMessage(client, v, fmt.Sprintf("🔎 *Checking files for:* %s\nPlease wait...", selectedMovie.Title))
			
			// بیک گراؤنڈ میں پروسیس شروع
			go downloadFromIdentifier(client, v, selectedMovie)
			
			// میموری صاف نہ کریں تاکہ یوزر دوسری مووی بھی ڈاؤن لوڈ کر سکے
			return
		}
	}

	// --- 2️⃣ کیا یہ ڈائریکٹ لنک ہے؟ ---
	if strings.HasPrefix(input, "http") {
		react(client, v.Info.Chat, v.Info.ID, "🔗")
		replyMessage(client, v, "⏳ *Processing Direct Link...*")
		go downloadFileDirectly(client, v, input, "Unknown_File")
		return
	}

	// --- 3️⃣ یہ سرچ کوئری ہے! ---
	react(client, v.Info.Chat, v.Info.ID, "🔎")
	go performSearch(client, v, input, senderJID)
}

// --- 🔍 Helper: Search Engine ---
func performSearch(client *whatsmeow.Client, v *events.Message, query string, senderJID string) {
	encodedQuery := url.QueryEscape(fmt.Sprintf("title:(%s) AND mediatype:(movies)", query))
	apiURL := fmt.Sprintf("https://archive.org/advancedsearch.php?q=%s&fl[]=identifier&fl[]=title&fl[]=year&fl[]=downloads&sort[]=downloads+desc&output=json&rows=10", encodedQuery)

	req, _ := http.NewRequest("GET", apiURL, nil)
	// Archive کبھی کبھی بلاک کرتا ہے اگر User-Agent نہ ہو
	req.Header.Set("User-Agent", "Mozilla/5.0")

	// سرچ کے لیے 30 سیکنڈ ٹائم آؤٹ کافی ہے
	clientHttp := &http.Client{Timeout: 30 * time.Second}
	resp, err := clientHttp.Do(req)
	
	if err != nil {
		replyMessage(client, v, "❌ Network Error: Could not reach Archive API.")
		return
	}
	defer resp.Body.Close()

	var result IAResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		replyMessage(client, v, "❌ API Error: Archive.org returned invalid data.")
		return
	}

	docs := result.Response.Docs
	if len(docs) == 0 {
		replyMessage(client, v, "🚫 No movies found. Try a different name.")
		return
	}

	var movieList []MovieResult
	msgText := fmt.Sprintf("🎬 *Archive Results for:* '%s'\n\n", query)

	for i, doc := range docs {
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

	movieMutex.Lock()
	searchCache[senderJID] = movieList
	movieMutex.Unlock()

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
	fmt.Println("🔍 [ARCHIVE] Fetching metadata for:", movie.Identifier)
	
	metaURL := fmt.Sprintf("https://archive.org/metadata/%s", movie.Identifier)
	resp, err := http.Get(metaURL)
	if err != nil { 
		replyMessage(client, v, "❌ Metadata Error: Could not fetch file list.")
		return 
	}
	defer resp.Body.Close()

	var meta IAMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		replyMessage(client, v, "❌ Metadata Error: JSON parse failed.")
		return
	}

	bestFile := ""
	maxSize := int64(0)

	fmt.Printf("📂 [ARCHIVE] Found %d files. Scanning for video...\n", len(meta.Files))

	for _, f := range meta.Files {
		fName := strings.ToLower(f.Name)
		// صرف MP4 اور MKV کو ترجیح دیں
		if strings.HasSuffix(fName, ".mp4") || strings.HasSuffix(fName, ".mkv") {
			s, _ := strconv.ParseInt(f.Size, 10, 64)
			// سب سے بڑی فائل اٹھائیں (تاکہ ٹریلر ڈاؤن لوڈ نہ ہو)
			if s > maxSize {
				maxSize = s
				bestFile = f.Name
			}
		}
	}

	if bestFile == "" {
		replyMessage(client, v, "❌ Sorry! No .mp4 or .mkv video files found in this archive.")
		return
	}

	finalURL := fmt.Sprintf("https://archive.org/download/%s/%s", movie.Identifier, url.PathEscape(bestFile))
	
	// سائز کو MB میں دکھانے کے لیے
	sizeMB := float64(maxSize) / (1024 * 1024)
	
	infoMsg := fmt.Sprintf("🚀 *Starting Download!*\n\n🎬 *Title:* %s\n📁 *File:* %s\n📊 *Size:* %.2f MB\n\n_Please wait, downloading large files takes time..._", movie.Title, bestFile, sizeMB)
	replyMessage(client, v, infoMsg)
	
	fmt.Printf("🚀 [ARCHIVE] Starting Download: %s (%.2f MB)\n", bestFile, sizeMB)

	// اب اصل ڈاؤن لوڈنگ شروع
	downloadFileDirectly(client, v, finalURL, movie.Title)
}

// --- 🚀 Core Downloader ---
func downloadFileDirectly(client *whatsmeow.Client, v *events.Message, urlStr string, customTitle string) {
	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	
	// 🔥 اہم تبدیلی: یہاں ٹائم آؤٹ ہٹا دیا ہے تاکہ بڑی مووی پوری ڈاؤن لوڈ ہو سکے
	clientHttp := &http.Client{
		Timeout: 0, // No Timeout (Infinite wait for large files)
	}
	
	resp, err := clientHttp.Do(req)
	if err != nil {
		replyMessage(client, v, fmt.Sprintf("❌ Connection Error: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		replyMessage(client, v, fmt.Sprintf("❌ Server Error: HTTP %d", resp.StatusCode))
		return
	}

	fileName := customTitle
	if fileName == "Unknown_File" {
		parts := strings.Split(urlStr, "/")
		fileName = parts[len(parts)-1]
	}
	// اسپیشل کیریکٹرز ہٹا دیں جو فائل سسٹم خراب کر سکتے ہیں
	fileName = strings.ReplaceAll(fileName, "/", "_")
	fileName = strings.ReplaceAll(fileName, "\\", "_")
	if !strings.Contains(fileName, ".") { fileName += ".mp4" }

	tempFile := fmt.Sprintf("temp_%d_%s", time.Now().UnixNano(), fileName)
	out, err := os.Create(tempFile)
	if err != nil {
		replyMessage(client, v, "❌ System Error: Could not create temp file.")
		return
	}
	
	// فائل ڈاؤن لوڈ ہو رہی ہے
	_, err = io.Copy(out, resp.Body)
	out.Close()

	if err != nil {
		replyMessage(client, v, "❌ Download Interrupted: Network fail.")
		os.Remove(tempFile)
		return
	}

	// فائل ریڈ کریں
	fileData, err := os.ReadFile(tempFile)
	if err != nil {
		replyMessage(client, v, "❌ File Error: Could not read downloaded file.")
		return
	}
	defer os.Remove(tempFile)

	fmt.Println("✅ [ARCHIVE] Download Complete. Uploading to WhatsApp...")

	// اپلوڈ کریں
	up, err := client.Upload(context.Background(), fileData, whatsmeow.MediaDocument)
	if err != nil {
		replyMessage(client, v, fmt.Sprintf("❌ WhatsApp Upload Failed: %v", err))
		return
	}

	// بھیجیں
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
			Caption:       proto.String("✅ *Done:* " + fileName),
		},
	})
	react(client, v.Info.Chat, v.Info.ID, "✅")
	fmt.Println("✅ [ARCHIVE] Sent Successfully!")
}

// ✅ helper function
func isNumber(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
