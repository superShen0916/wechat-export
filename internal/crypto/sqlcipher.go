// Package crypto 实现 SQLCipher 4 解密（WeChat macOS 版使用）
//
// WeChat 使用 WCDB（基于 SQLCipher 4），加密参数：
//   - 加密算法：AES-256-CBC
//   - Page size：4096 字节
//   - Reserved 区：48 字节（16 IV + 32 HMAC）
//   - 密钥：从进程内存提取的 32 字节原始密钥（64 位十六进制字符串）
//
// 密钥提取工具：https://github.com/Thearas/wechat-db-decrypt-macos
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"os"
)

const (
	pageSize     = 4096
	saltSize     = 16
	ivSize       = 16
	hmacSize     = 32
	reservedSize = ivSize + hmacSize // 48
)

// sqliteMagic SQLite 文件头魔数
var sqliteMagic = []byte("SQLite format 3\x00")

// DecryptDB 解密 SQLCipher 4 数据库文件
// keyHex: 64 位十六进制字符串（32 字节原始密钥）
// srcPath: 加密的 .db 文件路径
// dstPath: 输出的明文 SQLite 文件路径
func DecryptDB(keyHex, srcPath, dstPath string) error {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return fmt.Errorf("密钥格式错误，需要 64 位十六进制字符串: %w", err)
	}
	if len(key) != 32 {
		return fmt.Errorf("密钥长度错误：需要 32 字节（64 hex），实际 %d 字节", len(key))
	}

	src, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("读取数据库失败: %w", err)
	}
	if len(src) < pageSize {
		return fmt.Errorf("文件太小，不是有效的 SQLCipher 数据库")
	}
	if len(src)%pageSize != 0 {
		return fmt.Errorf("文件大小不是 %d 的整数倍，可能不是 SQLCipher 数据库", pageSize)
	}

	// 检查是否已经是明文 SQLite
	if string(src[:16]) == string(sqliteMagic) {
		return fmt.Errorf("文件已经是明文 SQLite，无需解密")
	}

	numPages := len(src) / pageSize
	dst := make([]byte, len(src))

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	for i := 0; i < numPages; i++ {
		srcPage := src[i*pageSize : (i+1)*pageSize]
		dstPage := dst[i*pageSize : (i+1)*pageSize]

		// IV 在 reserved 区的前 16 字节
		iv := srcPage[pageSize-reservedSize : pageSize-reservedSize+ivSize]

		// 复制 reserved 区（IV + HMAC）到输出，SQLite 会忽略 reserved 字节
		copy(dstPage[pageSize-reservedSize:], srcPage[pageSize-reservedSize:])

		// 确定加密内容范围
		encStart := 0
		if i == 0 {
			// 第一页：前 16 字节是 salt（明文），加密内容从 16 开始
			encStart = saltSize
			// 用 SQLite 魔数替换 salt
			copy(dstPage[:saltSize], sqliteMagic)
		}

		encData := srcPage[encStart : pageSize-reservedSize]
		decData := dstPage[encStart : pageSize-reservedSize]

		mode := cipher.NewCBCDecrypter(block, iv)
		mode.CryptBlocks(decData, encData)
	}

	// 写入临时文件
	if err := os.WriteFile(dstPath, dst, 0600); err != nil {
		return fmt.Errorf("写入解密文件失败: %w", err)
	}

	return nil
}

// IsEncrypted 检查文件是否是 SQLCipher 加密数据库
func IsEncrypted(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	header := make([]byte, 16)
	if _, err := f.Read(header); err != nil {
		return false, err
	}

	return string(header) != string(sqliteMagic), nil
}
