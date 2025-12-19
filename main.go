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

// میسج سے ٹیکسٹ نکالنے کا تفصیلی طریقہ
func getBody(msg *waProto.Message) string {
	if msg == nil { return "" }
	if msg.Conversation != nil { return msg.GetConversation() }
	if msg.ExtendedTextMessage != nil { return msg.ExtendedTextMessage.GetText() }
	if msg.ImageMessage != nil { return msg.ImageMessage.GetCaption() }
	if msg.VideoMessage != nil { return msg.VideoMessage.GetCaption() }
	if msg.ViewOnceMessageV2 != nil { return getBody(msg.ViewOnceMessageV2.Message) }
	return ""
}

// ایونٹ ہینڈلر: ہر میسج پر لاگ پرنٹ کرے گا
func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		fmt.Println("✅ [Status] Connection Established Successfully!")
	case *events.Message:
		if v.Info.IsFromMe { return }
		
		body := strings.TrimSpace(getBody(v.Message))
		fmt.Printf("📩 [Incoming] From: %s | Text: '%s' | Type: %T\n", v.Info.Sender.User, body, v.Message)

		if body == "#menu" {
			fmt.Printf("⚙️ [Action] #menu command detected from %s\n", v.Info.Sender.User)
			// ری ایکشن دیں
			_, _ = client.SendMessage(context.Background(), v.Info.Chat, client.BuildReaction(v.Info.Chat, v.Info.Sender, v.Info.ID, "📜"))
			sendMenuWithImage(v.Info.Chat)
		}
	}
}

func main() {
	fmt.Println("🚀 [Impossible Bot] Booting Up...")

	dbURL := os.Getenv("DATABASE_URL")
	dbType := "postgres"
	if dbURL == "" {
		dbURL = "file:impossible.db?_foreign_keys=on"
		dbType = "sqlite3"
	}

	var err error
	container, err = sqlstore.New(context.Background(), dbType, dbURL, waLog.Stdout("Database", "INFO", true))
	if err != nil {
		fmt.Printf("❌ [Fatal] DB Connection Failed: %v\n", err)
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil { panic(err) }

	client = whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))
	client.AddEventHandler(eventHandler)

	// اسٹارٹ اپ پر موجودہ سیشن چیک کرنا
	if client.Store.ID != nil {
		fmt.Printf("🔄 [Auth] Found existing session for: %s. Attempting to connect...\n", client.Store.ID.User)
		err := client.Connect()
		if err != nil {
			fmt.Printf("❌ [Auth] Connection Failed for existing session: %v\n", err)
		} else {
			fmt.Println("✅ [Auth] Existing session connected successfully!")
		}
	} else {
		fmt.Println("ℹ️ [Auth] No active session found. Waiting for web pairing...")
	}

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.StaticFile("/", "./web/index.html")
	r.StaticFile("/pic.png", "./web/pic.png")

	r.POST("/api/pair", handlePairAPI)

	go func() {
		fmt.Printf("🌐 [Web] Dashboard live on port %s\n", port)
		r.Run(":" + port)
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	client.Disconnect()
}

func sendMenuWithImage(chat types.JID) {
	fmt.Println("🖼️ [Menu] Preparing image menu...")
	imgData, err := os.ReadFile("./web/pic.png")
	if err != nil {
		fmt.Printf("❌ [Menu] Error reading pic.png: %v\n", err)
		client.SendMessage(context.Background(), chat, &waProto.Message{Conversation: proto.String("*IMPOSSIBLE MENU*\n(Image File Error)")})
		return
	}

	// میڈیا اپلوڈ
	uploadResp, err := client.Upload(context.Background(), imgData, whatsmeow.MediaImage)
	if err != nil {
		fmt.Printf("❌ [Menu] Media upload failed: %v\n", err)
		return
	}

	listMsg := &waProto.ListMessage{
		Title:       proto.String("IMPOSSIBLE BOT"),
		Description: proto.String("Select a command below:"),
		ButtonText:  proto.String("MENU"),
		ListType:    waProto.ListMessage_SINGLE_SELECT.Enum(),
		Sections: []*waProto.ListMessage_Section{
			{
				Title: proto.String("COMMANDS"),
				Rows: []*waProto.ListMessage_Row{
					{Title: proto.String("Ping"), RowID: proto.String("ping")},
					{Title: proto.String("ID Info"), RowID: proto.String("id")},
				},
			},
		},
	}

	imageMsg := &waProto.ImageMessage{
		Mimetype:      proto.String("image/png"),
		Caption:       proto.String("*📜 IMPOSSIBLE MENU*\n\nPowered by Go Engine"),
		URL:           &uploadResp.URL,
		DirectPath:    &uploadResp.DirectPath,
		MediaKey:      uploadResp.MediaKey,
		FileEncSHA256: uploadResp.FileEncSHA256,
		FileSHA256:    uploadResp.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(imgData))),
	}

	msg := &waProto.Message{
		ImageMessage: imageMsg,
		ListMessage:  listMsg,
	}

	resp, sendErr := client.SendMessage(context.Background(), chat, msg)
	if sendErr != nil {
		fmt.Printf("❌ [Menu] Send delivery failed: %v\n", sendErr)
	} else {
		fmt.Printf("✅ [Menu] Successfully delivered! ID: %s\n", resp.ID)
	}
}

func handlePairAPI(c *gin.Context) {
	var req struct{ Number string `json:"number"` }
	c.BindJSON(&req)
	cleanNum := strings.ReplaceAll(req.Number, "+", "")
	
	fmt.Printf("🧹 [Security] Cleaning old records for: %s\n", cleanNum)

	devices, _ := container.GetAllDevices(context.Background())
	for _, dev := range devices {
		if dev.ID != nil && strings.Contains(dev.ID.User, cleanNum) {
			container.DeleteDevice(context.Background(), dev)
			fmt.Printf("🗑️ [Database] Deleted session for %s\n", cleanNum)
		}
	}

	newDevice := container.NewDevice()
	if client.IsConnected() { client.Disconnect() }
	
	// نیا کلائنٹ بنا کر ایونٹ ہینڈلر دوبارہ اٹیچ کرنا (بہت ضروری)
	client = whatsmeow.NewClient(newDevice, waLog.Stdout("Client", "INFO", true))
	client.AddEventHandler(eventHandler) 
	
	client.Connect()
	time.Sleep(10 * time.Second)

	fmt.Println("🔑 [Auth] Requesting pairing code...")
	code, err := client.PairPhone(context.Background(), cleanNum, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		fmt.Printf("❌ [Auth] Pairing error: %v\n", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": code})
}