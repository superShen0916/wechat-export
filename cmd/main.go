package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/superShen0916/wechat-export/internal/crypto"
	"github.com/superShen0916/wechat-export/internal/db"
	"github.com/superShen0916/wechat-export/internal/export"
	"github.com/superShen0916/wechat-export/internal/model"
)

var rootCmd = &cobra.Command{
	Use:   "wechat-export",
	Short: "微信聊天记录导出工具（macOS）",
	Long: `从 macOS 微信客户端导出聊天记录，支持 JSON / CSV / HTML 格式。

使用前需要获取数据库密钥，推荐工具：
  https://github.com/Thearas/wechat-db-decrypt-macos`,
}

// ── decrypt 命令 ──────────────────────────────────────────────────────────────
var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "解密微信数据库",
	Long:  "将 SQLCipher 加密的微信数据库解密为普通 SQLite 文件",
	RunE: func(cmd *cobra.Command, args []string) error {
		key, _ := cmd.Flags().GetString("key")
		src, _ := cmd.Flags().GetString("db")
		dst, _ := cmd.Flags().GetString("out")

		if key == "" {
			return fmt.Errorf("请使用 --key 提供 64 位十六进制密钥\n\n获取密钥：https://github.com/Thearas/wechat-db-decrypt-macos")
		}

		// 自动找数据库
		if src == "" {
			var err error
			src, err = db.FindMainDB()
			if err != nil {
				return fmt.Errorf("自动定位数据库失败: %w\n请使用 --db 手动指定路径", err)
			}
			fmt.Printf("📂 自动找到数据库: %s\n", src)
		}

		// 自动生成输出路径
		if dst == "" {
			dst = strings.TrimSuffix(src, filepath.Ext(src)) + "_plain.db"
		}

		fmt.Printf("🔓 解密中: %s\n", src)
		if err := crypto.DecryptDB(key, src, dst); err != nil {
			return fmt.Errorf("解密失败: %w\n\n提示：请确认密钥正确，并且微信版本与密钥提取工具匹配", err)
		}
		fmt.Printf("✅ 解密成功: %s\n", dst)
		return nil
	},
}

// ── list 命令 ─────────────────────────────────────────────────────────────────
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有聊天对象",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, _ := cmd.Flags().GetString("db")
		key, _ := cmd.Flags().GetString("key")

		dbPath, cleanup, err := prepareDB(dbPath, key)
		if err != nil {
			return err
		}
		defer cleanup()

		r, err := db.Open(dbPath)
		if err != nil {
			return err
		}
		defer r.Close()

		tables, _ := r.Tables()
		fmt.Printf("📊 数据表: %s\n\n", strings.Join(tables, ", "))

		contacts, err := r.QueryContacts()
		if err != nil {
			return err
		}

		talkers, err := r.ListTalkers()
		if err != nil {
			return err
		}

		fmt.Printf("%-30s %-20s %-10s\n", "wxid", "昵称", "类型")
		fmt.Println(strings.Repeat("─", 65))
		for _, t := range talkers {
			c, ok := contacts[t]
			if !ok {
				c = model.Contact{UserName: t, NickName: t}
			}
			cType := "好友"
			if c.IsGroup {
				cType = "群聊"
			}
			display := c.NickName
			if c.Remark != "" {
				display = c.Remark + " (" + c.NickName + ")"
			}
			fmt.Printf("%-30s %-20s %-10s\n", t, display, cType)
		}
		return nil
	},
}

// ── export 命令 ───────────────────────────────────────────────────────────────
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出聊天记录",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, _ := cmd.Flags().GetString("db")
		key, _ := cmd.Flags().GetString("key")
		talker, _ := cmd.Flags().GetString("contact")
		format, _ := cmd.Flags().GetString("format")
		outputDir, _ := cmd.Flags().GetString("output")
		all, _ := cmd.Flags().GetBool("all")
		sinceStr, _ := cmd.Flags().GetString("since")
		untilStr, _ := cmd.Flags().GetString("until")

		if talker == "" && !all {
			return fmt.Errorf("请指定 --contact <wxid> 或使用 --all 导出全部")
		}

		dbPath, cleanup, err := prepareDB(dbPath, key)
		if err != nil {
			return err
		}
		defer cleanup()

		r, err := db.Open(dbPath)
		if err != nil {
			return err
		}
		defer r.Close()

		contacts, _ := r.QueryContacts()

		opts := db.QueryOptions{Talker: talker}

		if sinceStr != "" {
			t, err := time.Parse("2006-01-02", sinceStr)
			if err != nil {
				return fmt.Errorf("--since 日期格式错误，请用 YYYY-MM-DD: %w", err)
			}
			opts.Since = t
		}
		if untilStr != "" {
			t, err := time.Parse("2006-01-02", untilStr)
			if err != nil {
				return fmt.Errorf("--until 日期格式错误，请用 YYYY-MM-DD: %w", err)
			}
			opts.Until = t.Add(24 * time.Hour)
		}

		if outputDir == "" {
			outputDir = "./wechat_export"
		}

		// 获取要导出的 talker 列表
		var talkers []string
		if all {
			talkers, err = r.ListTalkers()
			if err != nil {
				return err
			}
		} else {
			talkers = []string{talker}
		}

		fmt.Printf("📤 导出格式: %s → %s\n\n", format, outputDir)

		for _, t := range talkers {
			opts.Talker = t
			msgs, err := r.QueryMessages(opts)
			if err != nil {
				fmt.Printf("  ⚠️  %s: 查询失败 (%v)\n", t, err)
				continue
			}
			if len(msgs) == 0 {
				continue
			}

			contact, ok := contacts[t]
			if !ok {
				contact = model.Contact{UserName: t, NickName: t, IsGroup: strings.HasSuffix(t, "@chatroom")}
			}

			conv := model.Conversation{
				Talker:   contact,
				Messages: msgs,
				Total:    len(msgs),
			}

			var outPath string
			switch format {
			case "json":
				outPath, err = export.ExportJSON(outputDir, conv)
			case "csv":
				outPath, err = export.ExportCSV(outputDir, conv)
			case "html":
				outPath, err = export.ExportHTML(outputDir, conv)
			default:
				return fmt.Errorf("不支持的格式: %s（支持: json / csv / html）", format)
			}

			if err != nil {
				fmt.Printf("  ⚠️  %s: 导出失败 (%v)\n", t, err)
				continue
			}
			fmt.Printf("  ✅ %s (%d 条) → %s\n", contact.NickName, len(msgs), outPath)
		}

		fmt.Printf("\n🎉 导出完成！文件保存在: %s\n", outputDir)
		return nil
	},
}

