package ports

import (
	"time"
	"twitchspam/internal/app/infrastructure/config"
)

type APIPort interface {
	GetChannelID(username string) (string, error)
	GetLiveStream() (*Stream, error)
	GetUrlVOD(streams []config.Markers) (map[string]string, error)
	SendChatMessage(message string) error
	DeleteChatMessage(messageID string) error
	TimeoutUser(userID string, duration int, reason string)
	BanUser(userID string, reason string)
}

type IRCPort interface {
	WaitForIRC(msgID string, timeout time.Duration) (bool, bool)
	NotifyIRC(msgID string, isFirst bool)
}

type Stream struct {
	ID          string
	IsOnline    bool
	ViewerCount int
	StartedAt   time.Time
}

type ChatMessage struct {
	Broadcaster Broadcaster
	Chatter     Chatter
	Message     Message
	Reply       *Reply
}

type Broadcaster struct {
	UserID   string
	Login    string
	Username string
}

type Chatter struct {
	UserID        string
	Login         string
	Username      string
	IsBroadcaster bool
	IsMod         bool
	IsVip         bool
	IsSubscriber  bool
}

type Message struct {
	ID        string
	Text      string
	EmoteOnly bool     // если Fragments type == "text" отсутствует
	Emotes    []string // text в Fragments при type == "emote"
}

type Reply struct {
	ParentChatter Chatter
	ParentMessage Message
}
