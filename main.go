package main

import (
	"compress/flate"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings" // اب یہ یو آر ایل چیک کرنے کے لیے استعمال ہو رہا ہے
	"syscall"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var client *whatsmeow.Client

// ریلوے پورٹ مینجمنٹ
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "8080"
	}
	return port
}

// آپ کا فراہم کردہ ڈیٹا فیچر (Node.js سے Go میں کنورٹڈ)
func handleFetch(c *gin.Context) {
	dataType := c.Query("type")
	var targetURL string
	var referer string

	if dataType == "numbers" {
		targetURL = "http://217.182.195.194/ints/agent/res/data_smsnumbers.php?..." 
		referer = "http://217.182.195.194/ints/agent/MySMSNumbers"
	} else if dataType == "sms" {
		targetURL = "http://217.182.195.194/ints/agent/res/data_smscdr.php?..."
		referer = "http://217.182.195.194/ints/agent/SMSCDRStats"
	} else {
		c.JSON(400, gin.H{"error": "Invalid type"})
		return
	}

	req, _ := http.NewRequest("GET", targetURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0...")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Referer", referer)
	req.Header.Set("Cookie", "PHPSESSID=pb3620rtcrklvvrmndf8kmt93n")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(500, gin.H{"error": "Fetch failed"})
		return
	}
	defer resp.Body.Close()

	var reader io.ReadCloser
	// یہاں 'strings' کا استعمال ہو رہا ہے
	encoding := resp.Header.Get("Content-Encoding")
	if strings.Contains(encoding, "gzip") {
		reader, _ = gzip.NewReader(resp.Body)
	} else if strings.Contains(encoding, "deflate") {
		reader = flate.NewReader(resp.Body)
	} else {
		reader = resp.Body
	}
	defer reader.Close()

	body, _ := io.ReadAll(reader)
	c.Data(200, "application/json", body)
}

func main() {
	dbLog := waLog.Stdout("Database", "INFO", true)
	// سیشن ہینڈلنگ (Postgres)
	container, err := sqlstore.New(context.Background(), "postgres", os.Getenv("DATABASE_URL"), dbLog)
	if err != nil { panic(err) }

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil { panic(err) }

	client = whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))
	client.AddEventHandler(eventHandler)

	// ویب سرور سیٹ اپ
	r := gin.Default()
	r.StaticFile("/", "./web/index.html")
	r.StaticFile("/pic.png", "./web/pic.png")
	r.GET("/api/fetch", handleFetch)
	
	r.POST("/api/pair", func(c *gin.Context) {
		var req struct{ Number string `json:"number"` }
		c.BindJSON(&req)
		client.Connect()
		// پیرنگ کوڈ ریکویسٹ
		code, _ := client.PairPhone(context.Background(), req.Number, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
		c.JSON(200, gin.H{"code": code})
	})

	go r.Run(":" + getPort())

	if client.Store.ID != nil {
		client.Connect()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	client.Disconnect()
}

// واٹس ایپ MENU بٹن لاجک
func sendOfficialMenu(chat types.JID) {
	listMsg := &waProto.ListMessage{
		Title:       proto.String("IMPOSSIBLE MENU"),
		Description: proto.String("Select category"),
		ButtonText:  proto.String("MENU"), // بٹن کا نام "MENU"
		ListType:    waProto.ListMessage_SINGLE_SELECT.Enum(),
		Sections: []*waProto.ListMessage_Section{
			{
				Title: proto.String("COMMANDS"),
				Rows: []*waProto.ListMessage_Row{
					{Title: proto.String("Ping"), RowID: proto.String("ping")},
					{Title: proto.String("ID"), RowID: proto.String("id")},
				},
			},
		},
	}
	client.SendMessage(context.Background(), chat, &waProto.Message{ListMessage: listMsg})
}

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if v.Message.GetConversation() == "#menu" {
			sendOfficialMenu(v.Info.Chat)
		}
	}
}