// ── find 命令 ─────────────────────────────────────────────────────────────────
var findCmd = &cobra.Command{
	Use:   "find",
	Short: "查找本机微信数据库文件",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbs, err := db.FindDBs()
		if err != nil {
			return err
		}
		fmt.Printf("找到 %d 个数据库文件：\n\n", len(dbs))
		for i, d := range dbs {
			fi, _ := os.Stat(d.Path)
			size := ""
			if fi != nil {
				size = fmt.Sprintf("%.1f MB", float64(fi.Size())/1024/1024)
			}
			encrypted, _ := crypto.IsEncrypted(d.Path)
			status := "🔓 明文"
			if encrypted {
				status = "🔒 加密"
			}
			fmt.Printf("[%d] %s  %s  %s\n    %s\n\n", i+1, status, size, d.Version, d.Path)
		}
		return nil
	},
}

// ── 辅助函数 ──────────────────────────────────────────────────────────────────

// prepareDB 准备数据库文件（如果加密则先解密到临时文件）
// 返回可用的数据库路径和清理函数
func prepareDB(dbPath, key string) (string, func(), error) {
	if dbPath == "" {
		var err error
		dbPath, err = db.FindMainDB()
		if err != nil {
			return "", nil, fmt.Errorf("自动定位数据库失败: %w\n请使用 --db 手动指定路径", err)
		}
		fmt.Printf("📂 使用数据库: %s\n", dbPath)
	}

	encrypted, err := crypto.IsEncrypted(dbPath)
	if err != nil {
		return "", nil, err
	}

	if !encrypted {
		return dbPath, func() {}, nil
	}

	if key == "" {
		return "", nil, fmt.Errorf(`数据库已加密，请提供 --key 密钥

获取密钥方法：
  1. 安装工具：https://github.com/Thearas/wechat-db-decrypt-macos
  2. 运行后复制输出的 64 位十六进制密钥
  3. 使用：wechat-export --key <密钥> ...`)
	}

	// 解密到临时文件
	tmpFile, err := os.CreateTemp("", "wechat_*.db")
	if err != nil {
		return "", nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	fmt.Printf("🔓 解密数据库中...\n")
	if err := crypto.DecryptDB(key, dbPath, tmpPath); err != nil {
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("解密失败: %w", err)
	}
	fmt.Printf("✅ 解密成功\n\n")

	cleanup := func() {
		os.Remove(tmpPath)
	}
	return tmpPath, cleanup, nil
}

func init() {
	// decrypt 子命令
	decryptCmd.Flags().StringP("key", "k", "", "64 位十六进制密钥（必需）")
	decryptCmd.Flags().StringP("db", "d", "", "加密数据库路径（默认自动查找）")
	decryptCmd.Flags().StringP("out", "o", "", "输出路径（默认同目录加 _plain 后缀）")

	// list 子命令
	listCmd.Flags().StringP("key", "k", "", "密钥（数据库加密时必需）")
	listCmd.Flags().StringP("db", "d", "", "数据库路径（默认自动查找）")

	// export 子命令
	exportCmd.Flags().StringP("key", "k", "", "密钥（数据库加密时必需）")
	exportCmd.Flags().StringP("db", "d", "", "数据库路径（默认自动查找）")
	exportCmd.Flags().StringP("contact", "c", "", "联系人 wxid")
	exportCmd.Flags().StringP("format", "f", "json", "导出格式: json / csv / html")
	exportCmd.Flags().StringP("output", "o", "./wechat_export", "输出目录")
	exportCmd.Flags().Bool("all", false, "导出所有联系人")
	exportCmd.Flags().String("since", "", "开始日期 YYYY-MM-DD")
	exportCmd.Flags().String("until", "", "结束日期 YYYY-MM-DD")

	rootCmd.AddCommand(decryptCmd, listCmd, exportCmd, findCmd)
}

func main() {
	// 检查 cobra 是否安装
	if _, err := exec.LookPath("go"); err == nil {
		_ = err
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
