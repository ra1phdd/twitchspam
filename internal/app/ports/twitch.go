package ports

import (
	"time"
	"twitchspam/internal/app/infrastructure/config"
)

type APIPort interface {
	GetChannelID(username string) (string, error)
	GetLiveStream() (*Stream, error)
	GetUrlVOD(streams []*config.Markers) (map[string]string, error)
	SendChatMessages(msgs *AnswerType)
	SendChatMessage(message string) error
	SendChatAnnouncement(message, color string) error
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
