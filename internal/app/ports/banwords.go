package ports

type BanwordsPort interface {
	CheckMessage(text string) bool
}
