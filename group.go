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

func handleKick(client *whatsmeow.Client, v *events.Message, args []string) {
	groupAction(client, v, args, "remove")
}

func handleAdd(client *whatsmeow.Client, v *events.Message, args []string) {
	if !v.Info.IsGroup {
		msg := `╔════════════════╗
║ ❌ GROUP ONLY
╠════════════════
║ This command
║ works only in
║ group chats
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	if !isAdmin(client, v.Info.Chat, v.Info.Sender) && !isOwner(client, v.Info.Sender) {
		msg := `╔════════════════╗
║ ❌ DENIED
╠════════════════
║ 🔒 Admin Only
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	if len(args) == 0 {
		msg := `╔════════════════╗
║ ⚠️ INVALID
╠════════════════
║ Usage:
║ .add <number>
║
║ Example:
║ .add 92300xxx
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	num := strings.ReplaceAll(args[0], "+", "")
	jid, _ := types.ParseJID(num + "@s.whatsapp.net")
	client.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{jid}, whatsmeow.ParticipantChangeAdd)

	msg := fmt.Sprintf(`╔════════════════╗
║ ✅ ADDED
╠════════════════
║ Number: %s
║ Added to group
╚════════════════`, args[0])

	replyMessage(client, v, msg)
}

func handlePromote(client *whatsmeow.Client, v *events.Message, args []string) {
	groupAction(client, v, args, "promote")
}

func handleDemote(client *whatsmeow.Client, v *events.Message, args []string) {
	groupAction(client, v, args, "demote")
}

func handleTagAll(client *whatsmeow.Client, v *events.Message, args []string) {
	if !v.Info.IsGroup {
		msg := `╔════════════════╗
║ ❌ GROUP ONLY
╠════════════════
║ This command
║ works only in
║ group chats
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	if !isAdmin(client, v.Info.Chat, v.Info.Sender) && !isOwner(client, v.Info.Sender) {
		msg := `╔════════════════╗
║ ❌ DENIED
╠════════════════
║ 🔒 Admin Only
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	info, _ := client.GetGroupInfo(context.Background(), v.Info.Chat)
	mentions := []string{}
	out := "╔════════════════╗\n"
	out += "║ 📣 TAG ALL\n"
	out += "╠════════════════\n"

	if len(args) > 0 {
		out += "║ 💬 " + strings.Join(args, " ") + "\n"
	}

	for _, p := range info.Participants {
		mentions = append(mentions, p.JID.String())
		out += "║ @" + p.JID.User + "\n"
	}

	out += fmt.Sprintf("║ 👥 Total: %d\n", len(info.Participants))
	out += "╚════════════════"

	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(out),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: mentions,
				StanzaID:     proto.String(v.Info.ID),
				Participant:  proto.String(v.Info.Sender.String()),
			},
		},
	})
}

func handleHideTag(client *whatsmeow.Client, v *events.Message, args []string) {
	if !v.Info.IsGroup {
		msg := `╔════════════════╗
║ ❌ GROUP ONLY
╠════════════════
║ This command
║ works only in
║ group chats
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	if !isAdmin(client, v.Info.Chat, v.Info.Sender) && !isOwner(client, v.Info.Sender) {
		msg := `╔════════════════╗
║ ❌ DENIED
╠════════════════
║ 🔒 Admin Only
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	info, _ := client.GetGroupInfo(context.Background(), v.Info.Chat)
	mentions := []string{}
	text := strings.Join(args, " ")

	if text == "" {
		text = "🔔 Hidden Tag"
	}

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

func handleGroup(client *whatsmeow.Client, v *events.Message, args []string) {
	if !v.Info.IsGroup {
		msg := `╔════════════════╗
║ ❌ GROUP ONLY
╠════════════════
║ This command
║ works only in
║ group chats
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	if !isAdmin(client, v.Info.Chat, v.Info.Sender) && !isOwner(client, v.Info.Sender) {
		msg := `╔════════════════╗
║ ❌ DENIED
╠════════════════
║ 🔒 Admin Only
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	if len(args) == 0 {
		msg := `╔════════════════╗
║ ⚙️ SETTINGS
╠════════════════
║ Commands:
║
║ 🔒 .group close
║    Close group
║
║ 🔓 .group open
║    Open group
║
║ 🔗 .group link
║    Get link
║
║ 🔄 .group revoke
║    Revoke link
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	switch strings.ToLower(args[0]) {
	case "close":
		client.SetGroupAnnounce(context.Background(), v.Info.Chat, true)
		msg := `╔════════════════╗
║ 🔒 CLOSED
╠════════════════
║ Only admins
║ can send now
╚════════════════`
		replyMessage(client, v, msg)

	case "open":
		client.SetGroupAnnounce(context.Background(), v.Info.Chat, false)
		msg := `╔════════════════╗
║ 🔓 OPENED
╠════════════════
║ All members
║ can send now
╚════════════════`
		replyMessage(client, v, msg)

	case "link":
		code, _ := client.GetGroupInviteLink(context.Background(), v.Info.Chat, false)
		msg := fmt.Sprintf(`╔════════════════╗
║ 🔗 LINK
╠════════════════
║ Group Link 🖇️ 
║ %s
╚════════════════`, code)
		replyMessage(client, v, msg)

	case "revoke":
		client.GetGroupInviteLink(context.Background(), v.Info.Chat, true)
		msg := `╔════════════════╗
║ 🔄 REVOKED
╠════════════════
║ Old link is
║ now invalid
║ Use .group link
║ for new one
╚════════════════`
		replyMessage(client, v, msg)

	default:
		msg := `╔════════════════╗
║ ❌ INVALID
╠════════════════
║ Use: close,
║ open, link, or
║ revoke
╚════════════════`
		replyMessage(client, v, msg)
	}
}

func handleDelete(client *whatsmeow.Client, v *events.Message) {
	if !v.Info.IsGroup {
		return
	}

	if !isAdmin(client, v.Info.Chat, v.Info.Sender) && !isOwner(client, v.Info.Sender) {
		msg := `╔════════════════╗
║ ❌ DENIED
╠════════════════
║ 🔒 Admin Only
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	if v.Message.ExtendedTextMessage == nil {
		msg := `╔════════════════╗
║ ⚠️ INVALID
╠════════════════
║ Reply to a
║ message to
║ delete it
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	ctx := v.Message.ExtendedTextMessage.ContextInfo
	if ctx == nil || ctx.StanzaID == nil {
		return
	}

	client.RevokeMessage(context.Background(), v.Info.Chat, *ctx.StanzaID)

	msg := `╔════════════════╗
║ 🗑️ DELETED
╠════════════════
║ ✅ Removed
╚════════════════`
	replyMessage(client, v, msg)
}

func groupAction(client *whatsmeow.Client, v *events.Message, args []string, action string) {
	if !v.Info.IsGroup {
		msg := `╔════════════════╗
║ ❌ GROUP ONLY
╠════════════════
║ This command
║ works only in
║ group chats
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	if !isAdmin(client, v.Info.Chat, v.Info.Sender) && !isOwner(client, v.Info.Sender) {
		msg := `╔════════════════╗
║ ❌ DENIED
╠════════════════
║ 🔒 Admin Only
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	var targetJID types.JID
	if len(args) > 0 {
		num := strings.TrimSpace(args[0])
		num = strings.ReplaceAll(num, "+", "")
		if !strings.Contains(num, "@") {
			num = num + "@s.whatsapp.net"
		}
		jid, err := types.ParseJID(num)
		if err != nil {
			msg := `╔════════════════╗
║ ❌ INVALID
╠════════════════
║ Invalid number
╚════════════════`
			replyMessage(client, v, msg)
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
		msg := `╔════════════════╗
║ ⚠️ NO USER
╠════════════════
║ Mention or
║ reply to user
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	if targetJID.User == v.Info.Sender.User && action == "remove" {
		msg := `╔════════════════╗
║ ❌ INVALID
╠════════════════
║ Cannot kick
║ yourself
╚════════════════`
		replyMessage(client, v, msg)
		return
	}

	var actionText, actionEmoji string
	var participantChange whatsmeow.ParticipantChange

	switch action {
	case "remove":
		participantChange = whatsmeow.ParticipantChangeRemove
		actionText = "Kicked"
		actionEmoji = "👢"
	case "promote":
		participantChange = whatsmeow.ParticipantChangePromote
		actionText = "Promoted"
		actionEmoji = "⬆️"
	case "demote":
		participantChange = whatsmeow.ParticipantChangeDemote
		actionText = "Demoted"
		actionEmoji = "⬇️"
	}

	client.UpdateGroupParticipants(context.Background(), v.Info.Chat, []types.JID{targetJID}, participantChange)

	msg := fmt.Sprintf(`╔════════════════╗
║ %s %s
╠════════════════
║ User: @%s
║ ✅ Done
╚════════════════`, actionEmoji, strings.ToUpper(actionText), targetJID.User)

	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(msg),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: []string{targetJID.String()},
				StanzaID:     proto.String(v.Info.ID),
				Participant:  proto.String(v.Info.Sender.String()),
			},
		},
	})
}