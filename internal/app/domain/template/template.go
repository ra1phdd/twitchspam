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
	options      ports.OptionsPort
	parser       ports.ParserPort
	punishment   ports.PunishmentPort
	spamPause    ports.SpamPausePort
}

func New(log logger.Logger, aliases map[string]string, aliasGroups map[string]*config.AliasGroups, banWords []string, banRegexps []*regexp.Regexp, stream ports.StreamPort) *Template {
	return &Template{
		aliases:      NewAliases(aliases, aliasGroups),
		placeholders: NewPlaceholders(stream),
		banwords:     NewBanwords(log, banWords, banRegexps),
		options:      NewOptions(),
		parser:       NewParser(),
		punishment:   NewPunishment(),
		spamPause:    NewSpamPause(),
	}
}

var zeroWidthRunes = map[rune]struct{}{
	'\u200B': {}, // ZERO WIDTH SPACE
	'\u200C': {}, // ZERO WIDTH NON-JOINER
	'\u200D': {}, // ZERO WIDTH JOINER
	'\uFEFF': {}, // ZERO WIDTH NO-BREAK SPACE (BOM)в
}

func (t *Template) CleanMessage(text string) string {
	out := make([]rune, 0, len(text))
	for _, r := range text {
		if _, bad := zeroWidthRunes[r]; bad {
			continue
		}
		out = append(out, r)
	}
	return string(out)
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

func (t *Template) Options() ports.OptionsPort {
	return t.options
}

func (t *Template) Parser() ports.ParserPort {
	return t.parser
}

func (t *Template) Punishment() ports.PunishmentPort {
	return t.punishment
}

func (t *Template) SpamPause() ports.SpamPausePort {
	return t.spamPause
}
