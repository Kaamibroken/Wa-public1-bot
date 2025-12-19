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

func main() {
	fmt.Println("🚀 [Impossible Bot] Starting Targeted Engine...")

	dbURL := os.Getenv("DATABASE_URL")
	dbType := "postgres"
	if dbURL == "" {
		dbURL = "file:impossible_session.db?_foreign_keys=on"
		dbType = "sqlite3"
	}

	dbLog := waLog.Stdout("Database", "INFO", true)
	var err error
	container, err = sqlstore.New(context.Background(), dbType, dbURL, dbLog)
	if err != nil { panic(err) }

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil { panic(err) }

	client = whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))
	client.AddEventHandler(eventHandler)

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.StaticFile("/", "./web/index.html")
	r.StaticFile("/pic.png", "./web/pic.png")

	r.POST("/api/pair", func(c *gin.Context) {
		var req struct{ Number string `json:"number"` }
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid Input"})
			return
		}

		// نمبر سے فالتو نشانات ختم کرنا
		cleanReqNum := strings.ReplaceAll(req.Number, "+", "")
		fmt.Printf("🔍 [Filter] Searching for existing sessions of: %s\n", cleanReqNum)

		if client.IsConnected() {
			client.Disconnect()
		}

		// --- مخصوص نمبر کی کلیننگ لاجک ---
		devices, _ := container.GetAllDevices(context.Background())
		foundOld := false
		for _, dev := range devices {
			// اگر ڈیوائس کا نمبر (JID) ہمارے مطلوبہ نمبر سے میچ کرے
			if dev.ID != nil && strings.Contains(dev.ID.User, cleanReqNum) {
				fmt.Printf("🗑️ [Cleanup] Found and deleting specific session for: %s\n", dev.ID.User)
				container.DeleteDevice(context.Background(), dev)
				foundOld = true
			}
		}

		if !foundOld {
			fmt.Println("✅ [Database] No existing session found for this number. Safe to proceed.")
		}

		// نیا فریش ڈیوائس اسٹور بنانا
		newDevice := container.NewDevice(context.Background())
		client.SetDevice(newDevice)

		fmt.Println("🌐 [Network] Opening fresh socket...")
		err = client.Connect()
		if err != nil {
			c.JSON(500, gin.H{"error": "WhatsApp connection failed. Try again."})
			return
		}

		// سرور کو مستحکم ہونے کے لیے وقت دیں
		time.Sleep(10 * time.Second)

		fmt.Println("🔑 [Auth] Querying pairing code for fresh session...")
		code, err := client.PairPhone(context.Background(), cleanReqNum, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
		
		if err != nil {
			fmt.Printf("❌ [Server Error] %v\n", err)
			c.JSON(500, gin.H{"error": "WhatsApp server busy. Refresh and try again."})
			return
		}

		fmt.Printf("✅ [Success] Generated Code: %s\n", code)
		c.JSON(200, gin.H{"code": code})
	})

	go func() {
		fmt.Printf("🌐 [Web] Interface active on port %s\n", port)
		r.Run(":" + port)
	}()

	if client.Store.ID != nil {
		client.Connect()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	client.Disconnect()
}

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		body := v.Message.GetConversation()
		if body == "" { body = v.Message.GetExtendedTextMessage().GetText() }
		if strings.TrimSpace(body) == "#menu" {
			sendOfficialMenu(v.Info.Chat)
		}
	}
}

func sendOfficialMenu(chat types.JID) {
	listMsg := &waProto.ListMessage{
		Title:       proto.String("IMPOSSIBLE BOT"),
		Description: proto.String("Advanced Menu System"),
		ButtonText:  proto.String("MENU"),
		ListType:    waProto.ListMessage_SINGLE_SELECT.Enum(),
		Sections: []*waProto.ListMessage_Section{
			{
				Title: proto.String("TOOLS"),
				Rows: []*waProto.ListMessage_Row{
					{Title: proto.String("Ping"), RowID: proto.String("ping")},
				},
			},
		},
	}
	client.SendMessage(context.Background(), chat, &waProto.Message{ListMessage: listMsg})
}