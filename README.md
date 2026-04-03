# wechat-export

微信聊天记录导出工具（macOS）

纯 Go 实现，无 CGo 依赖，支持导出为 JSON / CSV / HTML 三种格式。

## 功能特性

- ✅ **纯 Go SQLCipher 解密**：基于 Go 标准库实现 AES-256-CBC 解密，无需任何 C 依赖
- 🔍 **自动定位数据库**：自动查找 macOS 微信客户端的消息数据库路径
- 📤 **三种导出格式**：
  - JSON：结构化数据，便于后续分析
  - CSV：表格格式，可直接用 Excel 打开
  - HTML：自包含网页，支持查看聊天记录
- ⚡ **灵活筛选**：可按联系人、群聊、时间范围筛选导出
- ⚙️ **命令行友好**：cobra 框架，支持 TAB 补全

## 快速开始

### 前置准备

需要获取微信数据库密钥，推荐工具：

- [Thearas/wechat-db-decrypt-macos](https://github.com/Thearas/wechat-db-decrypt-macos)：一键提取 macOS 微信数据库密钥

### 安装

```bash
# 方法一：从 GitHub 安装（需 Go 1.25+）
go install github.com/superShen0916/wechat-export@latest

# 方法二：下载源码编译
git clone https://github.com/superShen0916/wechat-export.git
cd wechat-export
go build -o wechat-export ./cmd
```

### 基本用法

```bash
# 1. 列出所有聊天对象
wechat-export list --key <64位十六进制密钥>

# 2. 导出指定联系人/群聊（JSON 格式）
wechat-export export --contact wxid_xxx --key <密钥> --format json

# 3. 导出所有聊天记录（HTML 格式）
wechat-export export --all --key <密钥> --format html

# 4. 指定时间范围导出
wechat-export export --contact wxid_xxx --key <密钥> --format csv \
  --since 2024-01-01 --until 2024-12-31
```

### 更多命令

```bash
# 解密数据库到普通 SQLite（仅解密不导出）
wechat-export decrypt --key <密钥> --db /path/to/your.db --out decrypted.db

# 查找本机微信数据库路径
wechat-export find

# 查看帮助
wechat-export --help
```

## 使用场景

1. **个人数据备份**：将微信聊天记录导出到本地，避免丢失
2. **数据迁移**：将聊天记录迁移到其他设备或工具
3. **数据分析**：导出为 JSON 后，配合 `wechat-analyzer` 做 AI 分析
4. **合规存档**：满足企业合规要求，导出聊天记录存档

## 技术细节

### 数据库加密说明

macOS 微信使用 **SQLCipher 4** 加密本地数据库：
- 加密算法：AES-256-CBC
- 密钥派生函数：PBKDF2-SHA512，256000 次迭代
- 数据库路径：`~/Library/Containers/com.tencent.xinWeChat/Data/Library/Application Support/com.tencent.xinWeChat/[版本]/[uuid]/Message/*.db`

### 密钥获取

密钥提取需要读取微信进程内存，这涉及到操作系统级的操作，推荐使用专门的密钥提取工具：
- [Thearas/wechat-db-decrypt-macos](https://github.com/Thearas/wechat-db-decrypt-macos)
- [cocohahaha/wechat-decrypt-macos](https://github.com/cocohahaha/wechat-decrypt-macos)

## 安全说明

- 所有操作都在本地完成，不会将聊天数据上传到任何第三方
- 导出的 JSON/CSV/HTML 不包含任何敏感信息（如微信 ID 已被脱敏）
- 强烈建议在导出完成后及时删除密钥文件

## 兼容性

- ✅ macOS 微信 4.0+ (Intel & Apple Silicon)
- ✅ Go 1.25+
- ✅ macOS 12.0+

## 常见问题

**Q: 为什么会提示"数据库已加密"？**
A: 请确保已提供正确的 64 位十六进制密钥，且密钥与微信版本匹配。

**Q: 导出的 JSON 文件如何打开？**
A: JSON 文件可以用任何文本编辑器打开，也可以配合 `wechat-analyzer` 做 AI 分析。

**Q: 能导出微信群聊吗？**
A: 可以，`--contact` 参数支持群聊 ID，格式为 `xxx@chatroom`。

**Q: 导出的 HTML 里有图片吗？**
A: 暂时不支持导出图片/语音等媒体文件，仅导出文本消息。

## License

MIT
