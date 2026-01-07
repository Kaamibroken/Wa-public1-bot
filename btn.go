package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// 🎛️ MAIN SWITCH HANDLER
func HandleButtonCommands(client *whatsmeow.Client, evt *events.Message) {
	text := evt.Message.GetConversation()
	if text == "" {
		text = evt.Message.GetExtendedTextMessage().GetText()
	}

	if !strings.HasPrefix(strings.ToLower(text), ".btn") {
		return
	}

	cmd := strings.TrimSpace(strings.ToLower(text))

	// لمبی تحریر (Professional Body Text)
	longBody := "السلام علیکم! 👋\n\n" +
		"یہ آپ کا *ویریفکیشن کوڈ* ہے۔ برائے مہربانی اسے کسی کے ساتھ شیئر نہ کریں۔\n\n" +
		"📌 *ہدایات:* \n" +
		"1. نیچے دیئے گئے بٹن پر کلک کریں۔\n" +
		"2. کوڈ خود بخود کاپی ہو جائے گا۔\n" +
		"3. ایپ میں جا کر پیسٹ کریں۔\n\n" +
		"⚠️ *نوٹ:* یہ کوڈ اگلے 10 منٹ تک کارآمد ہے۔"

	switch cmd {
	case ".btn 1":
		fmt.Println("🚀 sending Copy Button with Long Text...")
		params := map[string]string{
			"display_text": "کاپی کوڈ (Copy Code)",
			"copy_code":    "IMPOSSIBLE-2026",
			"id":           "btn_copy_123",
		}
		sendNativeFlow(client, evt, "🔐 *IMPOSSIBLE SECURITY*", longBody, "cta_copy", params)

	case ".btn 2":
		fmt.Println("🚀 sending URL Button with Long Text...")
		params := map[string]string{
			"display_text": "ویب سائٹ کھولیں",
			"url":          "https://google.com",
			"merchant_url": "https://google.com",
			"id":           "btn_url_456",
		}
		urlBody := "🌍 *دنیا کو دریافت کریں*\n\n" +
			"ہماری نئی ویب سائٹ لانچ ہو چکی ہے! بہترین تجربے کے لیے ابھی وزٹ کریں۔\n" +
			"نیچے دیئے گئے بٹن پر کلک کر کے براہ راست گوگل کھولیں۔"
		
		sendNativeFlow(client, evt, "🌐 *OFFICIAL LINK*", urlBody, "cta_url", params)

	case ".btn 3":
		fmt.Println("🚀 sending List Menu...")
		listParams := map[string]interface{}{
			"title": "✨ مینو کھولیں",
			"sections": []map[string]interface{}{
				{
					"title": "Main Features",
					"rows": []map[string]string{
						{"header": "🤖", "title": "AI Chat", "description": "Ask Gemini Anything", "id": "row_ai"},
						{"header": "📥", "title": "Downloader", "description": "Save TikTok/Insta", "id": "row_dl"},
					},
				},
				{
					"title": "Admin Tools",
					"rows": []map[string]string{
						{"header": "⚙️", "title": "Control Panel", "description": "Manage Bot Settings", "id": "row_panel"},
					},
				},
			},
		}
		listBody := "📂 *مین مینو (Main Menu)*\n\n" +
			"براہ کرم اپنی پسندیدہ سروس کا انتخاب کریں۔\n" +
			"ہمارا سسٹم 24/7 آن لائن ہے۔"
			
		sendNativeFlow(client, evt, "🤖 *IMPOSSIBLE BOT*", listBody, "single_select", listParams)
	}
}

// ---------------------------------------------------------
// 👇 HELPER FUNCTION (HEAVY MESSAGE STRUCTURE)
// ---------------------------------------------------------

func sendNativeFlow(client *whatsmeow.Client, evt *events.Message, title string, body string, btnName string, params interface{}) {
	
	jsonBytes, err := json.Marshal(params)
	if err != nil {
		fmt.Printf("❌ JSON Error: %v\n", err)
		return
	}

	buttons := []*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
		{
			Name:             proto.String(btnName),
			ButtonParamsJSON: proto.String(string(jsonBytes)),
		},
	}

	msg := &waE2E.Message{
		ViewOnceMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				InteractiveMessage: &waE2E.InteractiveMessage{
					// 🔥 HEADER (BOLD TITLE)
					Header: &waE2E.InteractiveMessage_Header{
						Title:              proto.String(title),
						Subtitle:           proto.String("Authorized Service"), // Extra Validation
						HasMediaAttachment: proto.Bool(false),
					},
					// 🔥 BODY (LONG TEXT)
					Body: &waE2E.InteractiveMessage_Body{
						Text: proto.String(body),
					},
					// 🔥 FOOTER (LIGHT TEXT)
					Footer: &waE2E.InteractiveMessage_Footer{
						Text: proto.String("Powered by Impossible Bot ⚡"),
					},
					
					InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
						NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
							Buttons:           buttons,
							MessageParamsJSON: proto.String(""), // Empty string is key!
							MessageVersion:    proto.Int32(1),
						},
					},

					// 🔥 CONTEXT INFO (The Reply Trick)
					ContextInfo: &waE2E.ContextInfo{
						StanzaID:      proto.String(evt.Info.ID),
						Participant:   proto.String(evt.Info.Sender.String()),
						QuotedMessage: evt.Message,
						IsForwarded:   proto.Bool(true),
					},
				},
			},
		},
	}

	// Send & Log
	resp, err := client.SendMessage(context.Background(), evt.Info.Chat, msg)
	if err != nil {
		fmt.Printf("❌ Error sending: %v\n", err)
	} else {
		fmt.Printf("✅ Sent with Long Text! ID: %s\n", resp.ID)
	}
}
