package ports

type BanwordsPort interface {
	CheckMessage(text, textOriginal string) bool
}
