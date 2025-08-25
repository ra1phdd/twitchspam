package ports

type ChatPort interface {
	Connect() error
	Join(channel string)
	Part(channel string)
	Say(message, channel string)
}

type IRCMessage struct {
	MessageID    string
	UserID       string
	RoomID       string
	Username     string
	Text         string
	IsFirst      bool
	IsSubscriber bool
	IsMod        bool
	IsVIP        bool
	Tags         map[string]string
}
