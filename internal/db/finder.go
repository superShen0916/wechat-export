// Package db 负责定位和读取微信数据库
package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WechatDB 微信数据库信息
type WechatDB struct {
	Path    string
	Version string // WeChat 版本目录名
	UUID    string // 用户目录名
}

// FindDBs 在 macOS 上自动定位微信消息数据库
// 返回所有找到的 .db 文件路径（按修改时间排序，最新的在前）
func FindDBs() ([]WechatDB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("无法获取用户目录: %w", err)
	}

	// macOS 微信数据库路径
	baseDir := filepath.Join(home, "Library", "Containers",
		"com.tencent.xinWeChat", "Data", "Library",
		"Application Support", "com.tencent.xinWeChat")

	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("未找到微信数据目录，请确认已安装并登录过 macOS 微信\n路径：%s", baseDir)
	}

	var results []WechatDB

	// 遍历版本目录 → UUID 目录 → Message/*.db
	versionDirs, _ := os.ReadDir(baseDir)
	for _, vd := range versionDirs {
		if !vd.IsDir() {
			continue
		}
		versionPath := filepath.Join(baseDir, vd.Name())
		uuidDirs, _ := os.ReadDir(versionPath)
		for _, ud := range uuidDirs {
			if !ud.IsDir() {
				continue
			}
			msgDir := filepath.Join(versionPath, ud.Name(), "Message")
			dbFiles, _ := filepath.Glob(filepath.Join(msgDir, "*.db"))
			for _, dbPath := range dbFiles {
				results = append(results, WechatDB{
					Path:    dbPath,
					Version: vd.Name(),
					UUID:    ud.Name(),
				})
			}
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("未找到微信数据库文件，请确认已登录过微信")
	}

	// 按修改时间排序（最新的在前）
	sort.Slice(results, func(i, j int) bool {
		fi, _ := os.Stat(results[i].Path)
		fj, _ := os.Stat(results[j].Path)
		if fi == nil || fj == nil {
			return false
		}
		return fi.ModTime().After(fj.ModTime())
	})

	return results, nil
}

// FindMainDB 返回最可能是主消息数据库的路径（通常是最大的或名为 message 的）
func FindMainDB() (string, error) {
	dbs, err := FindDBs()
	if err != nil {
		return "", err
	}

	// 优先找 message_0.db 或最大的 db 文件
	var best string
	var bestSize int64
	for _, db := range dbs {
		name := strings.ToLower(filepath.Base(db.Path))
		if name == "message_0.db" || name == "enmicromsg.db" {
			return db.Path, nil
		}
		fi, err := os.Stat(db.Path)
		if err != nil {
			continue
		}
		if fi.Size() > bestSize {
			bestSize = fi.Size()
			best = db.Path
		}
	}

	if best == "" {
		return dbs[0].Path, nil
	}
	return best, nil
}
