package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// ==================== ٹولز سسٹم ====================

func handleSticker(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "🎨")
	
	msg := `╔═════════════════════╗
║   🎨 STICKER PROCESSING    
╠═════════════════════╣
║  ⏳ Creating sticker...    
║  Please wait...           
╚═════════════════════╝`
	replyMessage(client, v, msg)

	// Robust Media Extraction
	data, err := downloadMedia(client, v.Message)
	if err != nil {
		errMsg := `╔═════════════════╗
║  ❌ NO MEDIA FOUND       
╠═════════════════╣
║  Reply to an image or     
║  video to create sticker  
╚═════════════════╝`
		replyMessage(client, v, errMsg)
		return
	}

	tempIn := fmt.Sprintf("temp_%s.jpg", v.Info.ID)
	tempOut := fmt.Sprintf("temp_%s.webp", v.Info.ID)

	os.WriteFile(tempIn, data, 0644)
	exec.Command("ffmpeg", "-y", "-i", tempIn, "-vcodec", "libwebp", "-filter:v", "scale='if(gt(a,1),512,-1)':'if(gt(a,1),-1,512)'", tempOut).Run()
	
	b, _ := os.ReadFile(tempOut)
	up, err := client.Upload(context.Background(), b, whatsmeow.MediaImage)
	if err != nil {
		fmt.Printf("❌ [STICKER] Upload failed: %v\n", err)
		return
	}

	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(b))), // Fixed
			Mimetype:      proto.String("image/webp"),
		},
	})

	os.Remove(tempIn)
	os.Remove(tempOut)
}

func handleToImg(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "🖼️")
	
	msg := `╔══════════════════╗
║ 🖼️ IMAGE CONVERSION      
╠══════════════════╣
║ ⏳ Converting to image... 
║       Please wait...           
╚══════════════════╝`
	replyMessage(client, v, msg)

	data, err := downloadMedia(client, v.Message)
	if err != nil {
		errMsg := `╔══════════════════╗
║  ❌ NO STICKER FOUND     
╠══════════════════╣
║  Reply to a sticker to    
║  convert it to image      
╚══════════════════╝`
		replyMessage(client, v, errMsg)
		return
	}

	tempIn := fmt.Sprintf("temp_%s.webp", v.Info.ID)
	tempOut := fmt.Sprintf("temp_%s.png", v.Info.ID)

	os.WriteFile(tempIn, data, 0644)
	exec.Command("ffmpeg", "-y", "-i", tempIn, tempOut).Run()
	
	b, _ := os.ReadFile(tempOut)
	up, err := client.Upload(context.Background(), b, whatsmeow.MediaImage)
	if err != nil {
		return
	}

	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(b))), // Fixed
			Mimetype:      proto.String("image/png"),
			Caption:       proto.String("✅ Converted to Image"),
		},
	})

	os.Remove(tempIn)
	os.Remove(tempOut)
}

func handleToVideo(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "🎥")
	
	msg := `╔═════════════════╗
║ 🎥 VIDEO CONVERSION      
╠═════════════════╣
║ ⏳ Converting to video... 
║       Please wait...           
╚═════════════════╝`
	replyMessage(client, v, msg)

	data, err := downloadMedia(client, v.Message)
	if err != nil {
		errMsg := `╔══════════════════╗
║  ❌ NO STICKER FOUND     
╠══════════════════╣
║  Reply to a sticker to    
║  convert it to video      
╚══════════════════╝`
		replyMessage(client, v, errMsg)
		return
	}

	tempIn := fmt.Sprintf("temp_%s.webp", v.Info.ID)
	tempOut := fmt.Sprintf("temp_%s.mp4", v.Info.ID)

	os.WriteFile(tempIn, data, 0644)
	exec.Command("ffmpeg", "-y", "-i", tempIn, "-pix_fmt", "yuv420p", tempOut).Run()
	
	d, _ := os.ReadFile(tempOut)
	up, err := client.Upload(context.Background(), d, whatsmeow.MediaVideo)
	if err != nil {
		return
	}

	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(d))), // Fixed
			Mimetype:      proto.String("video/mp4"),
			Caption:       proto.String("✅ Converted to Video"),
		},
	})

	os.Remove(tempIn)
	os.Remove(tempOut)
}

