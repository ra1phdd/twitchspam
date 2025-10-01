package ports

import (
	"strings"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
)

type APIPort interface {
	GetChannelID(username string) (string, error)
	GetLiveStream() (*Stream, error)
	GetUrlVOD(streams []*config.Markers) (map[string]string, error)
	SendChatMessages(msgs *AnswerType)
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
	Text      MessageText
	EmoteOnly bool     // если Fragments type == "text" отсутствует
	Emotes    []string // text в Fragments при type == "emote"
}

type MessageText struct {
	Original string

	lower          *string
	normalized     *string
	lowerNorm      *string
	words          *[]string
	wordsLower     *[]string
	wordsNorm      *[]string
	wordsLowerNorm *[]string
}

type Reply struct {
	ParentChatter Chatter
	ParentMessage Message
}

func (m *MessageText) ReplaceOriginal(text string) {
	m.Original = text
	m.lower = nil
	m.normalized = nil
	m.lowerNorm = nil
	m.words = nil
	m.wordsLower = nil
	m.wordsNorm = nil
	m.wordsLowerNorm = nil
}

func (m *MessageText) Lower() string {
	if m.lower == nil {
		l := strings.ToLower(m.Original)
		m.lower = &l
	}
	return *m.lower
}

func (m *MessageText) Normalized() string {
	if m.normalized == nil {
		n := domain.NormalizeText(m.Original)
		m.normalized = &n
	}
	return *m.normalized
}

func (m *MessageText) LowerNorm() string {
	if m.lowerNorm == nil {
		ln := strings.ToLower(m.Normalized())
		m.lowerNorm = &ln
	}
	return *m.lowerNorm
}

func (m *MessageText) Words() []string {
	if m.words != nil {
		return *m.words
	}

	words := strings.Fields(m.Original)
	m.words = &words
	return words
}

func (m *MessageText) WordsLower() []string {
	if m.wordsLower == nil {
		wl := make([]string, len(m.Words()))
		for i, w := range m.Words() {
			wl[i] = strings.ToLower(w)
		}
		m.wordsLower = &wl
	}
	return *m.wordsLower
}

func (m *MessageText) WordsNorm() []string {
	if m.wordsNorm == nil {
		wn := strings.Fields(m.Normalized())
		m.wordsNorm = &wn
	}
	return *m.wordsNorm
}

func (m *MessageText) WordsLowerNorm() []string {
	if m.wordsLowerNorm == nil {
		wln := make([]string, len(m.WordsNorm()))
		for i, w := range m.WordsNorm() {
			wln[i] = strings.ToLower(w)
		}
		m.wordsLowerNorm = &wln
	}
	return *m.wordsLowerNorm
}
