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
	SendChatAnnouncements(channelID string, msgs *AnswerType, color string)
	SendChatAnnouncement(channelID, message, color string) error
	DeleteChatMessage(channelName, channelID, messageID string) error
	TimeoutUser(channelName, channelID, userID string, duration int, reason string)
	WarnUser(channelName, broadcasterID, userID, reason string) error
	BanUser(channelName, channelID, userID string, reason string)
	SearchCategory(gameName string) (string, string, error)
	UpdateChannelCategoryID(broadcasterID string, gameID string) error
	UpdateChannelTitle(broadcasterID string, title string) error
	ManageHeldAutoModMessage(userID, msgID, action string) error
	CreatePrediction(broadcasterID, title string, outcomes []string, predictionWindow int) (*Predictions, error)
	EndPrediction(broadcasterID, predictionID, status, winningOutcomeID string) error
	CreatePoll(broadcasterID, title string, choices []string, duration int, enablePoints bool, pointsPerVote int) (*Poll, error)
	EndPoll(broadcasterID, pollID, status string) error
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
	GameName    string
	StartedAt   time.Time
}

type Predictions struct {
	ID               string
	Title            string
	WinningOutcomeID string
	Outcomes         []PredictionsOutcome
	PredictionWindow int
	Status           string
	CreatedAt        time.Time
	EndedAt          time.Time
	LockedAt         time.Time
}

type PredictionsOutcome struct {
	ID    string
	Title string
}

type Poll struct {
	ID                         string
	Title                      string
	Choices                    []PollChoiceResponse
	Status                     string
	Duration                   int
	StartedAt                  time.Time
	EndedAt                    time.Time
	ChannelPointsVotingEnabled bool
	ChannelPointsPerVote       int
}

type PollChoiceResponse struct {
	ID    string
	Title string
}
