package ports

import (
	"time"
	"twitchspam/internal/app/infrastructure/config"
)

type APIPollPort interface {
	Submit(task func()) error
	Stop()
}

type APIPort interface {
	Pool() APIPollPort
	GetChannelID(username string) (string, error)
	GetLiveStreams(channelIDs []string) ([]*Stream, error)
	GetUrlVOD(channelID string, streams []*config.Markers) (map[string]string, error)
	SendChatMessages(channelID string, msgs *AnswerType)
	SendChatMessage(channelID, message string) error
	SendChatAnnouncement(channelID, message, color string) error
	DeleteChatMessage(channelName, channelID, messageID string) error
	TimeoutUser(channelName, channelID, userID string, duration int, reason string)
	BanUser(channelName, channelID, userID string, reason string)
	SearchCategory(gameName string) (string, string, error)
	UpdateChannelGameID(broadcasterID string, gameID string) error
	UpdateChannelTitle(broadcasterID string, title string) error
	ManageHeldAutoModMessage(userID, msgID, action string) error
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
	UserID      string
	UserLogin   string
	Username    string
	ViewerCount int
	StartedAt   time.Time
}