func handleRemoveBG(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "✂️")
	
	msg := `╔════════════════════╗
║ ✂️ BACKGROUND REMOVAL     
╠════════════════════╣
║  ⏳ Removing background... 
║          Please wait...           
╚════════════════════╝`
	replyMessage(client, v, msg)

	d, err := downloadMedia(client, v.Message)
	if err != nil {
		errMsg := `╔═════════════════╗
║  ❌ NO IMAGE FOUND       
╠═════════════════╣
║  Reply to an image to     
║  remove background        
╚═════════════════╝`
		replyMessage(client, v, errMsg)
		return
	}

	u := uploadToCatbox(d)
	imgURL := "https://bk9.fun/tools/removebg?url=" + u

	r, err := http.Get(imgURL)
	if err != nil {
		return
	}
	defer r.Body.Close()
	
	imgData, _ := io.ReadAll(r.Body)
	up, err := client.Upload(context.Background(), imgData, whatsmeow.MediaImage)
	if err != nil {
		return
	}

	client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imgData))), // Fixed
			Mimetype:      proto.String("image/png"),
			Caption:       proto.String("✂️ Background Removed\n\n✅ Successfully Processed"),
		},
	})
}

func handleRemini(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "✨")
	
	// 1. میسج یا رپلائی میں امیج ڈھونڈیں
	var imgMsg *waProto.ImageMessage
	if v.Message.ImageMessage != nil {
		imgMsg = v.Message.ImageMessage
	} else if v.Message.GetExtendedTextMessage().GetContextInfo() != nil {
		quoted := v.Message.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
		if quoted != nil && quoted.ImageMessage != nil {
			imgMsg = quoted.ImageMessage
		}
	}

	if imgMsg == nil {
		replyMessage(client, v, "╔═══════════════════╗\n║ ❌ NO IMAGE FOUND    \n╠═══════════════════╣\n║ Please reply to an \n║ image to enhance.  \n╚═══════════════════╝")
		return
	}

	replyMessage(client, v, "╔═══════════════════╗\n║ ✨ IMAGE ENHANCE    \n╠═══════════════════╣\n║ ⏳ Enhancing...    \n║ Please wait a moment\n╚═══════════════════╝")

	ctx := context.Background()
	data, err := client.Download(ctx, imgMsg)
	if err != nil {
		return
	}

	u := uploadToCatbox(data)

	type ReminiResponse struct {
		Status string `json:"status"`
		Url    string `json:"url"`
	}
	
	var r ReminiResponse
	apiUrl := "https://remini.mobilz.pw/enhance?url=" + u
	getJson(apiUrl, &r)

	if r.Url != "" {
		resp, err := http.Get(r.Url)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		
		enhancedData, _ := io.ReadAll(resp.Body)
		up, err := client.Upload(ctx, enhancedData, whatsmeow.MediaImage)
		if err != nil {
			return
		}

		msgToSend := &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				URL:           proto.String(up.URL),
				DirectPath:    proto.String(up.DirectPath),
				MediaKey:      up.MediaKey,
				Mimetype:      proto.String("image/jpeg"),
				FileSHA256:    up.FileSHA256,
				FileEncSHA256: up.FileEncSHA256,
				FileLength:    proto.Uint64(uint64(len(enhancedData))), // Fixed
				Caption:       proto.String("✨ *IMAGE ENHANCED*\n\n✅ Quality successfully improved!"),
			},
		}
		client.SendMessage(ctx, v.Info.Chat, msgToSend)
	} else {
		replyMessage(client, v, "╔═══════════════════╗\n║ ❌ FAILED           \n╠═══════════════════╣\n║ API could not     \n║ process the image. \n╚═══════════════════╝")
	}
}

