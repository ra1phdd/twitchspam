package ports

type Chat interface {
	GetChannelID(username string) (string, error)
	GetOnline(username string) (int, bool, error)
	SendChatMessage(broadcasterID, senderID, message string) error
}

type ModerationPort interface {
	Timeout(userID string, duration int, reason string)
	Ban(userID string, reason string)
}

type ChatMessage struct {
	Broadcaster Broadcaster
	Chatter     Chatter
	Message     Message
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
