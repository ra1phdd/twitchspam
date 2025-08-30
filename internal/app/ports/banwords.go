package ports

type BanwordsPort interface {
	CheckMessage(words []string) bool
	CheckOnline(text string) bool
}
