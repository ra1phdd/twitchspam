package ports

type TemplatePort interface {
	ReplaceAlias(text string) string
	UpdateAliases(newAliases map[string]string)
	ReplacePlaceholders(text string, parts []string) string
	CheckOnBanwords(text, textOriginal string) bool
	MatchPhrase(words []string, phrase string) bool
	ParseOptions(words *[]string, opts map[string]struct{}) map[string]bool
	ParseOption(words *[]string, opt string) *bool
}
