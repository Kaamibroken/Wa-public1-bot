package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// --- 🌐 GLOBAL CONNECTION VARIABLES ---
var (
	container   *sqlstore.Container
	clientMap   = make(map[string]*whatsmeow.Client)
	clientMutex sync.RWMutex
)

// --- 🚀 MAIN START ---
func main() {
	fmt.Println("🚀 IMPOSSIBLE BOT FINAL V4 | STARTING SYSTEM...")

	// 1. Load Data (Defined in commands.go)
	loadData()

	// 2. Database Connection
	dbURL := os.Getenv("DATABASE_URL")
	dbType := "postgres"
	if dbURL == "" {
		dbType = "sqlite3"
		dbURL = "file:impossible_sessions.db?_foreign_keys=on"
	}

	dbLog := waLog.Stdout("DB", "INFO", true)
	var err error
	container, err = sqlstore.New(context.Background(), dbType, dbURL, dbLog)
	if err != nil {
		log.Fatalf("❌ DB Error: %v", err)
	}

	// 3. Restore Sessions
	devices, err := container.GetAllDevices(context.Background())
	if err == nil {
		fmt.Printf("🔄 Restoring %d sessions...\n", len(devices))
		for _, device := range devices {
			go connectClient(device)
		}
	}

	// 4. Web Server
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.LoadHTMLGlob("web/*.html")

	r.GET("/", func(c *gin.Context) {
		clientMutex.RLock()
		count := len(clientMap)
		clientMutex.RUnlock()
		c.JSON(200, gin.H{"status": "Online", "sessions": count})
	})
	
	r.POST("/api/pair", handlePairing)

	go r.Run(":8080")
	fmt.Println("🌐 Server running on :8080")

	// 5. Shutdown Handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("🔻 Shutting down...")
	saveData() // Defined in commands.go
	clientMutex.Lock()
	for _, cli := range clientMap {
		cli.Disconnect()
	}
	clientMutex.Unlock()
}

// --- 🔌 CLIENT CONNECTION ---
func connectClient(device *store.Device) {
	client := whatsmeow.NewClient(device, waLog.Stdout("Client", "INFO", true))
	
	client.AddEventHandler(func(evt interface{}) {
		handler(client, evt)
	})

	if err := client.Connect(); err == nil && client.Store.ID != nil {
		clientMutex.Lock()
		clientMap[client.Store.ID.String()] = client
		clientMutex.Unlock()
		fmt.Printf("✅ Connected: %s\n", client.Store.ID.User)
		
		dataMutex.RLock()
		if data.AlwaysOnline {
			client.SendPresence(context.Background(), types.PresenceAvailable)
		}
		dataMutex.RUnlock()
	}
}

// --- 🔗 PAIRING HANDLER ---
func handlePairing(c *gin.Context) {
	var req struct{ Number string `json:"number"` }
	if c.BindJSON(&req) != nil { return }
	num := strings.ReplaceAll(req.Number, " ", "")
	num = strings.ReplaceAll(num, "+", "")

	device := container.NewDevice()
	client := whatsmeow.NewClient(device, waLog.Stdout("Pairing", "INFO", true))

	if err := client.Connect(); err != nil {
		c.JSON(500, gin.H{"error": "Connection Failed"})
		return
	}

	code, err := client.PairPhone(context.Background(), num, true, whatsmeow.PairClientChrome, "Linux")
	if err != nil {
		client.Disconnect()
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	client.AddEventHandler(func(evt interface{}) {
		handler(client, evt)
	})
	c.JSON(200, gin.H{"code": code})
}