package template

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/ports"
)

type MwordTemplate struct {
	options ports.OptionsPort
	cache   ports.CachePort[map[bool][]config.Punishment] // ключ - мворд применяется только для первых сообщений или нет
	mwords  []Mwords
}

type Mwords struct {
	Punishments []config.Punishment
	Options     config.MwordOptions
	Word        string
	NameRegexp  string
	Regexp      *regexp.Regexp
}

func NewMword(options ports.OptionsPort, mwords []config.Mword, mwordGroups map[string]*config.MwordGroup) *MwordTemplate {
	mt := &MwordTemplate{
		options: options,
		cache:   storage.NewCache[map[bool][]config.Punishment](1000, 3*time.Minute),
	}
	mt.Update(mwords, mwordGroups)

	return mt
}

func (t *MwordTemplate) Update(mwords []config.Mword, mwordGroups map[string]*config.MwordGroup) {
	mws := make([]Mwords, 0, len(mwords)+len(mwordGroups))
	for _, mw := range mwords {
		mws = append(mws, Mwords{
			Punishments: mw.Punishments,
			Options:     mw.Options,
			Word:        mw.Word,
			NameRegexp:  mw.NameRegexp,
			Regexp:      mw.Regexp,
		})
	}

	for _, mwg := range mwordGroups {
		if !mwg.Enabled {
			continue
		}

		for _, mw := range mwg.Words {
			punishments := mwg.Punishments
			if len(mw.Punishments) != 0 {
				punishments = mw.Punishments
			}

			options := mwg.Options
			if mw.Options != (config.MwordOptions{}) {
				options = t.options.MergeMword(mwg.Options, map[string]bool{
					"is_first":       mw.Options.IsFirst,
					"no_sub":         mw.Options.NoSub,
					"no_vip":         mw.Options.NoVip,
					"norepeat":       mw.Options.NoRepeat,
					"one_word":       mw.Options.OneWord,
					"contains":       mw.Options.Contains,
					"case_sensitive": mw.Options.CaseSensitive,
				})
			}

			mws = append(mws, Mwords{
				Punishments: punishments,
				Options:     options,
				Word:        mw.Word,
				NameRegexp:  mw.NameRegexp,
				Regexp:      mw.Regexp,
			})
		}
	}

	hasEnabledOptions := func(o config.MwordOptions) bool {
		return o.IsFirst ||
			o.NoSub ||
			o.NoVip ||
			o.NoRepeat ||
			o.OneWord ||
			o.Contains ||
			o.CaseSensitive
	}

	sort.Slice(mws, func(i, j int) bool {
		return hasEnabledOptions(mws[i].Options) && !hasEnabledOptions(mws[j].Options)
	})

	t.mwords = mws
}

func (t *MwordTemplate) Check(msg *domain.ChatMessage) []config.Punishment {
	match, ok := t.cache.Get(t.getCacheKey(msg))
	if ok {
		if trueMatch, exists := match[true]; exists && msg.Message.IsFirst() {
			return trueMatch
		}

		if falseMatch, exists := match[false]; exists {
			return falseMatch
		}
	} else {
		match = make(map[bool][]config.Punishment)
	}

	for _, mw := range t.mwords {
		if !t.matchMwordRule(msg, mw.Word, mw.Regexp, mw.Options) {
			continue
		}

		match[mw.Options.IsFirst] = mw.Punishments
		t.cache.Set(t.getCacheKey(msg), match)

		return mw.Punishments
	}

	match[false] = nil
	t.cache.Set(t.getCacheKey(msg), match)
	return nil
}

func (t *MwordTemplate) matchMwordRule(msg *domain.ChatMessage, word string, re *regexp.Regexp, opts config.MwordOptions) bool {
	if opts.NoVip && msg.Chatter.IsVip {
		return false
	}
	if opts.NoSub && msg.Chatter.IsSubscriber {
		return false
	}
	if opts.IsFirst && !msg.Message.IsFirst() {
		return false
	}
	if opts.OneWord && len(msg.Message.Text.Words()) > 1 {
		return false
	}

	var text string
	var words []string
	switch {
	case opts.CaseSensitive && opts.NoRepeat:
		text = msg.Message.Text.Text()
		words = msg.Message.Text.Words()
	case opts.NoRepeat:
		text = msg.Message.Text.Text(domain.Lower)
		words = msg.Message.Text.Words(domain.Lower)
	case opts.CaseSensitive:
		text = msg.Message.Text.Text(domain.RemovePunctuation, domain.RemoveDuplicateLetters)
		words = msg.Message.Text.Words(domain.RemovePunctuation, domain.RemoveDuplicateLetters)
	case opts.Contains:
		text = msg.Message.Text.Text(domain.Lower)
		words = msg.Message.Text.Words(domain.Lower)
	default:
		text = msg.Message.Text.Text(domain.Lower, domain.RemovePunctuation, domain.RemoveDuplicateLetters)
		words = msg.Message.Text.Words(domain.Lower, domain.RemovePunctuation, domain.RemoveDuplicateLetters)
	}

	if re != nil {
		return re.MatchString(text)
	}

	if word == "" {
		return false
	}

	if opts.Contains {
		return strings.Contains(text, word)
	}
	return slices.Contains(words, word)
}

func (t *MwordTemplate) getCacheKey(msg *domain.ChatMessage) string {
	return fmt.Sprintf("%s_%v_%v", msg.Message.Text.Text(domain.RemovePunctuation),
		msg.Chatter.IsVip, msg.Chatter.IsSubscriber)
}
