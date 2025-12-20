package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ڈاؤن لوڈر کمانڈز
func HandleTikTok(client *whatsmeow.Client, v *events.Message, url string) {
	react(client, v.Info.Chat, v.Info.ID, "🎵")
	type R struct { Data struct { Play string `json:"play"` } `json:"data"` }
	var r R
	getJson("https://www.tikwm.com/api/?url="+url, &r)
	if r.Data.Play != "" { 
		sendVideo(client, v.Info.Chat, r.Data.Play, "🎵 TikTok Video") 
	} else {
		reply(client, v.Info.Chat, "❌ Failed to download TikTok")
	}
}

func HandleFacebook(client *whatsmeow.Client, v *events.Message, url string) {
	react(client, v.Info.Chat, v.Info.ID, "📘")
	type R struct { BK9 struct { HD string `json:"HD"` } `json:"BK9"`; Status bool `json:"status"` }
	var r R
	getJson("https://bk9.fun/downloader/facebook?url="+url, &r)
	if r.Status { 
		sendVideo(client, v.Info.Chat, r.BK9.HD, "📘 Facebook Video") 
	} else {
		reply(client, v.Info.Chat, "❌ Failed to download Facebook video")
	}
}

func HandleInstagram(client *whatsmeow.Client, v *events.Message, url string) {
	react(client, v.Info.Chat, v.Info.ID, "📸")
	type R struct { Video struct { Url string `json:"url"` } `json:"video"` }
	var r R
	getJson("https://api.tiklydown.eu.org/api/download?url="+url, &r)
	if r.Video.Url != "" { 
		sendVideo(client, v.Info.Chat, r.Video.Url, "📸 Instagram Video") 
	} else {
		reply(client, v.Info.Chat, "❌ Failed to download Instagram video")
	}
}

func HandlePinterest(client *whatsmeow.Client, v *events.Message, url string) {
	react(client, v.Info.Chat, v.Info.ID, "📌")
	type R struct { BK9 struct { Url string `json:"url"` } `json:"BK9"`; Status bool `json:"status"` }
	var r R
	getJson("https://bk9.fun/downloader/pinterest?url="+url, &r)
	if r.Status {
		sendImage(client, v.Info.Chat, r.BK9.Url, "📌 Pinterest Image")
	} else {
		reply(client, v.Info.Chat, "❌ Failed to download Pinterest image")
	}
}

func HandleYouTubeMP3(client *whatsmeow.Client, v *events.Message, url string) {
	react(client, v.Info.Chat, v.Info.ID, "📺")
	type R struct { BK9 struct { Mp3 string `json:"mp3"` } `json:"BK9"`; Status bool `json:"status"` }
	var r R
	getJson("https://bk9.fun/downloader/youtube?url="+url, &r)
	if r.Status {
		sendDocument(client, v.Info.Chat, r.BK9.Mp3, "audio.mp3", "audio/mpeg")
	} else {
		reply(client, v.Info.Chat, "❌ Failed to download YouTube audio")
	}
}

func HandleYouTubeMP4(client *whatsmeow.Client, v *events.Message, url string) {
	react(client, v.Info.Chat, v.Info.ID, "📺")
	type R struct { BK9 struct { Mp4 string `json:"mp4"` } `json:"BK9"`; Status bool `json:"status"` }
	var r R
	getJson("https://bk9.fun/downloader/youtube?url="+url, &r)
	if r.Status {
		sendVideo(client, v.Info.Chat, r.BK9.Mp4, "📺 YouTube Video")
	} else {
		reply(client, v.Info.Chat, "❌ Failed to download YouTube video")
	}
}