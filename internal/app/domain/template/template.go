package template

import (
	"regexp"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Template struct {
	aliases      ports.AliasesPort
	placeholders ports.PlaceholdersPort
	banwords     ports.BanwordsPort
	regex        ports.RegexPort
	options      ports.OptionsPort
	parser       ports.ParserPort
	punishment   ports.PunishmentPort
}

func New(log logger.Logger, aliases map[string]string, aliasGroups map[string]*config.AliasGroups, banWords []string, banRegexps []*regexp.Regexp, stream ports.StreamPort) *Template {
	return &Template{
		aliases:      NewAliases(aliases, aliasGroups),
		placeholders: NewPlaceholders(stream),
		banwords:     NewBanwords(log, banWords, banRegexps),
		regex:        NewRegex(),
		options:      NewOptions(),
		parser:       NewParser(),
		punishment:   NewPunishment(),
	}
}

func (t *Template) Aliases() ports.AliasesPort {
	return t.aliases
}

func (t *Template) Placeholders() ports.PlaceholdersPort {
	return t.placeholders
}

func (t *Template) Banwords() ports.BanwordsPort {
	return t.banwords
}

func (t *Template) Regex() ports.RegexPort {
	return t.regex
}

func (t *Template) Options() ports.OptionsPort {
	return t.options
}

func (t *Template) Parser() ports.ParserPort {
	return t.parser
}

func (t *Template) Punishment() ports.PunishmentPort {
	return t.punishment
}
