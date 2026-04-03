package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/superShen0916/wechat-export/internal/model"
	_ "modernc.org/sqlite"
)

// Reader 数据库读取器
type Reader struct {
	db *sql.DB
}

// Open 打开明文 SQLite 数据库
func Open(path string) (*Reader, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接失败（可能仍是加密状态）: %w", err)
	}
	return &Reader{db: db}, nil
}

// Close 关闭数据库
func (r *Reader) Close() error {
	return r.db.Close()
}

// Tables 返回数据库中所有表名（用于调试）
func (r *Reader) Tables() ([]string, error) {
	rows, err := r.db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		tables = append(tables, name)
	}
	return tables, nil
}

// ListTalkers 列出所有聊天对象（去重的 StrTalker）
func (r *Reader) ListTalkers() ([]string, error) {
	// 尝试不同的表名（不同版本可能不同）
	query := `
		SELECT DISTINCT StrTalker, COUNT(*) as cnt
		FROM MSG
		WHERE StrTalker != ''
		GROUP BY StrTalker
		ORDER BY cnt DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询聊天对象失败: %w", err)
	}
	defer rows.Close()

	var talkers []string
	for rows.Next() {
		var talker string
		var cnt int
		if err := rows.Scan(&talker, &cnt); err != nil {
			continue
		}
		talkers = append(talkers, talker)
	}
	return talkers, nil
}

// QueryOptions 查询选项
type QueryOptions struct {
	Talker    string    // 指定聊天对象 wxid（空表示全部）
	Since     time.Time // 起始时间（零值表示不限）
	Until     time.Time // 结束时间（零值表示不限）
	Limit     int       // 最大条数（0 表示不限）
	OnlyText  bool      // 只要文本消息
}

// QueryMessages 查询消息
func (r *Reader) QueryMessages(opts QueryOptions) ([]model.Message, error) {
	var conditions []string
	var args []interface{}

	if opts.Talker != "" {
		conditions = append(conditions, "StrTalker = ?")
		args = append(args, opts.Talker)
	}
	if !opts.Since.IsZero() {
		conditions = append(conditions, "CreateTime >= ?")
		args = append(args, opts.Since.Unix())
	}
	if !opts.Until.IsZero() {
		conditions = append(conditions, "CreateTime <= ?")
		args = append(args, opts.Until.Unix())
	}
	if opts.OnlyText {
		conditions = append(conditions, "Type = 1")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	limit := ""
	if opts.Limit > 0 {
		limit = fmt.Sprintf("LIMIT %d", opts.Limit)
	}

	query := fmt.Sprintf(`
		SELECT localId, COALESCE(MsgSvrID,''), Type, COALESCE(SubType,0),
		       IsSender, CreateTime, StrTalker,
		       COALESCE(StrContent,''), COALESCE(DisplayContent,'')
		FROM MSG
		%s
		ORDER BY CreateTime ASC, localId ASC
		%s`, where, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询消息失败: %w", err)
	}
	defer rows.Close()

	var msgs []model.Message
	for rows.Next() {
		var m model.Message
		var msgType int
		var isSender int
		if err := rows.Scan(
			&m.LocalID, &m.MsgSvrID, &msgType, new(int),
			&isSender, &m.CreateTime, &m.StrTalker,
			&m.Content, &m.DisplayContent,
		); err != nil {
			continue
		}
		m.Type = model.MsgType(msgType)
		m.TypeName = m.Type.String()
		m.IsSender = isSender == 1
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// QueryContacts 查询联系人（如果有 WCContact 或 Contact 表）
func (r *Reader) QueryContacts() (map[string]model.Contact, error) {
	contacts := make(map[string]model.Contact)

	// 尝试多种可能的表名
	tableNames := []string{"WCContact", "Contact"}
	var found bool
	for _, tableName := range tableNames {
		var count int
		err := r.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='%s'", tableName)).Scan(&count)
		if err != nil || count == 0 {
			continue
		}

		rows, err := r.db.Query(fmt.Sprintf(
			"SELECT COALESCE(userName,''), COALESCE(dbContactNickName,''), COALESCE(dbContactRemark,'') FROM %s", tableName))
		if err != nil {
			continue
		}
		defer rows.Close()

		for rows.Next() {
			var c model.Contact
			var rawNick, rawRemark []byte
			if err := rows.Scan(&c.UserName, &rawNick, &rawRemark); err != nil {
				continue
			}
			c.NickName = extractNickName(rawNick)
			c.Remark = extractNickName(rawRemark)
			c.IsGroup = strings.HasSuffix(c.UserName, "@chatroom")
			if c.UserName != "" {
				contacts[c.UserName] = c
			}
		}
		found = true
		break
	}

	if !found {
		// 没有联系人表，从消息表中推断
		talkers, err := r.ListTalkers()
		if err == nil {
			for _, t := range talkers {
				contacts[t] = model.Contact{
					UserName: t,
					NickName: t,
					IsGroup:  strings.HasSuffix(t, "@chatroom"),
				}
			}
		}
	}

	return contacts, nil
}

// extractNickName 从 BLOB 字段提取昵称（微信用 protobuf 编码存储，尝试简单提取可读字符串）
func extractNickName(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	// 尝试直接作为 UTF-8 字符串
	s := string(raw)
	if isPrintable(s) {
		return s
	}
	// 从 blob 中提取可打印字符串片段
	var result strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] >= 0x20 && raw[i] < 0x7f {
			result.WriteByte(raw[i])
		} else if raw[i] > 0x7f {
			// 可能是 UTF-8 多字节字符
			result.WriteByte(raw[i])
		}
	}
	return strings.TrimSpace(result.String())
}

func isPrintable(s string) bool {
	for _, r := range s {
		if r < 0x20 && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}
