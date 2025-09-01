package ports

type AdminPort interface {
	FindMessages(msg *ChatMessage) ActionType
}

type UserPort interface {
	FindMessages(msg *ChatMessage) ActionType
}
