package main

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// گروپ کمانڈز
func HandleKick(client *whatsmeow.Client, v *events.Message, args []string) {
	groupAction(client, v, args, "remove")
}

func HandleAdd(client *whatsmeow.Client, v *events.Message, args []string) {
	if !v.Info.IsGroup || len(args) == 0 { 
		return 
	}
	jid, _ := types.ParseJID(args[0] + "@s.whatsapp.net")
	client.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{jid}, whatsmeow.ParticipantChangeAdd)
	reply(client, v.Info.Chat, fmt.Sprintf("✅ Added: %s", args[0]))
}

func HandlePromote(client *whatsmeow.Client, v *events.Message, args []string) {
	groupAction(client, v, args, "promote")
}

func HandleDemote(client *whatsmeow.Client, v *events.Message, args []string) {
	groupAction(client, v, args, "demote")
}

func HandleTagAll(client *whatsmeow.Client, v *events.Message, args []string) {
	if !v.Info.IsGroup { 
		return 
	}
	info, _ := client.GetGroupInfo(context.Background(), v.Info.Chat)
	mentions := []string{}
	out := "📣 *TAG ALL*\n"
	
	if len(args) > 0 {
		out += strings.Join(args, " ") + "\n\n"
	}
	
	for _, p := range info.Participants {
		mentions = append(mentions, p.JID.String())
		out += "@" + p.JID.User + "\n"
	}
	
	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(out),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: mentions,
			},
		},
	})
}

func HandleHideTag(client *whatsmeow.Client, v *events.Message, args []string) {
	if !v.Info.IsGroup { 
		return 
	}
	info, _ := client.GetGroupInfo(context.Background(), v.Info.Chat)
	mentions := []string{}
	text := strings.Join(args, " ")
	
	for _, p := range info.Participants {
		mentions = append(mentions, p.JID.String())
	}
	
	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: mentions,
			},
		},
	})
}

func HandleGroup(client *whatsmeow.Client, v *events.Message, args []string) {
	if !v.Info.IsGroup || len(args) == 0 { 
		return 
	}
	switch args[0] {
	case "close": 
		client.SetGroupAnnounce(context.Background(), v.Info.Chat, true)
		reply(client, v.Info.Chat, "🔒 Group Closed")
	case "open": 
		client.SetGroupAnnounce(context.Background(), v.Info.Chat, false)
		reply(client, v.Info.Chat, "🔓 Group Opened")
	case "link":
		code, _ := client.GetGroupInviteLink(context.Background(), v.Info.Chat, false)
		reply(client, v.Info.Chat, "🔗 https://chat.whatsapp.com/"+code)
	case "revoke":
		client.GetGroupInviteLink(context.Background(), v.Info.Chat, true)
		reply(client, v.Info.Chat, "🔄 Link Revoked")
	}
}

func HandleDelete(client *whatsmeow.Client, v *events.Message) {
	if v.Message.ExtendedTextMessage == nil { 
		return 
	}
	ctx := v.Message.ExtendedTextMessage.ContextInfo
	if ctx == nil { 
		return 
	}
	client.RevokeMessage(context.Background(), v.Info.Chat, *ctx.StanzaID)
}

func groupAction(client *whatsmeow.Client, v *events.Message, args []string, action string) {
	if !v.Info.IsGroup { 
		return 
	}
	
	var targetJID types.JID
	if len(args) > 0 {
		num := strings.TrimSpace(args[0])
		if !strings.Contains(num, "@") {
			num = num + "@s.whatsapp.net"
		}
		jid, err := types.ParseJID(num)
		if err != nil {
			reply(client, v.Info.Chat, "❌ Invalid number")
			return
		}
		targetJID = jid
	} else if v.Message.ExtendedTextMessage != nil && v.Message.ExtendedTextMessage.ContextInfo != nil {
		ctx := v.Message.ExtendedTextMessage.ContextInfo
		if ctx.Participant != nil {
			jid, _ := types.ParseJID(*ctx.Participant)
			targetJID = jid
		} else if len(ctx.MentionedJID) > 0 {
			jid, _ := types.ParseJID(ctx.MentionedJID[0])
			targetJID = jid
		}
	}
	
	if targetJID.User == "" {
		reply(client, v.Info.Chat, "⚠️ Mention or reply to user")
		return
	}
	
	// خود کو نہ نکالے
	if targetJID.User == v.Info.Sender.User && action == "remove" {
		reply(client, v.Info.Chat, "❌ Can't kick yourself")
		return
	}
	
	var actionText string
	switch action {
	case "remove":
		client.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{targetJID}, whatsmeow.ParticipantChangeRemove)
		actionText = "Kicked"
	case "promote":
		client.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{targetJID}, whatsmeow.ParticipantChangePromote)
		actionText = "Promoted"
	case "demote":
		client.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{targetJID}, whatsmeow.ParticipantChangeDemote)
		actionText = "Demoted"
	}
	
	reply(client, v.Info.Chat, fmt.Sprintf("✅ %s: %s", actionText, targetJID.User))
}