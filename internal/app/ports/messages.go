package ports

type AdminPort interface {
	FindMessages(irc *IRCMessage) ActionType
}

type UserPort interface {
	FindMessages(irc *IRCMessage) ActionType
}