func handleToURL(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "🔗")
	
	msg := `╔══════════════════╗
║  🔗 UPLOADING MEDIA       
╠══════════════════╣
║ ⏳ Uploading to server... 
║         Please wait...           
╚══════════════════╝`
	replyMessage(client, v, msg)

	d, err := downloadMedia(client, v.Message)
	if err != nil {
		errMsg := `╔═════════════════╗
║  ❌ NO MEDIA FOUND       
╠═══════════════════╣
║ Reply to media to get URL
╚═══════════════════╝`
		replyMessage(client, v, errMsg)
		return
	}

	uploadURL := uploadToCatbox(d)
	
	resultMsg := fmt.Sprintf(`╔═════════════════╗
║  🔗 MEDIA UPLOADED        
╠═════════════════╣
║                           
║  📎 *Direct Link:* ║  %s                       
║                           
║ ✅ *Successfully Uploaded*
║                           
╚═══════════════════╝`, uploadURL)

	replyMessage(client, v, resultMsg)
}

func handleWeather(client *whatsmeow.Client, v *events.Message, city string) {
	if city == "" {
		msg := `╔════════════════════╗
║🌤️ WEATHER INFORMATION   
╠════════════════════╣
║                           
║  Usage:                   
║  .weather <city>          
║                           
║  Example:                 
║  .weather Karachi         
║             .weather London          
║                           
╚════════════════════╝`
		replyMessage(client, v, msg)
		return
	}

	react(client, v.Info.Chat, v.Info.ID, "🌦️")
	
	r, err := http.Get("https://wttr.in/" + city + "?format=%C+%t")
	if err != nil {
		replyMessage(client, v, "❌ Weather fetch failed.")
		return
	}
	defer r.Body.Close()

	d, _ := io.ReadAll(r.Body)
	weatherInfo := string(d)

	msg := fmt.Sprintf(`╔═══════════════╗
║  🌤️ WEATHER INFO          
╠═══════════════╣
║                           
║  📍 *City:* %s            
║  🌡️ *Info:* %s            
║                           
╚═══════════════╝`, city, weatherInfo)

	replyMessage(client, v, msg)
}

