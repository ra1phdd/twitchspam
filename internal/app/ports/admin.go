package ports

type AdminPort interface {
	FindMessages(irc *IRCMessage) ActionType
}
