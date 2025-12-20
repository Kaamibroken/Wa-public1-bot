package main

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// سیٹنگز کمانڈز
func HandleAlwaysOnline(client *whatsmeow.Client, v *events.Message) {
	if !isOwner(client, v.Info.Sender) { 
		reply(client, v.Info.Chat, "❌ Owner Only")
		return 
	}
	
	status := "OFF 🔴"
	dataMutex.Lock()
	data.AlwaysOnline = !data.AlwaysOnline
	if data.AlwaysOnline { 
		client.SendPresence(context.Background(), types.PresenceAvailable)
		status = "ON 🟢" 
	} else { 
		client.SendPresence(context.Background(), types.PresenceUnavailable)
	}
	dataMutex.Unlock()
	
	reply(client, v.Info.Chat, fmt.Sprintf("⚙️ *ALWAYSONLINE:* %s", status))
}

func HandleAutoRead(client *whatsmeow.Client, v *events.Message) {
	if !isOwner(client, v.Info.Sender) { 
		reply(client, v.Info.Chat, "❌ Owner Only")
		return 
	}
	
	status := "OFF 🔴"
	dataMutex.Lock()
	data.AutoRead = !data.AutoRead
	if data.AutoRead { status = "ON 🟢" }
	dataMutex.Unlock()
	
	reply(client, v.Info.Chat, fmt.Sprintf("⚙️ *AUTOREAD:* %s", status))
}

func HandleAutoReact(client *whatsmeow.Client, v *events.Message) {
	if !isOwner(client, v.Info.Sender) { 
		reply(client, v.Info.Chat, "❌ Owner Only")
		return 
	}
	
	status := "OFF 🔴"
	dataMutex.Lock()
	data.AutoReact = !data.AutoReact
	if data.AutoReact { status = "ON 🟢" }
	dataMutex.Unlock()
	
	reply(client, v.Info.Chat, fmt.Sprintf("⚙️ *AUTOREACT:* %s", status))
}

func HandleAutoStatus(client *whatsmeow.Client, v *events.Message) {
	if !isOwner(client, v.Info.Sender) { 
		reply(client, v.Info.Chat, "❌ Owner Only")
		return 
	}
	
	status := "OFF 🔴"
	dataMutex.Lock()
	data.AutoStatus = !data.AutoStatus
	if data.AutoStatus { status = "ON 🟢" }
	dataMutex.Unlock()
	
	reply(client, v.Info.Chat, fmt.Sprintf("⚙️ *AUTOSTATUS:* %s", status))
}

func HandleStatusReact(client *whatsmeow.Client, v *events.Message) {
	if !isOwner(client, v.Info.Sender) { 
		reply(client, v.Info.Chat, "❌ Owner Only")
		return 
	}
	
	status := "OFF 🔴"
	dataMutex.Lock()
	data.StatusReact = !data.StatusReact
	if data.StatusReact { status = "ON 🟢" }
	dataMutex.Unlock()
	
	reply(client, v.Info.Chat, fmt.Sprintf("⚙️ *STATUSREACT:* %s", status))
}

func HandleAddStatus(client *whatsmeow.Client, v *events.Message, args []string) {
	if !isOwner(client, v.Info.Sender) { 
		reply(client, v.Info.Chat, "❌ Owner Only")
		return 
	}
	
	if len(args) < 1 { 
		reply(client, v.Info.Chat, "⚠️ Number?")
		return 
	}
	
	num := args[0]
	dataMutex.Lock()
	data.StatusTargets = append(data.StatusTargets, num)
	dataMutex.Unlock()
	
	reply(client, v.Info.Chat, "✅ Added to status targets")
}

func HandleDelStatus(client *whatsmeow.Client, v *events.Message, args []string) {
	if !isOwner(client, v.Info.Sender) { 
		reply(client, v.Info.Chat, "❌ Owner Only")
		return 
	}
	
	if len(args) < 1 { 
		reply(client, v.Info.Chat, "⚠️ Number?")
		return 
	}
	
	num := args[0]
	dataMutex.Lock()
	newList := []string{}
	for _, n := range data.StatusTargets { 
		if n != num { 
			newList = append(newList, n) 
		} 
	}
	data.StatusTargets = newList
	dataMutex.Unlock()
	
	reply(client, v.Info.Chat, "🗑️ Removed from status targets")
}

func HandleListStatus(client *whatsmeow.Client, v *events.Message) {
	if !isOwner(client, v.Info.Sender) { 
		return 
	}
	
	dataMutex.RLock()
	targets := data.StatusTargets
	dataMutex.RUnlock()
	
	if len(targets) == 0 {
		reply(client, v.Info.Chat, "📭 No status targets")
		return
	}
	
	msg := "📜 *Status Targets:*\n"
	for i, t := range targets {
		msg += fmt.Sprintf("%d. %s\n", i+1, t)
	}
	
	reply(client, v.Info.Chat, msg)
}

func HandleSetPrefix(client *whatsmeow.Client, v *events.Message, args []string) {
	if !isOwner(client, v.Info.Sender) { 
		reply(client, v.Info.Chat, "❌ Owner Only")
		return 
	}
	
	if len(args) < 1 { 
		reply(client, v.Info.Chat, "⚠️ Prefix?")
		return 
	}
	
	newPrefix := args[0]
	dataMutex.Lock()
	data.Prefix = newPrefix
	dataMutex.Unlock()
	
	reply(client, v.Info.Chat, fmt.Sprintf("╭━━━〔 SETTINGS 〕━━━┈\n┃ ✅ Prefix updated: %s\n╰━━━━━━━━━━━━━━━━━━┈", newPrefix))
}

func HandleMode(client *whatsmeow.Client, v *events.Message, args []string) {
	if !v.Info.IsGroup {
		reply(client, v.Info.Chat, "❌ Group only command")
		return
	}
	
	if !isAdmin(client, v.Info.Chat, v.Info.Sender) && !isOwner(client, v.Info.Sender) {
		reply(client, v.Info.Chat, "❌ Admin only")
		return
	}
	
	if len(args) < 1 {
		reply(client, v.Info.Chat, "⚠️ Mode? (public/private/admin)")
		return
	}
	
	mode := strings.ToLower(args[0])
	if mode != "public" && mode != "private" && mode != "admin" {
		reply(client, v.Info.Chat, "❌ Invalid mode. Use: public/private/admin")
		return
	}
	
	s := getGroupSettings(v.Info.Chat.String())
	s.Mode = mode
	saveGroupSettings(s)
	
	reply(client, v.Info.Chat, fmt.Sprintf("╭━━━〔 MODE CHANGED 〕━━━┈\n┃ 🔒 Mode: %s\n╰━━━━━━━━━━━━━━━━━━┈", strings.ToUpper(mode)))
}

func HandleReadAllStatus(client *whatsmeow.Client, v *events.Message) {
	if !isOwner(client, v.Info.Sender) {
		return
	}
	
	client.MarkRead(context.Background(), []types.MessageID{v.Info.ID}, time.Now(), types.NewJID("status@broadcast", types.DefaultUserServer), v.Info.Sender, types.ReceiptTypeRead)
	reply(client, v.Info.Chat, "✅ Recent statuses marked as read")
}