func handleTranslate(client *whatsmeow.Client, v *events.Message, args []string) {
	react(client, v.Info.Chat, v.Info.ID, "🌍")

	t := strings.Join(args, " ")
	if t == "" {
		if v.Message.GetExtendedTextMessage().GetContextInfo() != nil {
			q := v.Message.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
			if q != nil {
				t = q.GetConversation()
			}
		}
	}

	if t == "" {
		replyMessage(client, v, "╔══════════════╗\n║   🌍 TRANSLATOR            \n╠══════════════╣\n║  Usage: .tr <text>  \n╚═══════════════════╝")
		return
	}

	r, err := http.Get(fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=ur&dt=t&q=%s", url.QueryEscape(t)))
	if err != nil {
		return
	}
	defer r.Body.Close()

	var res []interface{}
	json.NewDecoder(r.Body).Decode(&res)

	if len(res) > 0 {
		translated := res[0].([]interface{})[0].([]interface{})[0].(string)
		msg := fmt.Sprintf(`╔═══════════════════╗
║ 🌍 TRANSLATION RESULT    
╠═══════════════════╣
║                           
║  📝 *Original:* ║  %s                       
║                           
║  📝 *Translated:* ║  %s                       
║                           
╚════════════════════╝`, t, translated)
		replyMessage(client, v, msg)
	}
}

func handleVV(client *whatsmeow.Client, v *events.Message) {
	react(client, v.Info.Chat, v.Info.ID, "🫣")

	cInfo := v.Message.GetExtendedTextMessage().GetContextInfo()
	if cInfo == nil {
		replyMessage(client, v, "⚠️ Please reply to a ViewOnce media.")
		return
	}

	quoted := cInfo.GetQuotedMessage()
	if quoted == nil { return }

	var (
		imgMsg *waProto.ImageMessage
		vidMsg *waProto.VideoMessage
		audMsg *waProto.AudioMessage
	)

	// Direct check and ViewOnce extraction
	if quoted.ImageMessage != nil {
		imgMsg = quoted.ImageMessage
	} else if quoted.VideoMessage != nil {
		vidMsg = quoted.VideoMessage
	} else if quoted.AudioMessage != nil {
		audMsg = quoted.AudioMessage
	} else {
		vo := quoted.GetViewOnceMessage().GetMessage()
		if vo == nil { vo = quoted.GetViewOnceMessageV2().GetMessage() }
		if vo != nil {
			if vo.ImageMessage != nil { imgMsg = vo.ImageMessage }
			if vo.VideoMessage != nil { vidMsg = vo.VideoMessage }
		}
	}

	if imgMsg == nil && vidMsg == nil && audMsg == nil {
		replyMessage(client, v, "❌ No copyable media found.")
		return
	}

	ctx := context.Background()
	var data []byte
	var err error
	var mType whatsmeow.MediaType

	if imgMsg != nil {
		data, err = client.Download(ctx, imgMsg)
		mType = whatsmeow.MediaImage
	} else if vidMsg != nil {
		data, err = client.Download(ctx, vidMsg)
		mType = whatsmeow.MediaVideo
	} else if audMsg != nil {
		data, err = client.Download(ctx, audMsg)
		mType = whatsmeow.MediaAudio
	}

	if err != nil || len(data) == 0 { return }

	up, err := client.Upload(ctx, data, mType)
	if err != nil { return }

	var finalMsg waProto.Message
	cap := "📂 *RETRIEVED MEDIA*"

	if imgMsg != nil {
		finalMsg.ImageMessage = &waProto.ImageMessage{
			URL: proto.String(up.URL), DirectPath: proto.String(up.DirectPath),
			MediaKey: up.MediaKey, Mimetype: proto.String("image/jpeg"),
			FileSHA256: up.FileSHA256, FileEncSHA256: up.FileEncSHA256,
			FileLength: proto.Uint64(uint64(len(data))), Caption: proto.String(cap),
		}
	} else if vidMsg != nil {
		finalMsg.VideoMessage = &waProto.VideoMessage{
			URL: proto.String(up.URL), DirectPath: proto.String(up.DirectPath),
			MediaKey: up.MediaKey, Mimetype: proto.String("video/mp4"),
			FileSHA256: up.FileSHA256, FileEncSHA256: up.FileEncSHA256,
			FileLength: proto.Uint64(uint64(len(data))), Caption: proto.String(cap),
		}
	} else if audMsg != nil {
		finalMsg.AudioMessage = &waProto.AudioMessage{
			URL: proto.String(up.URL), DirectPath: proto.String(up.DirectPath),
			MediaKey: up.MediaKey, Mimetype: proto.String("audio/ogg; codecs=opus"),
			FileSHA256: up.FileSHA256, FileEncSHA256: up.FileEncSHA256,
			FileLength: proto.Uint64(uint64(len(data))), PTT: proto.Bool(false),
		}
	}

	client.SendMessage(ctx, v.Info.Chat, &finalMsg)
}

// ==================== میڈیا ہیلپرز ====================

func downloadMedia(client *whatsmeow.Client, m *waProto.Message) ([]byte, error) {
	var d whatsmeow.DownloadableMessage
	
	// 1. Direct message check
	if m.ImageMessage != nil {
		d = m.ImageMessage
	} else if m.VideoMessage != nil {
		d = m.VideoMessage
	} else if m.StickerMessage != nil {
		d = m.StickerMessage
	} else if m.AudioMessage != nil {
		d = m.AudioMessage
	} else if m.GetExtendedTextMessage().GetContextInfo() != nil {
		// 2. Quoted message check
		q := m.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
		if q != nil {
			if q.ImageMessage != nil { d = q.ImageMessage
			} else if q.VideoMessage != nil { d = q.VideoMessage
			} else if q.StickerMessage != nil { d = q.StickerMessage
			} else if q.AudioMessage != nil { d = q.AudioMessage
			} else if q.GetViewOnceMessage().GetMessage() != nil {
				vo := q.GetViewOnceMessage().GetMessage()
				if vo.ImageMessage != nil { d = vo.ImageMessage } else if vo.VideoMessage != nil { d = vo.VideoMessage }
			} else if q.GetViewOnceMessageV2().GetMessage() != nil {
				vo := q.GetViewOnceMessageV2().GetMessage()
				if vo.ImageMessage != nil { d = vo.ImageMessage } else if vo.VideoMessage != nil { d = vo.VideoMessage }
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
	r, err := http.Post("https://catbox.moe/user/api.php", w.FormDataContentType(), b)
	if err != nil { return "" }
	defer r.Body.Close()
	res, _ := io.ReadAll(r.Body)
	return string(res)
}