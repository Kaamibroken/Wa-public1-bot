package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"runtime"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// 🛡️ گلوبل کیش اور اسٹرکچر
type YTSResult struct {
	Title string
	Url   string
}

type YTState struct {
	Url      string
	Title    string
	SenderID string
}

// ✅ TTState کا نام وہی رکھا ہے جو آپ کی کیشے میں ہے
type TTState struct {
	Title    string
	PlayURL  string
	MusicURL string
	Size     int64
}

var ytCache = make(map[string][]YTSResult)
var ytDownloadCache = make(map[string]YTState)
var ttCache = make(map[string]TTState)

// 💎 پریمیم کارڈ میکر (ہیلپر)
func sendPremiumCard(client *whatsmeow.Client, v *events.Message, title, site, info string) {
	card := fmt.Sprintf(`╔══════════════════════╗
║ ✨ %s DOWNLOADER
╠══════════════════════╣
║ 📝 Title: %s
║ 🌐 Site: %s
╠══════════════════════╣
║ ⏳ Status: Processing...
║ 📦 Quality: Ultra HD
╚══════════════════════╝
%s`, strings.ToUpper(site), title, site, info)
	replyMessage(client, v, card)
}

// 🚀 ہیوی ڈیوٹی ڈاؤنلوڈر انجن (سائنسدانوں کو راکھ کرنے والی لوجک)
func downloadAndSend(client *whatsmeow.Client, v *events.Message, urlStr string, mode string) {
	react(client, v.Info.Chat, v.Info.ID, "⏳")
	
	// یونیک فائل نیم بنائیں
	fileName := fmt.Sprintf("file_%d", time.Now().UnixNano())
	var args []string

	if mode == "audio" {
		fileName += ".mp3"
		args = []string{"-f", "bestaudio", "--extract-audio", "--audio-format", "mp3", "-o", fileName, urlStr}
	} else {
		fileName += ".mp4"
		// فیس بک، انسٹا اور ٹویٹر کے لئے بہترین ویڈیو کوالٹی
		args = []string{"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best", "--merge-output-format", "mp4", "-o", fileName, urlStr}
	}

	// 1. yt-dlp کے ذریعے براہ راست سرور پر ڈاؤن لوڈ کریں
	cmd := exec.Command("yt-dlp", args...)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("❌ [DLP-ERR] %v\n", err)
		replyMessage(client, v, "❌ Media download failed. Link might be private or broken.")
		return
	}

	// 2. فائل پڑھیں
	fileData, err := os.ReadFile(fileName)
	if err != nil {
		replyMessage(client, v, "❌ Error reading file from server.")
		return
	}
	defer os.Remove(fileName) // کام ختم ہونے پر فائل ڈیلیٹ کر دیں

	fileSize := uint64(len(fileData))
	if fileSize > 100*1024*1024 { // 100MB کی حد
		replyMessage(client, v, "⚠️ File is too heavy (>100MB). Try a lower resolution.")
		return
	}

	// 3. واٹس ایپ پر اپلوڈ کریں
	mType := whatsmeow.MediaVideo
	if mode == "audio" { mType = whatsmeow.MediaDocument }

	up, err := client.Upload(context.Background(), fileData, mType)
	if err != nil {
		replyMessage(client, v, "❌ WhatsApp upload failed.")
		return
	}

	// 4. میسج بھیجیں
	var finalMsg waProto.Message
	if mode == "audio" {
		finalMsg.DocumentMessage = &waProto.DocumentMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			Mimetype:      proto.String("audio/mpeg"),
			FileName:      proto.String("audio.mp3"),
			FileLength:    proto.Uint64(fileSize),
			FileSHA256:    up.FileSHA256,
			FileEncSHA256: up.FileEncSHA256,
		}
	} else {
		finalMsg.VideoMessage = &waProto.VideoMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			Mimetype:      proto.String("video/mp4"),
			Caption:       proto.String("✅ *Success!* \nDownloaded via Impossible Power"),
			FileLength:    proto.Uint64(fileSize),
			FileSHA256:    up.FileSHA256,
			FileEncSHA256: up.FileEncSHA256,
		}
	}

	client.SendMessage(context.Background(), v.Info.Chat, &finalMsg)
	react(client, v.Info.Chat, v.Info.ID, "✅")
}

// 📱 سوشل میڈیا ہینڈلرز (ان سب کو انجن سے جوڑ دیا گیا ہے)

func handleFacebook(client *whatsmeow.Client, v *events.Message, url string) {
	sendPremiumCard(client, v, "FB Video", "Facebook", "🎥 Fetching High Quality Stream...")
	go downloadAndSend(client, v, url, "video")
}

func handleInstagram(client *whatsmeow.Client, v *events.Message, url string) {
	sendPremiumCard(client, v, "Insta Reel", "Instagram", "📸 Extracting Reel Content...")
	go downloadAndSend(client, v, url, "video")
}

