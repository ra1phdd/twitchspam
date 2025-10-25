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
	mword        ports.MwordPort
	nuke         ports.NukePort
}

type Option func(*Template)

func WithAliases(aliases map[string]string, groups map[string]*config.AliasGroups, globalAliases map[string]string) Option {
	return func(t *Template) {
		t.aliases = NewAliases(aliases, groups, globalAliases)
	}
}

func WithPlaceholders(stream ports.StreamPort) Option {
	return func(t *Template) {
		t.placeholders = NewPlaceholders(stream)
	}
}

func WithBanwords(log logger.Logger, words []string, regexps []*regexp.Regexp) Option {
	return func(t *Template) {
		t.banwords = NewBanwords(log, words, regexps)
	}
}

func WithMword(mwords []config.Mword, mwordGroups map[string]*config.MwordGroup) Option {
	return func(t *Template) {
		t.mword = NewMword(t.options, mwords, mwordGroups)
	}
}

func New(opts ...Option) *Template {
	t := &Template{
		options:    NewOptions(),
		parser:     NewParser(),
		punishment: NewPunishment(),
		spamPause:  NewSpamPause(),
		nuke:       NewNuke(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
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

func (t *Template) Mword() ports.MwordPort {
	return t.mword
}

func (t *Template) Nuke() ports.NukePort {
	return t.nuke
}

func (t *Template) CheckOneWord(words []string) bool {
	if len(words) == 1 {
		return true
	}

	first := words[0]
	for _, w := range words[1:] {
		if w != first {
			return false
		}
	}
	return true
}
