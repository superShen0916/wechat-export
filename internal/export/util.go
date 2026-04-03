package export

import "strings"

// sanitizeFilename 清理文件名中的非法字符
func sanitizeFilename(name string) string {
	if name == "" {
		name = "unknown"
	}
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	s := replacer.Replace(name)
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}
