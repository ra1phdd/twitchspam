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
	store        ports.StoresPort
}

type Option func(*Template)

func WithAliases(aliases map[string]string, groups map[string]*config.AliasGroups) Option {
	return func(t *Template) {
		t.aliases = NewAliases(aliases, groups)
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

func WithMword(irc ports.IRCPort, mwords map[string]*config.Mword, mwordGroups map[string]*config.MwordGroup) Option {
	return func(t *Template) {
		t.mword = NewMword(irc, mwords, mwordGroups)
	}
}

func WithStore(cfg *config.Config) Option {
	return func(t *Template) {
		t.store = NewStores(cfg)
	}
}

func New(opts ...Option) *Template {
	t := &Template{
		options:    NewOptions(),
		parser:     NewParser(),
		punishment: NewPunishment(),
		spamPause:  NewSpamPause(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
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

func (t *Template) Mword() ports.MwordPort {
	return t.mword
}

func (t *Template) Store() ports.StoresPort {
	return t.store
}
