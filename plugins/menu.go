package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// مینیو کمانڈز
func HandleMenu(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "📜")
	sendMenu(client, v.Info.Chat)
}

func HandlePing(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "⚡")
	sendPing(client, v.Info.Chat)
}

func HandleID(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "🆔")
	sendID(client, v)
}

func HandleOwner(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "👑")
	sendOwner(client, v.Info.Chat, v.Info.Sender)
}

func sendMenu(client *whatsmeow.Client, chat types.JID) {
	uptime := time.Since(startTime).Round(time.Second)
	dataMutex.RLock()
	p := data.Prefix
	dataMutex.RUnlock()
	
	s := getGroupSettings(chat.String())
	currentMode := strings.ToUpper(s.Mode)
	if !strings.Contains(chat.String(), "@g.us") {
		currentMode = "PRIVATE"
	}
	
	menu := fmt.Sprintf(`╭━━━〔 %s 〕━━━┈
┃ 👋 *Assalam-o-Alaikum*
┃ 👑 *Owner:* %s
┃ 🛡️ *Mode:* %s
┃ ⏳ *Uptime:* %s
┃
┃ ╭━━〔 *DOWNLOADERS* 〕━━┈
┃ ┃ 🔸 *%sfb*
┃ ┃ 🔸 *%sig*
┃ ┃ 🔸 *%spin*
┃ ┃ 🔸 *%stiktok*
┃ ┃ 🔸 *%sytmp3*
┃ ┃ 🔸 *%sytmp4*
┃ ╰━━━━━━━━━━━━━━━━━━┈
┃
┃ ╭━━〔 *GROUP* 〕━━┈
┃ ┃ 🔸 *%sadd*
┃ ┃ 🔸 *%sdemote*
┃ ┃ 🔸 *%sgroup*
┃ ┃ 🔸 *%shidetag*
┃ ┃ 🔸 *%skick*
┃ ┃ 🔸 *%spromote*
┃ ┃ 🔸 *%stagall*
┃ ╰━━━━━━━━━━━━━━━━━━┈
┃
┃ ╭━━〔 *SETTINGS* 〕━━┈
┃ ┃ 🔸 *%saddstatus*
┃ ┃ 🔸 *%salwaysonline*
┃ ┃ 🔸 *%santilink*
┃ ┃ 🔸 *%santipic*
┃ ┃ 🔸 *%santisticker*
┃ ┃ 🔸 *%santivideo*
┃ ┃ 🔸 *%sautoreact*
┃ ┃ 🔸 *%sautoread*
┃ ┃ 🔸 *%sautostatus*
┃ ┃ 🔸 *%sdelstatus*
┃ ┃ 🔸 *%sliststatus*
┃ ┃ 🔸 *%smode*
┃ ┃ 🔸 *%sowner*
┃ ┃ 🔸 *%sreadallstatus*
┃ ┃ 🔸 *%sstatusreact*
┃ ╰━━━━━━━━━━━━━━━━━━┈
┃
┃ ╭━━〔 *TOOLS* 〕━━┈
┃ ┃ 🔸 *%sdata*
┃ ┃ 🔸 *%sid*
┃ ┃ 🔸 *%sping*
┃ ┃ 🔸 *%sremini*
┃ ┃ 🔸 *%sremovebg*
┃ ┃ 🔸 *%ssticker*
┃ ┃ 🔸 *%stoimg*
┃ ┃ 🔸 *%stourl*
┃ ┃ 🔸 *%stovideo*
┃ ┃ 🔸 *%stranslate*
┃ ┃ 🔸 *%svv*
┃ ┃ 🔸 *%sweather*
┃ ╰━━━━━━━━━━━━━━━━━━┈
┃
┃ © 2025 Nothing is Impossible
╰━━━━━━━━━━━━━━━━━━┈`, 
		BOT_NAME, OWNER_NAME, currentMode, uptime,
		p, p, p, p, p, p,
		p, p, p, p, p, p, p,
		p, p, p, p, p, p, p, p, p, p, p, p, p, p, p,
		p, p, p, p, p, p, p, p, p, p, p, p)

	imgData, err := ioutil.ReadFile("pic.png")
	if err != nil {
		imgData, err = ioutil.ReadFile("web/pic.png")
	}

	if err == nil {
		resp, err := client.Upload(context.Background(), imgData, whatsmeow.MediaImage)
		if err == nil {
			client.SendMessage(context.Background(), chat, &waProto.Message{
				ImageMessage: &waProto.ImageMessage{
					Caption:       proto.String(menu),
					URL:           proto.String(resp.URL),
					DirectPath:    proto.String(resp.DirectPath),
					MediaKey:      resp.MediaKey,
					Mimetype:      proto.String("image/png"),
					FileEncSHA256: resp.FileEncSHA256,
					FileSHA256:    resp.FileSHA256,
				},
			})
			return
		}
	}
	
	client.SendMessage(context.Background(), chat, &waProto.Message{
		Conversation: proto.String(menu),
	})
}

func sendPing(client *whatsmeow.Client, chat types.JID) {
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	ms := time.Since(start).Milliseconds()
	uptime := time.Since(startTime).Round(time.Second)

	msg := fmt.Sprintf(`
╔════════════════════════╗
║        Dev    ║    %s      ║
╚══════════════╩═════════╝
               ┌────────────┐                
               │        ✨ PING          │              
               │           %d MS            │                
               └────────────┘                
╔════════════════════════╗
║    ⏱ UPTIME                      %s       ║
╚══════════════╩═════════╝`,
		OWNER_NAME, ms, uptime)

	client.SendMessage(context.Background(), chat, &waProto.Message{
		Conversation: proto.String(msg),
	})
}

func sendID(client *whatsmeow.Client, v *events.Message) {
	user := v.Info.Sender.User
	chat := v.Info.Chat.User
	chatType := "Private"
	if v.Info.IsGroup {
		chatType = "Group"
	}

	msg := fmt.Sprintf(`╭━━━〔 ID INFO 〕━━━┈
┃ 👤 *User:* `+"`%s`"+`
┃ 👥 *Chat:* `+"`%s`"+`
┃ 🏷️ *Type:* %s
╰━━━━━━━━━━━━━━━━━━┈`, user, chat, chatType)

	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		Conversation: proto.String(msg),
	})
}

func sendOwner(client *whatsmeow.Client, chat types.JID, sender types.JID) {
	status := "❌ You are NOT the Owner."
	if isOwner(client, sender) {
		status = "👑 You are the OWNER!"
	}
	
	botNum := cleanNumber(client.Store.ID.User)
	userNum := cleanNumber(sender.User)
	
	reply(client, chat, fmt.Sprintf(`╭━━━〔 OWNER VERIFICATION 〕━━━┈
┃ 🤖 *Bot:* %s
┃ 👤 *You:* %s
┃
┃ %s
╰━━━━━━━━━━━━━━━━━━┈`, botNum, userNum, status))
}