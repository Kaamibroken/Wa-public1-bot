package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// ٹولز کمانڈز
func HandleSticker(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "🎨")
	data, err := downloadMedia(client, v.Message)
	if err != nil { 
		reply(client, v.Info.Chat, "❌ No media found")
		return 
	}
	ioutil.WriteFile("temp.jpg", data, 0644)
	exec.Command("ffmpeg", "-y", "-i", "temp.jpg", "-vcodec", "libwebp", "temp.webp").Run()
	b, _ := ioutil.ReadFile("temp.webp")
	up, _ := client.Upload(context.Background(), b, whatsmeow.MediaImage)
	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
			URL: proto.String(up.URL), 
			DirectPath: proto.String(up.DirectPath), 
			MediaKey: up.MediaKey,
			FileEncSHA256: up.FileEncSHA256, 
			FileSHA256: up.FileSHA256, 
			Mimetype: proto.String("image/webp"),
		}})
	os.Remove("temp.jpg")
	os.Remove("temp.webp")
}

func HandleToImg(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "🖼️")
	data, err := downloadMedia(client, v.Message)
	if err != nil { 
		return 
	}
	ioutil.WriteFile("temp.webp", data, 0644)
	exec.Command("ffmpeg", "-y", "-i", "temp.webp", "temp.png").Run()
	b, _ := ioutil.ReadFile("temp.png")
	up, _ := client.Upload(context.Background(), b, whatsmeow.MediaImage)
	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			URL: proto.String(up.URL), 
			DirectPath: proto.String(up.DirectPath), 
			MediaKey: up.MediaKey,
			FileEncSHA256: up.FileEncSHA256, 
			FileSHA256: up.FileSHA256, 
			Mimetype: proto.String("image/png"),
		}})
	os.Remove("temp.webp")
	os.Remove("temp.png")
}

func HandleToVideo(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "🎥")
	data, err := downloadMedia(client, v.Message)
	if err != nil { 
		return 
	}
	ioutil.WriteFile("temp.webp", data, 0644)
	exec.Command("ffmpeg", "-y", "-i", "temp.webp", "temp.mp4").Run()
	d, _ := ioutil.ReadFile("temp.mp4")
	up, _ := client.Upload(context.Background(), d, whatsmeow.MediaVideo)
	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			URL: proto.String(up.URL), 
			DirectPath: proto.String(up.DirectPath), 
			MediaKey: up.MediaKey,
			FileEncSHA256: up.FileEncSHA256, 
			FileSHA256: up.FileSHA256, 
			Mimetype: proto.String("video/mp4"),
		}})
	os.Remove("temp.webp")
	os.Remove("temp.mp4")
}

func HandleRemoveBG(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "✂️")
	d, _ := downloadMedia(client, v.Message)
	u := uploadToCatbox(d)
	sendImage(client, v.Info.Chat, "https://bk9.fun/tools/removebg?url="+u, "✂️ Background Removed")
}

func HandleRemini(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "✨")
	d, _ := downloadMedia(client, v.Message)
	u := uploadToCatbox(d)
	type R struct{Url string `json:"url"`}
	var r R
	getJson("https://remini.mobilz.pw/enhance?url="+u, &r)
	sendImage(client, v.Info.Chat, r.Url, "✨ Enhanced Image")
}

func HandleToURL(client *whatsmeow.Client, v *events.Message) {
	d, _ := downloadMedia(client, v.Message)
	reply(client, v.Info.Chat, "🔗 "+uploadToCatbox(d))
}

func HandleWeather(client *whatsmeow.Client, v *events.Message, city string) {
	react(client, v.Info.Chat, v.Info.ID, "🌦️")
	r, _ := http.Get("https://wttr.in/"+city+"?format=%C+%t")
	d, _ := ioutil.ReadAll(r.Body)
	reply(client, v.Info.Chat, fmt.Sprintf("🌤️ Weather in %s:\n%s", city, string(d)))
}

