package ports

type BanwordsPort interface {
	CheckMessage(words []string) bool
}