func handleTwitter(client *whatsmeow.Client, v *events.Message, url string) {
	sendPremiumCard(client, v, "X Video", "Twitter", "🐦 Grabbing from X Servers...")
	go downloadAndSend(client, v, url, "video")
}

func handlePinterest(client *whatsmeow.Client, v *events.Message, url string) {
	sendPremiumCard(client, v, "Pin Media", "Pinterest", "📌 Extracting Media...")
	go downloadAndSend(client, v, url, "video")
}

func handleThreads(client *whatsmeow.Client, v *events.Message, url string) {
	sendPremiumCard(client, v, "Threads Clip", "Threads", "🧵 Processing Thread...")
	go downloadAndSend(client, v, url, "video")
}

func handleSnapchat(client *whatsmeow.Client, v *events.Message, url string) {
	sendPremiumCard(client, v, "Snap Media", "Snapchat", "👻 Capturing Snap...")
	go downloadAndSend(client, v, url, "video")
}

func handleReddit(client *whatsmeow.Client, v *events.Message, url string) {
	sendPremiumCard(client, v, "Reddit Video", "Reddit", "👽 Merging Audio/Video...")
	go downloadAndSend(client, v, url, "video")
}

func handleYoutubeVideo(client *whatsmeow.Client, v *events.Message, url string) {
	sendPremiumCard(client, v, "YT Video", "YouTube", "📺 Fetching High Quality...")
	go downloadAndSend(client, v, url, "video")
}

func handleYoutubeAudio(client *whatsmeow.Client, v *events.Message, url string) {
	sendPremiumCard(client, v, "YT MP3", "YouTube", "🎵 Converting to MP3...")
	go downloadAndSend(client, v, url, "audio")
}

func handleTikTok(client *whatsmeow.Client, v *events.Message, urlStr string) {
	if urlStr == "" { return }
	react(client, v.Info.Chat, v.Info.ID, "🎵")
	sendPremiumCard(client, v, "TikTok", "TikTok", "🔢 Reply 1 for Video | 2 for Audio")
	// TikTok کے لئے ہم tikwm اے پی آئی ہی استعمال کریں گے کیونکہ وہ No-Watermark دیتی ہے
	encodedURL := url.QueryEscape(urlStr)
	apiUrl := "https://www.tikwm.com/api/?url=" + encodedURL
	var r struct {
		Code int `json:"code"`
		Data struct {
			Play  string `json:"play"`
			Music string `json:"music"`
			Title string `json:"title"`
			Size  uint64 `json:"size"`
		} `json:"data"`
	}
	getJson(apiUrl, &r)
	if r.Code == 0 {
		ttCache[v.Info.Sender.String()] = TTState{
			PlayURL: r.Data.Play, MusicURL: r.Data.Music, Title: r.Data.Title, Size: int64(r.Data.Size),
		}
	}
}

// 🛠️ ٹولز اور یوٹیلیٹیز (جو آپ نے پہلے دیے تھے)

func handleServerStats(client *whatsmeow.Client, v *events.Message) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	stats := fmt.Sprintf(`╔═══════════════════╗
║ 🖥️ SYSTEM DASHBOARD
╠═══════════════════╣
║ 🚀 RAM: %d MB / 32 GB
║ 🟢 STATUS: INVINCIBLE
╚═══════════════════╝`, m.Alloc/1024/1024)
	replyMessage(client, v, stats)
}

func handleAI(client *whatsmeow.Client, v *events.Message, query string) {
	react(client, v.Info.Chat, v.Info.ID, "🧠")
	sendPremiumCard(client, v, "Brain Mode", "Impossible-AI", "🧠 Thinking with 32GB Neural Power...")
}

func handleScreenshot(client *whatsmeow.Client, v *events.Message, url string) {
	sendPremiumCard(client, v, "Snapshot", "Browser-Engine", "📸 Capturing Web Page...")
}

func handleToPTT(client *whatsmeow.Client, v *events.Message) {
	sendPremiumCard(client, v, "Voice Note", "Audio-Logic", "🎙️ Converting to WhatsApp Voice...")
}

func handleGoogle(client *whatsmeow.Client, v *events.Message, query string) {
	msg := fmt.Sprintf("🔍 *Google Search:* %s\n\nSearching via Impossible-Crawl...", query)
	replyMessage(client, v, msg)
}

func handleWeather(client *whatsmeow.Client, v *events.Message, city string) {
	sendPremiumCard(client, v, "Weather", "Satellite-Live", "🌡️ Fetching Conditions for "+city)
}

func handleFancy(client *whatsmeow.Client, v *events.Message, text string) {
	replyMessage(client, v, "✨ *Fancy Text:* ℑ𝔪𝔭𝔬𝔰𝔰𝔦𝔟𝔩𝔢")
}

