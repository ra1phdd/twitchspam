package ports

import (
	"twitchspam/internal/app/infrastructure/config"
)

type TemplatePort interface {
	Aliases() AliasesPort
	Placeholders() PlaceholdersPort
	Banwords() BanwordsPort
	Regex() RegexPort
	Options() OptionsPort
	Parser() ParserPort
	Punishment() PunishmentPort
}

type AliasesPort interface {
	Update(newAliases map[string]string)
	Replace(parts []string) (string, bool)
}

type PlaceholdersPort interface {
	ReplaceAll(text string, parts []string) string
}

type BanwordsPort interface {
	CheckMessage(textLower string, wordsOriginal []string) bool
}

type RegexPort interface {
	MatchPhrase(words []string, phrase string) bool
}

type OptionsPort interface {
	ParseAll(words *[]string, opts map[string]struct{}) map[string]bool
	Parse(words *[]string, opt string) *bool
	MergeMword(dst config.MwordOptions, src map[string]bool) config.MwordOptions
	MergeExcept(dst config.ExceptOptions, src map[string]bool) config.ExceptOptions
	ExceptToString(opts config.ExceptOptions) string
	MwordToString(opts config.MwordOptions) string
}

type ParserPort interface {
	ParseIntArg(valStr string, min int, max int) (int, bool)
	ParseFloatArg(valStr string, min float64, max float64) (float64, bool)
}

type PunishmentPort interface {
	Parse(punishment string, allowInherit bool) (config.Punishment, error)
	FormatAll(punishments []config.Punishment) []string
	Format(punishment config.Punishment) string
}
