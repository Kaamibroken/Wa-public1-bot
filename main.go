package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var client *whatsmeow.Client
var container *sqlstore.Container

// میسج ٹیکسٹ نکالنے کا لاجک
func getBody(msg *waProto.Message) string {
	if msg == nil { return "" }
	if msg.Conversation != nil { return msg.GetConversation() }
	if msg.ExtendedTextMessage != nil { return msg.ExtendedTextMessage.GetText() }
	if msg.ImageMessage != nil { return msg.ImageMessage.GetCaption() }
	if msg.VideoMessage != nil { return msg.VideoMessage.GetCaption() }
	if msg.ViewOnceMessageV2 != nil { return getBody(msg.ViewOnceMessageV2.Message) }
	return ""
}

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if v.Info.IsFromMe { return }

		body := strings.TrimSpace(getBody(v.Message))
		isGroup := v.Info.IsGroup
		
		// تفصیلی لاگنگ
		fmt.Printf("📩 [Message] Group: %v | From: %s | Text: %s\n", isGroup, v.Info.Sender.String(), body)

		if strings.ToLower(body) == "#menu" {
			fmt.Printf("⚙️ [Action] Sending menu to %s\n", v.Info.Chat)
			
			// ری ایکشن بھیجیں - اب گروپ کے لیے بھی فکس ہے
			_, err := client.SendMessage(context.Background(), v.Info.Chat, client.BuildReaction(v.Info.Chat, v.Info.Sender, v.Info.ID, "📜"))
			if err != nil { fmt.Printf("⚠️ Reaction Error: %v\n", err) }

			sendMenuFixed(v.Info.Chat)
		}
	}
}

func sendMenuFixed(chat types.JID) {
	imgData, err := os.ReadFile("./web/pic.png")
	if err != nil {
		fmt.Println("❌ pic.png missing")
		client.SendMessage(context.Background(), chat, &waProto.Message{Conversation: proto.String("*📜 MENU*\n(Image missing)")})
		return
	}

	// 1. تصویر اپلوڈ کرنا
	uploadResp, err := client.Upload(context.Background(), imgData, whatsmeow.MediaImage)
	if err != nil {
		fmt.Printf("❌ Upload failed: %v\n", err)
		return
	}

	// 2. پہلے تصویر اور ٹیکسٹ بھیجیں (تاکہ ایرر 479 نہ آئے)
	caption := "*📜 IMPOSSIBLE MENU*\n\n" +
		"• #ping - Check Latency\n" +
		"• #id - Get Chat Info\n\n" +
		"Click the MENU button below for all commands."

	imageMsg := &waProto.ImageMessage{
		Mimetype:      proto.String("image/png"),
		Caption:       proto.String(caption),
		URL:           &uploadResp.URL,
		DirectPath:    &uploadResp.DirectPath,
		MediaKey:      uploadResp.MediaKey,
		FileEncSHA256: uploadResp.FileEncSHA256,
		FileSHA256:    uploadResp.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(imgData))),
	}

	fmt.Println("📤 Sending Image Component...")
	_, err = client.SendMessage(context.Background(), chat, &waProto.Message{ImageMessage: imageMsg})
	if err != nil { fmt.Printf("⚠️ Image Send Failed: %v\n", err) }

	// 3. اب لسٹ مینیو الگ سے بھیجیں
	listMsg := &waProto.ListMessage{
		Title:       proto.String("SELECT CATEGORY"),
		ButtonText:  proto.String("MENU"),
		ListType:    waProto.ListMessage_SINGLE_SELECT.Enum(),
		Sections: []*waProto.ListMessage_Section{
			{
				Title: proto.String("TOOLS"),
				Rows: []*waProto.ListMessage_Row{
					{Title: proto.String("Ping"), RowID: proto.String("ping")},
					{Title: proto.String("ID"), RowID: proto.String("id")},
				},
			},
		},
	}

	fmt.Println("📤 Sending Button Component...")
	_, err = client.SendMessage(context.Background(), chat, &waProto.Message{ListMessage: listMsg})
	if err != nil {
		fmt.Printf("⚠️ Button Menu Error 479: This account/chat doesn't support buttons. Sending fallback text.\n")
	}
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	dbType := "postgres"
	if dbURL == "" { dbType = "sqlite3"; dbURL = "file:impossible.db?_foreign_keys=on" }

	container, _ = sqlstore.New(context.Background(), dbType, dbURL, waLog.Stdout("Database", "INFO", true))
	deviceStore, _ := container.GetFirstDevice(context.Background())
	client = whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))
	client.AddEventHandler(eventHandler)

	// ویب سرور
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	r := gin.Default()
	r.StaticFile("/", "./web/index.html")
	r.StaticFile("/pic.png", "./web/pic.png")
	r.POST("/api/pair", handlePairAPI)

	go r.Run(":" + port)
	
	if client.Store.ID != nil { client.Connect() }

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	client.Disconnect()
}

func handlePairAPI(c *gin.Context) {
	var req struct{ Number string `json:"number"` }
	c.BindJSON(&req)
	num := strings.ReplaceAll(req.Number, "+", "")
	
	newDevice := container.NewDevice()
	if client.IsConnected() { client.Disconnect() }
	client = whatsmeow.NewClient(newDevice, waLog.Stdout("Client", "INFO", true))
	client.AddEventHandler(eventHandler)
	client.Connect()
	
	time.Sleep(10 * time.Second)
	code, _ := client.PairPhone(context.Background(), num, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	c.JSON(200, gin.H{"code": code})
}