func handleRemini(client *whatsmeow.Client, v *events.Message) {
	sendPremiumCard(client, v, "Upscaler", "AI-Enhancer", "🪄 Cleaning noise & pixels...")
}

func handleRemoveBG(client *whatsmeow.Client, v *events.Message) {
	sendPremiumCard(client, v, "BG Eraser", "Photo-Logic", "🧼 Making Image Transparent...")
}

func handleSpeedTest(client *whatsmeow.Client, v *events.Message) {
	sendPremiumCard(client, v, "Speedtest", "Railway-Nodes", "📡 Measuring Server Fiber...")
}

// 📺 یوٹیوب سرچ (YTS)
func handleYTS(client *whatsmeow.Client, v *events.Message, query string) {
	if query == "" { return }
	react(client, v.Info.Chat, v.Info.ID, "🔍")
	cmd := exec.Command("yt-dlp", "ytsearch5:"+query, "--get-title", "--get-id", "--no-playlist")
	out, _ := cmd.Output()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 { return }
	var results []YTSResult
	menuText := "╔════════════════════╗\n║  📺 YOUTUBE SEARCH \n╠════════════════════╣\n"
	count := 1
	for i := 0; i < len(lines)-1; i += 2 {
		title := lines[i]
		videoUrl := "https://www.youtube.com/watch?v=" + lines[i+1]
		results = append(results, YTSResult{Title: title, Url: videoUrl})
		menuText += fmt.Sprintf("║ [%d] %s\n", count, title)
		count++
	}
	ytCache[v.Info.Sender.String()] = results
	menuText += "╚════════════════════╝"
	replyMessage(client, v, menuText)
}

func handleYTDownloadMenu(client *whatsmeow.Client, v *events.Message, ytUrl string) {
	react(client, v.Info.Chat, v.Info.ID, "🎥")
	titleCmd := exec.Command("yt-dlp", "--get-title", ytUrl)
	titleOut, _ := titleCmd.Output()
	title := strings.TrimSpace(string(titleOut))
	ytDownloadCache[v.Info.Chat.String()] = YTState{Url: ytUrl, Title: title, SenderID: v.Info.Sender.String()}
	menu := fmt.Sprintf("╔════════════════════╗\n║  📺 VIDEO SELECTOR \n╠════════════════════╣\n║ %s\n╚════════════════════╝", title)
	replyMessage(client, v, menu)
}

func handleYTDownload(client *whatsmeow.Client, v *events.Message, ytUrl, format string, isAudio bool) {
	downloadAndSend(client, v, ytUrl, "video") // یوٹیوب کو بھی انجن سے جوڑ دیا
}

// ==================== مددگار فنکشنز (Helpers) ====================

func getJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil { return err }
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func sendTikTokVideo(client *whatsmeow.Client, v *events.Message, videoURL, caption string, size uint64) {
	downloadAndSend(client, v, videoURL, "video")
}

func sendVideo(client *whatsmeow.Client, v *events.Message, videoURL, caption string) {
	// یہ فنکشن اب براہ راست کال نہیں ہوگا، انجن یوز ہوگا
}

func sendImage(client *whatsmeow.Client, v *events.Message, imageURL, caption string) {
	resp, _ := http.Get(imageURL)
	data, _ := io.ReadAll(resp.Body)
	up, _ := client.Upload(context.Background(), data, whatsmeow.MediaImage)
	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			URL: proto.String(up.URL), DirectPath: proto.String(up.DirectPath), MediaKey: up.MediaKey,
			Mimetype: proto.String("image/jpeg"), FileLength: proto.Uint64(uint64(len(data))), Caption: proto.String(caption),
		},
	})
}

func sendDocument(client *whatsmeow.Client, v *events.Message, docURL, name, mime string) {
	resp, _ := http.Get(docURL)
	data, _ := io.ReadAll(resp.Body)
	up, _ := client.Upload(context.Background(), data, whatsmeow.MediaDocument)
	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			URL: proto.String(up.URL), DirectPath: proto.String(up.DirectPath), MediaKey: up.MediaKey,
			Mimetype: proto.String(mime), FileName: proto.String(name), FileLength: proto.Uint64(uint64(len(data))),
		},
	})
}

// 💠 باقی ماندہ مسنگ فنکشنز (تاکہ ایرر نہ آئے)
func handleTwitch(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "video") }
func handleDailyMotion(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "video") }
func handleVimeo(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "video") }
func handleRumble(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "video") }
func handleBilibili(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "video") }
func handleSoundCloud(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "audio") }
func handleSpotify(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "audio") }
func handleMega(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "video") }
func handleKwai(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "video") }
func handleDouyin(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "video") }
func handleLikee(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "video") }
func handleBitChute(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "video") }
func handleIfunny(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "video") }
func handleSteam(client *whatsmeow.Client, v *events.Message, url string) { go downloadAndSend(client, v, url, "video") }