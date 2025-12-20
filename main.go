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
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var client *whatsmeow.Client
var container *sqlstore.Container

// بوٹ کی مخصوص شناخت اور ڈویلپر کا نام
const BOT_TAG = "IMPOSSIBLE_V1"
const DEVELOPER = "Nothing Is Impossible"

func main() {
	fmt.Printf("🚀 [%s] Starting Go Engine...\n", BOT_TAG)

	dbURL := os.Getenv("DATABASE_URL")
	dbType := "postgres"
	if dbURL == "" { dbType = "sqlite3"; dbURL = "file:impossible.db?_foreign_keys=on" }

	container, _ = sqlstore.New(context.Background(), dbType, dbURL, waLog.Stdout("Database", "INFO", true))
	
	// سیشن آئسولیشن لاجک
	var targetDevice *store.Device
	devices, _ := container.GetAllDevices(context.Background())
	for _, dev := range devices {
		if dev.PushName == BOT_TAG {
			targetDevice = dev
			break
		}
	}

	if targetDevice == nil {
		targetDevice = container.NewDevice()
		targetDevice.PushName = BOT_TAG
	}

	client = whatsmeow.NewClient(targetDevice, waLog.Stdout("Client", "INFO", true))
	client.AddEventHandler(eventHandler)

	if client.Store.ID != nil { client.Connect() }

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	r := gin.Default()
	r.StaticFile("/", "./web/index.html")
	r.POST("/api/pair", handlePairAPI)

	go r.Run(":" + port)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	client.Disconnect()
}

func getBody(msg *waProto.Message) string {
	if msg == nil { return "" }
	if msg.Conversation != nil { return msg.GetConversation() }
	if msg.ExtendedTextMessage != nil { return msg.ExtendedTextMessage.GetText() }
	return ""
}

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if v.Info.IsFromMe { return }
		body := strings.TrimSpace(strings.ToLower(getBody(v.Message)))
		
		fmt.Printf("📩 [Message] From: %s | Text: %s\n", v.Info.Sender.User, body)

		// مینیو کمانڈ
		if body == "#menu" {
			_, _ = client.SendMessage(context.Background(), v.Info.Chat, client.BuildReaction(v.Info.Chat, v.Info.Sender, v.Info.ID, "📜"))
			sendImpossibleMenu(v.Info.Chat)
		}

		// پنگ کمانڈ (اسپیڈ ٹیسٹ)
		if body == "#ping" {
			start := time.Now()
			_, _ = client.SendMessage(context.Background(), v.Info.Chat, client.BuildReaction(v.Info.Chat, v.Info.Sender, v.Info.ID, "⚡"))
			latency := time.Since(start)
			
			res := fmt.Sprintf("🚀 *Impossible Speed:* %s\n\n_© Developed by %s_", latency.String(), DEVELOPER)
			client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{Conversation: proto.String(res)})
		}
	}
}

func sendImpossibleMenu(chat types.JID) {
	fmt.Println("📤 [Action] Sending Advanced List Menu...")

	// جدید واٹس ایپ بٹن سٹرکچر
	listMsg := &waProto.ListMessage{
		Title:       proto.String("IMPOSSIBLE MENU"),
		Description: proto.String("Hi! Select an option below to explore bot commands."),
		ButtonText:  proto.String("OPEN TOOLS"),
		ListType:    waProto.ListMessage_SINGLE_SELECT.Enum(),
		Sections: []*waProto.ListMessage_Section{
			{
				Title: proto.String("SYSTEM TOOLS"),
				Rows: []*waProto.ListMessage_Row{
					{Title: proto.String("Ping Status"), RowID: proto.String("ping"), Description: proto.String("Check latency speed")},
					{Title: proto.String("My WhatsApp ID"), RowID: proto.String("id")},
				},
			},
		},
	}

	// بٹن بھیجنے کی کوشش
	_, err := client.SendMessage(context.Background(), chat, &waProto.Message{
		ListMessage: listMsg,
	})

	// اگر بٹن فیل ہو جائیں (Error 479) تو ٹیکسٹ مینیو خودکار طریقے سے جائے گا
	if err != nil {
		fmt.Printf("❌ [Error] Buttons failed. Sending backup text menu.\n")
		backup := fmt.Sprintf("*📜 IMPOSSIBLE MENU*\n\n" +
			"• #ping - Check Latency\n" +
			"• #id - Get User ID\n\n" +
			"_Developed by %s_", DEVELOPER)
		client.SendMessage(context.Background(), chat, &waProto.Message{Conversation: proto.String(backup)})
	}
}

func handlePairAPI(c *gin.Context) {
	var req struct{ Number string `json:"number"` }
	c.BindJSON(&req)
	num := strings.ReplaceAll(req.Number, "+", "")

	// سیشن کلین اپ
	devices, _ := container.GetAllDevices(context.Background())
	for _, dev := range devices {
		if dev.PushName == BOT_TAG {
			container.DeleteDevice(context.Background(), dev)
		}
	}

	newStore := container.NewDevice()
	newStore.PushName = BOT_TAG 

	if client.IsConnected() { client.Disconnect() }
	client = whatsmeow.NewClient(newStore, waLog.Stdout("Client", "INFO", true))
	client.AddEventHandler(eventHandler)
	client.Connect()
	
	time.Sleep(10 * time.Second)
	code, err := client.PairPhone(context.Background(), num, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": code})
}