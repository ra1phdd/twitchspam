package ports

type ModerationPort interface {
	Timeout(userID string, duration int, reason string)
	Ban(userID string, reason string)
}
