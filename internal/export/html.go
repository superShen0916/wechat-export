package export

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/superShen0916/wechat-export/internal/model"
)

const htmlTpl = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Talker.NickName}} 的聊天记录</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
         background: #f0f0f0; color: #333; }
  .header { background: #07c160; color: white; padding: 16px 20px;
            position: sticky; top: 0; z-index: 10; box-shadow: 0 2px 8px rgba(0,0,0,.15); }
  .header h1 { font-size: 18px; }
  .header p  { font-size: 12px; opacity: .8; margin-top: 4px; }
  .messages  { max-width: 800px; margin: 0 auto; padding: 20px 16px; }
  .day-label { text-align: center; margin: 20px 0 12px;
               font-size: 12px; color: #999; }
  .msg       { display: flex; margin-bottom: 16px; align-items: flex-end; }
  .msg.me    { flex-direction: row-reverse; }
  .avatar    { width: 36px; height: 36px; border-radius: 6px;
               background: #ccc; display: flex; align-items: center;
               justify-content: center; font-size: 14px; font-weight: bold;
               flex-shrink: 0; }
  .msg.me .avatar    { background: #07c160; color: white; margin-left: 10px; }
  .msg:not(.me) .avatar { background: #fff; color: #07c160; margin-right: 10px;
                           border: 1px solid #ddd; }
  .bubble    { max-width: 70%; padding: 10px 14px; border-radius: 10px;
               font-size: 15px; line-height: 1.5; word-break: break-word;
               position: relative; }
  .msg:not(.me) .bubble { background: white; border-radius: 2px 10px 10px 10px; }
  .msg.me .bubble        { background: #95ec69; border-radius: 10px 2px 10px 10px; }
  .bubble .time { font-size: 11px; color: #999; margin-top: 4px; display: block; }
  .type-image { color: #999; font-style: italic; }
  .type-voice { color: #666; }
  .system { text-align: center; margin: 8px 0;
            font-size: 12px; color: #999; padding: 4px 12px;
            background: rgba(0,0,0,.05); border-radius: 10px; display: inline-block; }
  .system-wrap { text-align: center; }
</style>
</head>
<body>
<div class="header">
  <h1>💬 {{.Talker.NickName}}{{if .Talker.Remark}} ({{.Talker.Remark}}){{end}}</h1>
  <p>共 {{.Total}} 条消息 · 导出时间 {{.ExportedAt}}</p>
</div>
<div class="messages">
{{range .Messages}}
  {{if isSystem .Type}}
  <div class="system-wrap"><span class="system">{{.Content}}</span></div>
  {{else}}
  <div class="msg {{if .IsSender}}me{{end}}">
    <div class="avatar">{{if .IsSender}}我{{else}}T{{end}}</div>
    <div class="bubble">
      {{renderContent .}}
      <span class="time">{{formatTime .CreateTime}}</span>
    </div>
  </div>
  {{end}}
{{end}}
</div>
</body>
</html>`

type htmlData struct {
	Talker     model.Contact
	Total      int
	ExportedAt string
	Messages   []model.Message
}

// ExportHTML 将消息导出为 HTML 文件
func ExportHTML(outputDir string, conv model.Conversation) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	filename := sanitizeFilename(conv.Talker.NickName) + ".html"
	outputPath := filepath.Join(outputDir, filename)

	funcMap := template.FuncMap{
		"isSystem": func(t model.MsgType) bool {
			return t == model.MsgTypeSystem || t == model.MsgTypeRecall
		},
		"formatTime": func(ts int64) string {
			return time.Unix(ts, 0).Format("2006-01-02 15:04")
		},
		"renderContent": func(m model.Message) template.HTML {
			switch m.Type {
			case model.MsgTypeImage:
				return template.HTML(`<span class="type-image">[图片]</span>`)
			case model.MsgTypeVoice:
				return template.HTML(`<span class="type-voice">🎵 [语音]</span>`)
			case model.MsgTypeVideo:
				return template.HTML(`<span class="type-image">🎬 [视频]</span>`)
			case model.MsgTypeEmoji:
				return template.HTML(`<span class="type-image">[表情]</span>`)
			case model.MsgTypeFile:
				return template.HTML(`<span class="type-image">📎 [文件/链接]</span>`)
			default:
				content := m.Content
				if m.DisplayContent != "" {
					content = m.DisplayContent
				}
				return template.HTML(template.HTMLEscapeString(content))
			}
		},
	}

	tpl, err := template.New("chat").Funcs(funcMap).Parse(htmlTpl)
	if err != nil {
		return "", fmt.Errorf("模板解析失败: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	data := htmlData{
		Talker:     conv.Talker,
		Total:      conv.Total,
		ExportedAt: time.Now().Format("2006-01-02 15:04:05"),
		Messages:   conv.Messages,
	}

	if err := tpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("渲染模板失败: %w", err)
	}

	return outputPath, nil
}
