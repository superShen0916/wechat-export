package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/superShen0916/wechat-export/internal/model"
)

// ExportJSON 将消息导出为 JSON 文件
// outputDir: 输出目录
// conv: 对话数据
func ExportJSON(outputDir string, conv model.Conversation) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	filename := sanitizeFilename(conv.Talker.NickName) + ".json"
	outputPath := filepath.Join(outputDir, filename)

	// 添加导出元信息
	output := map[string]interface{}{
		"exported_at": time.Now().Format(time.RFC3339),
		"talker":      conv.Talker,
		"total":       conv.Total,
		"messages":    conv.Messages,
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(output); err != nil {
		return "", fmt.Errorf("JSON 编码失败: %w", err)
	}

	return outputPath, nil
}
