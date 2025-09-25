package template

import (
	"regexp"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Template struct {
	aliases      *AliasesTemplate
	placeholders *PlaceholdersTemplate
	banwords     *BanwordsTemplate
	regex        *RegexTemplate
	options      *OptionsTemplate
	parser       *ParserTemplate
	punishment   *PunishmentTemplate
}

func New(log logger.Logger, al map[string]string, banWords []string, banRegexps []*regexp.Regexp, stream ports.StreamPort) *Template {
	return &Template{
		aliases:      NewAliases(al),
		placeholders: NewPlaceholders(stream),
		banwords:     NewBanwords(log, banWords, banRegexps),
		regex:        NewRegex(),
		options:      NewOptions(),
		parser:       NewParser(),
		punishment:   NewPunishment(),
	}
}

func (t *Template) ReplaceAlias(parts []string) (string, bool) {
	return t.aliases.replace(parts)
}

func (t *Template) UpdateAliases(newAliases map[string]string) {
	t.aliases.update(newAliases)
}

func (t *Template) ReplacePlaceholders(text string, parts []string) string {
	return t.placeholders.replaceAll(text, parts)
}

func (t *Template) CheckOnBanwords(textLower string, wordsOriginal []string) bool {
	return t.banwords.checkMessage(textLower, wordsOriginal)
}

func (t *Template) MatchPhrase(words []string, phrase string) bool {
	return t.regex.matchPhrase(words, phrase)
}

func (t *Template) ParseOptions(words *[]string, opts map[string]struct{}) map[string]bool {
	return t.options.parseAll(words, opts)
}

func (t *Template) ParseOption(words *[]string, opt string) *bool {
	return t.options.parse(words, opt)
}

func (t *Template) ParseIntArg(valStr string, min, max int) (int, bool) {
	return t.parser.parseIntArg(valStr, min, max)
}

func (t *Template) ParseFloatArg(valStr string, min, max float64) (float64, bool) {
	return t.parser.parseFloatArg(valStr, min, max)
}

func (t *Template) ParsePunishment(punishment string, allowInherit bool) (config.Punishment, error) {
	return t.punishment.parse(punishment, allowInherit)
}

func (t *Template) FormatPunishments(punishments []config.Punishment) []string {
	return t.punishment.formatAll(punishments)
}

func (t *Template) FormatPunishment(punishment config.Punishment) string {
	return t.punishment.format(punishment)
}
