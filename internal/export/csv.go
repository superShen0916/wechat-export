package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/superShen0916/wechat-export/internal/model"
)

// ExportCSV 将消息导出为 CSV 文件
func ExportCSV(outputDir string, conv model.Conversation) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	filename := sanitizeFilename(conv.Talker.NickName) + ".csv"
	outputPath := filepath.Join(outputDir, filename)

	f, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	// 写入 BOM（让 Excel 正确识别 UTF-8）
	f.WriteString("\xEF\xBB\xBF")

	w := csv.NewWriter(f)
	defer w.Flush()

	// 写表头
	w.Write([]string{"时间", "发送方", "消息类型", "内容", "local_id"})

	talkerName := conv.Talker.NickName
	if conv.Talker.Remark != "" {
		talkerName = conv.Talker.Remark
	}

	for _, msg := range conv.Messages {
		sender := talkerName
		if msg.IsSender {
			sender = "我"
		}
		t := time.Unix(msg.CreateTime, 0).Format("2006-01-02 15:04:05")
		content := msg.Content
		if msg.DisplayContent != "" && msg.DisplayContent != msg.Content {
			content = msg.DisplayContent
		}
		w.Write([]string{t, sender, msg.TypeName, content, strconv.FormatInt(msg.LocalID, 10)})
	}

	return outputPath, nil
}
