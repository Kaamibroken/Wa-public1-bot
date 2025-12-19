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
	fmt.Println("🚀 [Impossible Bot] Initializing Final Stable Engine...")

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

	// کلائنٹ بنانا
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
			c.JSON(400, gin.H{"error": "Invalid input"})
			return
		}

		cleanNum := strings.ReplaceAll(req.Number, "+", "")
		fmt.Printf("🧹 [Security] Cleaning old sessions for: %s\n", cleanNum)

		// سیشن کلین اپ لاجک
		devices, _ := container.GetAllDevices(context.Background())
		for _, dev := range devices {
			if dev.ID != nil && strings.Contains(dev.ID.User, cleanNum) {
				container.DeleteDevice(context.Background(), dev)
				fmt.Printf("🗑️ [Cleanup] Deleted existing session for %s\n", cleanNum)
			}
		}

		// فکسڈ: NewDevice اب بغیر کسی آرگیومنٹ کے کال ہو رہا ہے
		newDevice := container.NewDevice() 
		
		// فکسڈ: SetDevice کی جگہ نیا کلائنٹ انسٹنس بنانا
		if client.IsConnected() { client.Disconnect() }
		client = whatsmeow.NewClient(newDevice, waLog.Stdout("Client", "INFO", true))
		client.AddEventHandler(eventHandler)

		err = client.Connect()
		if err != nil {
			c.JSON(500, gin.H{"error": "Connection failed"})
			return
		}

		// واٹس ایپ نیٹ ورک کے مستحکم ہونے کا انتظار
		time.Sleep(10 * time.Second)

		fmt.Println("🔑 [Auth] Generating pairing code...")
		code, err := client.PairPhone(context.Background(), cleanNum, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
		
		if err != nil {
			fmt.Printf("❌ [Error] %v\n", err)
			c.JSON(500, gin.H{"error": "WhatsApp server timeout. Try again."})
			return
		}

		c.JSON(200, gin.H{"code": code})
	})

	go r.Run(":" + port)

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
		Title:       proto.String("IMPOSSIBLE MENU"),
		Description: proto.String("Advanced Go System"),
		ButtonText:  proto.String("MENU"),
		ListType:    waProto.ListMessage_SINGLE_SELECT.Enum(),
		Sections: []*waProto.ListMessage_Section{
			{
				Title: proto.String("COMMANDS"),
				Rows: []*waProto.ListMessage_Row{
					{Title: proto.String("Ping"), RowID: proto.String("ping")},
				},
			},
		},
	}
	client.SendMessage(context.Background(), chat, &waProto.Message{ListMessage: listMsg})
}