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
	irc    ports.IRCPort
	cache  ports.CachePort[map[bool][]config.Punishment] // ключ - мворд применяется только для первых сообщений или нет
	mwords []Mwords
}

type Mwords struct {
	Word        string
	Punishments []config.Punishment `json:"punishments"`
	Options     config.MwordOptions `json:"options"`
	Regexp      *regexp.Regexp      `json:"regexp"`
}

func NewMword(irc ports.IRCPort, mwords map[string]*config.Mword, mwordGroups map[string]*config.MwordGroup) *MwordTemplate {
	mt := &MwordTemplate{
		irc:   irc,
		cache: storage.NewCache[map[bool][]config.Punishment](1000, 3*time.Minute),
	}
	mt.Update(mwords, mwordGroups)

	return mt
}

func (t *MwordTemplate) Update(mwords map[string]*config.Mword, mwordGroups map[string]*config.MwordGroup) {
	var mws []Mwords
	for word, mw := range mwords {
		mws = append(mws, Mwords{
			Word:        word,
			Punishments: mw.Punishments,
			Options:     mw.Options,
			Regexp:      mw.Regexp,
		})
	}

	for _, mwg := range mwordGroups {
		if !mwg.Enabled {
			continue
		}

		for _, w := range mwg.Words {
			mws = append(mws, Mwords{
				Word:        w,
				Punishments: mwg.Punishments,
				Options:     mwg.Options,
			})
		}

		for _, re := range mwg.Regexp {
			mws = append(mws, Mwords{
				Word:        "",
				Punishments: mwg.Punishments,
				Options:     mwg.Options,
				Regexp:      re,
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
		if trueMatch, exists := match[true]; exists {
			isFirst, _ := t.irc.WaitForIRC(msg.Message.ID, 250*time.Millisecond)
			if isFirst {
				return trueMatch
			}
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
	if opts.IsFirst {
		if isFirst, _ := t.irc.WaitForIRC(msg.Message.ID, 250*time.Millisecond); !isFirst {
			return false
		}
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
		text = msg.Message.Text.Text()
		words = msg.Message.Text.Words()
	case opts.CaseSensitive:
		text = msg.Message.Text.Text(domain.RemovePunctuation, domain.RemoveDuplicateLetters)
		words = msg.Message.Text.Words(domain.RemovePunctuation, domain.RemoveDuplicateLetters)
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

	if opts.Contains || strings.Contains(word, " ") {
		return strings.Contains(text, word)
	}
	return slices.Contains(words, word)
}

func (t *MwordTemplate) getCacheKey(msg *domain.ChatMessage) string {
	return fmt.Sprintf("%s_%v_%v", msg.Message.Text.Text(domain.RemovePunctuation),
		msg.Chatter.IsVip, msg.Chatter.IsSubscriber)
}
