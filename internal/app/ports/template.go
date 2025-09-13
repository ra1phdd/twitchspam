package ports

import "twitchspam/internal/app/infrastructure/config"

type TemplatePort interface {
	ReplaceAlias(text string) string
	UpdateAliases(newAliases map[string]string)
	ReplacePlaceholders(text string, parts []string) string
	CheckOnBanwords(text, textOriginal string) bool
	MatchPhrase(words []string, phrase string) bool
	ParseOptions(words *[]string) (bool, config.Options)
	MergeOptions(dst, src *config.Options)
}