func HandleTranslate(client *whatsmeow.Client, v *events.Message, args []string) {
	react(client, v.Info.Chat, v.Info.ID, "🌍")
	t := strings.Join(args, " ")
	if t == "" { 
		q := v.Message.ExtendedTextMessage.GetContextInfo().GetQuotedMessage()
		if q != nil { 
			t = q.GetConversation() 
		}
	}
	r, _ := http.Get(fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=ur&dt=t&q=%s", url.QueryEscape(t)))
	var res []interface{}
	json.NewDecoder(r.Body).Decode(&res)
	if len(res)>0 { 
		reply(client, v.Info.Chat, res[0].([]interface{})[0].([]interface{})[0].(string)) 
	}
}

func HandleVV(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "🫣")
	quoted := v.Message.ExtendedTextMessage.GetContextInfo().GetQuotedMessage()
	if quoted == nil { 
		reply(client, v.Info.Chat, "⚠️ Reply to ViewOnce media.")
		return 
	}
	data, err := downloadMedia(client, &waProto.Message{
		ImageMessage: quoted.ImageMessage, 
		VideoMessage: quoted.VideoMessage, 
		ViewOnceMessage: quoted.ViewOnceMessage, 
		ViewOnceMessageV2: quoted.ViewOnceMessageV2,
	})
	if err != nil { 
		reply(client, v.Info.Chat, "❌ Failed to download.")
		return 
	}
	if quoted.ImageMessage != nil || (quoted.ViewOnceMessage != nil && quoted.ViewOnceMessage.Message.ImageMessage != nil) {
		up, _ := client.Upload(context.Background(), data, whatsmeow.MediaImage)
		client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				URL: proto.String(up.URL), 
				DirectPath: proto.String(up.DirectPath), 
				MediaKey: up.MediaKey,
				FileEncSHA256: up.FileEncSHA256, 
				FileSHA256: up.FileSHA256, 
				Mimetype: proto.String("image/jpeg"),
			}})
	} else {
		up, _ := client.Upload(context.Background(), data, whatsmeow.MediaVideo)
		client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				URL: proto.String(up.URL), 
				DirectPath: proto.String(up.DirectPath), 
				MediaKey: up.MediaKey,
				FileEncSHA256: up.FileEncSHA256, 
				FileSHA256: up.FileSHA256, 
				Mimetype: proto.String("video/mp4"),
			}})
	}
}

// ہیلپر فنکشنز
func getJson(url string, target interface{}) error { 
	r, err := http.Get(url)
	if err != nil { 
		return err 
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target) 
}

func downloadMedia(client *whatsmeow.Client, m *waProto.Message) ([]byte, error) { 
	var d whatsmeow.DownloadableMessage
	if m.ImageMessage != nil { 
		d = m.ImageMessage 
	} else if m.VideoMessage != nil { 
		d = m.VideoMessage 
	} else if m.DocumentMessage != nil { 
		d = m.DocumentMessage 
	} else if m.StickerMessage != nil { 
		d = m.StickerMessage 
	} else if m.ExtendedTextMessage != nil && m.ExtendedTextMessage.ContextInfo != nil { 
		q := m.ExtendedTextMessage.ContextInfo.QuotedMessage
		if q != nil { 
			if q.ImageMessage != nil { 
				d = q.ImageMessage 
			} else if q.VideoMessage != nil { 
				d = q.VideoMessage 
			} else if q.StickerMessage != nil { 
				d = q.StickerMessage 
			} 
		} 
	}
	if d == nil { 
		return nil, fmt.Errorf("no media") 
	}
	return client.Download(context.Background(), d) 
}

func uploadToCatbox(d []byte) string { 
	b := new(bytes.Buffer)
	w := multipart.NewWriter(b)
	p, _ := w.CreateFormFile("fileToUpload", "f.jpg")
	p.Write(d)
	w.WriteField("reqtype", "fileupload")
	w.Close()
	r, _ := http.Post("https://catbox.moe/user/api.php", w.FormDataContentType(), b)
	res, _ := ioutil.ReadAll(r.Body)
	return string(res) 
}

func sendVideo(client *whatsmeow.Client, chat types.JID, url, c string) { 
	r, _ := http.Get(url)
	d, _ := ioutil.ReadAll(r.Body)
	up, _ := client.Upload(context.Background(), d, whatsmeow.MediaVideo)
	client.SendMessage(context.Background(), chat, &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			URL: proto.String(up.URL), 
			DirectPath: proto.String(up.DirectPath), 
			MediaKey: up.MediaKey,
			FileEncSHA256: up.FileEncSHA256, 
			FileSHA256: up.FileSHA256, 
			Mimetype: proto.String("video/mp4"), 
			Caption: proto.String(c),
		}}) 
}

func sendImage(client *whatsmeow.Client, chat types.JID, url, c string) { 
	r, _ := http.Get(url)
	d, _ := ioutil.ReadAll(r.Body)
	up, _ := client.Upload(context.Background(), d, whatsmeow.MediaImage)
	client.SendMessage(context.Background(), chat, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			URL: proto.String(up.URL), 
			DirectPath: proto.String(up.DirectPath), 
			MediaKey: up.MediaKey,
			FileEncSHA256: up.FileEncSHA256, 
			FileSHA256: up.FileSHA256, 
			Mimetype: proto.String("image/jpeg"), 
			Caption: proto.String(c),
		}}) 
}

func sendDocument(client *whatsmeow.Client, chat types.JID, url, n, m string) { 
	r, _ := http.Get(url)
	d, _ := ioutil.ReadAll(r.Body)
	up, _ := client.Upload(context.Background(), d, whatsmeow.MediaDocument)
	client.SendMessage(context.Background(), chat, &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			URL: proto.String(up.URL), 
			DirectPath: proto.String(up.DirectPath), 
			MediaKey: up.MediaKey,
			FileEncSHA256: up.FileEncSHA256, 
			FileSHA256: up.FileSHA256, 
			Mimetype: proto.String(m), 
			FileName: proto.String(n),
		}}) 
}