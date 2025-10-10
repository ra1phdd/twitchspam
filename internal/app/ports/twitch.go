package ports

import (
	"time"
	"twitchspam/internal/app/infrastructure/config"
)

type APIPort interface {
	GetChannelID(username string) (string, error)
	GetLiveStream(channelID string) (*Stream, error)
	GetUrlVOD(channelID string, streams []*config.Markers) (map[string]string, error)
	SendChatMessages(channelID string, msgs *AnswerType)
	SendChatMessage(channelID, message string) error
	SendChatAnnouncement(channelID, message, color string) error
	DeleteChatMessage(channelID, messageID string) error
	TimeoutUser(channelID, userID string, duration int, reason string)
	BanUser(channelID, userID string, reason string)
	SearchCategory(gameName string) (string, string, error)
	UpdateChannelGameID(broadcasterID string, gameID string) error
	UpdateChannelTitle(broadcasterID string, title string) error
}

type IRCPort interface {
	AddChannel(channel string)
	RemoveChannel(channel string)
	WaitForIRC(msgID string, timeout time.Duration) (bool, bool)
	NotifyIRC(msgID string, isFirst bool)
}

type EventSubPort interface {
	AddChannel(channelID, channelName string, stream StreamPort)
}

type Stream struct {
	ID          string
	IsOnline    bool
	ViewerCount int
	StartedAt   time.Time
}
