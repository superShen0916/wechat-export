package model

// MsgType 消息类型
type MsgType int

const (
	MsgTypeText    MsgType = 1
	MsgTypeImage   MsgType = 3
	MsgTypeVoice   MsgType = 34
	MsgTypeVideo   MsgType = 43
	MsgTypeEmoji   MsgType = 47
	MsgTypeFile    MsgType = 49 // 文件/链接/小程序等 (XML)
	MsgTypeSystem  MsgType = 10000
	MsgTypeRecall  MsgType = 10002
)

func (t MsgType) String() string {
	switch t {
	case MsgTypeText:
		return "text"
	case MsgTypeImage:
		return "image"
	case MsgTypeVoice:
		return "voice"
	case MsgTypeVideo:
		return "video"
	case MsgTypeEmoji:
		return "emoji"
	case MsgTypeFile:
		return "file"
	case MsgTypeSystem:
		return "system"
	case MsgTypeRecall:
		return "recall"
	default:
		return "unknown"
	}
}

// Message 单条消息
type Message struct {
	LocalID     int64   `json:"local_id"`
	MsgSvrID    string  `json:"msg_svr_id"`
	Type        MsgType `json:"type"`
	TypeName    string  `json:"type_name"`
	IsSender    bool    `json:"is_sender"`  // true = 自己发的
	CreateTime  int64   `json:"create_time"` // Unix timestamp (秒)
	StrTalker   string  `json:"talker"`      // 对方 wxid 或群 id
	Content     string  `json:"content"`
	DisplayContent string `json:"display_content,omitempty"`
}

// Contact 联系人信息
type Contact struct {
	UserName string `json:"user_name"` // wxid
	NickName string `json:"nick_name"`
	Alias    string `json:"alias,omitempty"`
	Remark   string `json:"remark,omitempty"`
	IsGroup  bool   `json:"is_group"`
}

// Conversation 一段对话（导出单位）
type Conversation struct {
	Talker   Contact   `json:"talker"`
	Messages []Message `json:"messages"`
	Total    int       `json:"total"`
}
