package main

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// سیکورٹی کمانڈز
func HandleAntilink(client *whatsmeow.Client, v *events.Message) {
	startSecuritySetup(client, v, "antilink")
}

func HandleAntipic(client *whatsmeow.Client, v *events.Message) {
	startSecuritySetup(client, v, "antipic")
}

func HandleAntivideo(client *whatsmeow.Client, v *events.Message) {
	startSecuritySetup(client, v, "antivideo")
}

func HandleAntisticker(client *whatsmeow.Client, v *events.Message) {
	startSecuritySetup(client, v, "antisticker")
}

func startSecuritySetup(client *whatsmeow.Client, v *events.Message, secType string) {
	if !v.Info.IsGroup || !isAdmin(client, v.Info.Chat, v.Info.Sender) { 
		return 
	}
	setupMap[v.Info.Sender.String()] = &SetupState{
		Type: secType, 
		Stage: 1, 
		GroupID: v.Info.Chat.String(), 
		User: v.Info.Sender.String(),
	}
	reply(client, v.Info.Chat, fmt.Sprintf("╭━━━〔 %s SETUP (1/2) 〕━━━┈\n┃ 🛡️ *Allow Admin?*\n┃\n┃ Type *Yes* or *No*\n╰━━━━━━━━━━━━━━━━━━┈", strings.ToUpper(secType)))
}

func handleSetupResponse(client *whatsmeow.Client, v *events.Message, state *SetupState) {
	txt := strings.ToLower(getText(v.Message))
	s := getGroupSettings(state.GroupID)

	if state.Stage == 1 {
		if txt == "yes" { 
			s.AntilinkAdmin = true 
		} else if txt == "no" { 
			s.AntilinkAdmin = false 
		} else { 
			return 
		}
		state.Stage = 2
		reply(client, v.Info.Chat, "╭━━━〔 ACTION SETUP (2/2) 〕━━━┈\n┃ ⚡ *Choose Action:*\n┃\n┃ *Delete*\n┃ *Kick*\n┃ *Warn*\n╰━━━━━━━━━━━━━━━━━━┈")
		return
	}

	if state.Stage == 2 {
		if strings.Contains(txt, "kick") { 
			s.AntilinkAction = "kick" 
		} else if strings.Contains(txt, "warn") { 
			s.AntilinkAction = "warn" 
		} else { 
			s.AntilinkAction = "delete" 
		}
		switch state.Type {
		case "antilink": s.Antilink = true
		case "antipic": s.AntiPic = true
		case "antivideo": s.AntiVideo = true
		case "antisticker": s.AntiSticker = true
		}
		saveGroupSettings(s)
		delete(setupMap, state.User)
		reply(client, v.Info.Chat, fmt.Sprintf("╭━━━〔 ✅ %s ENABLED 〕━━━┈\n┃ 👑 Admin Allow: %v\n┃ ⚡ Action: %s\n╰━━━━━━━━━━━━━━━━━━┈", 
			strings.ToUpper(state.Type), s.AntilinkAdmin, strings.ToUpper(s.AntilinkAction)))
	}
}

func checkSecurity(client *whatsmeow.Client, v *events.Message) {
	if !v.Info.IsGroup {
		return
	}
	
	s := getGroupSettings(v.Info.Chat.String())
	if s.Mode == "private" {
		return
	}
	
	// Anti-link check
	if s.Antilink && containsLink(getText(v.Message)) {
		if s.AntilinkAdmin && isAdmin(client, v.Info.Chat, v.Info.Sender) {
			return
		}
		takeSecurityAction(client, v, s.AntilinkAction, "Link detected!")
		return
	}
	
	// Anti-picture check
	if s.AntiPic && v.Message.ImageMessage != nil {
		if s.AntilinkAdmin && isAdmin(client, v.Info.Chat, v.Info.Sender) {
			return
		}
		takeSecurityAction(client, v, "delete", "Image not allowed!")
		return
	}
	
	// Anti-video check
	if s.AntiVideo && v.Message.VideoMessage != nil {
		if s.AntilinkAdmin && isAdmin(client, v.Info.Chat, v.Info.Sender) {
			return
		}
		takeSecurityAction(client, v, "delete", "Video not allowed!")
		return
	}
	
	// Anti-sticker check
	if s.AntiSticker && v.Message.StickerMessage != nil {
		if s.AntilinkAdmin && isAdmin(client, v.Info.Chat, v.Info.Sender) {
			return
		}
		takeSecurityAction(client, v, "delete", "Sticker not allowed!")
		return
	}
}

func containsLink(text string) bool {
	if text == "" {
		return false
	}
	
	text = strings.ToLower(text)
	linkPatterns := []string{
		"http://", "https://", "www.",
		"chat.whatsapp.com/", "t.me/", "youtube.com/",
		"youtu.be/", "instagram.com/", "fb.com/",
		"facebook.com/", "twitter.com/", "x.com/",
	}
	
	for _, pattern := range linkPatterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	
	return false
}

func takeSecurityAction(client *whatsmeow.Client, v *events.Message, action, reason string) {
	switch action {
	case "delete":
		client.DeleteMessage(context.Background(), v.Info.Chat, v.Info.ID)
		reply(client, v.Info.Chat, fmt.Sprintf("🚫 %s (Message deleted)", reason))
		
	case "kick":
		client.UpdateGroupParticipants(context.Background(), v.Info.Chat, 
			[]types.JID{v.Info.Sender}, whatsmeow.ParticipantChangeRemove)
		reply(client, v.Info.Chat, fmt.Sprintf("👢 %s (User kicked)", reason))
		
	case "warn":
		s := getGroupSettings(v.Info.Chat.String())
		senderKey := v.Info.Sender.String()
		
		s.Warnings[senderKey]++
		warnCount := s.Warnings[senderKey]
		
		if warnCount >= 3 {
			client.UpdateGroupParticipants(context.Background(), v.Info.Chat, 
				[]types.JID{v.Info.Sender}, whatsmeow.ParticipantChangeRemove)
			delete(s.Warnings, senderKey)
			reply(client, v.Info.Chat, "🚫 User kicked after 3 warnings!")
		} else {
			reply(client, v.Info.Chat, fmt.Sprintf("⚠️ Warning %d/3: %s", warnCount, reason))
		}
		
		saveGroupSettings(s)
	}
}