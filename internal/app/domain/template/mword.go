package template

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
	"twitchspam/internal/app/domain/message"
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
	Options     *config.MwordOptions
	Word        string
	NameRegexp  string
	Regexp      *regexp.Regexp
}

func NewMword(options ports.OptionsPort, mwords []config.Mword, mwordGroups map[string]*config.MwordGroup) *MwordTemplate {
	mt := &MwordTemplate{
		options: options,
		cache:   storage.NewCache[map[bool][]config.Punishment](500, 3*time.Minute, false, false, "", 0),
	}
	mt.Update(mwords, mwordGroups)

	return mt
}

func (t *MwordTemplate) Update(mwords []config.Mword, mwordGroups map[string]*config.MwordGroup) {
	t.cache.ClearAll()

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
			if mw.Options != nil {
				src := make(map[string]bool)
				opts := map[*bool][2]string{
					mw.Options.IsFirst:       {"-first", "-nofirst"},
					mw.Options.NoSub:         {"-nosub", "-sub"},
					mw.Options.NoVip:         {"-novip", "-vip"},
					mw.Options.NoRepeat:      {"-norepeat", "-repeat"},
					mw.Options.OneWord:       {"-oneword", "-nooneword"},
					mw.Options.Contains:      {"-contains", "-nocontains"},
					mw.Options.CaseSensitive: {"-case", "-nocase"},
				}

				for opt, vals := range opts {
					if opt != nil {
						if *opt {
							src[vals[0]] = true
						} else {
							src[vals[1]] = true
						}
					}
				}

				options = t.options.MergeMword(mwg.Options, src)
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

	hasEnabledOptions := func(o *config.MwordOptions) bool {
		if o == nil {
			return false
		}
		return (o.IsFirst != nil && *o.IsFirst) ||
			(o.NoSub != nil && *o.NoSub) ||
			(o.NoVip != nil && *o.NoVip) ||
			(o.NoRepeat != nil && *o.NoRepeat) ||
			(o.OneWord != nil && *o.OneWord) ||
			(o.Contains != nil && *o.Contains) ||
			(o.CaseSensitive != nil && *o.CaseSensitive)
	}

	sort.Slice(mws, func(i, j int) bool {
		return hasEnabledOptions(mws[i].Options) && !hasEnabledOptions(mws[j].Options)
	})

	t.mwords = mws
}

func (t *MwordTemplate) Check(msg *message.ChatMessage, isLive bool) []config.Punishment {
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
		if !t.matchMwordRule(msg, mw.Word, mw.Regexp, mw.Options, isLive) {
			continue
		}

		isFirst := false
		if mw.Options != nil && mw.Options.IsFirst != nil {
			isFirst = *mw.Options.IsFirst
		}
		match[isFirst] = mw.Punishments
		t.cache.Set(t.getCacheKey(msg), match)

		return mw.Punishments
	}

	match[false] = nil
	t.cache.Set(t.getCacheKey(msg), match)
	return nil
}

func (t *MwordTemplate) matchMwordRule(msg *message.ChatMessage, word string, re *regexp.Regexp, opts *config.MwordOptions, isLive bool) bool {
	if word == "" {
		return false
	}

	mode := config.OnlineMode
	if opts != nil && opts.Mode != nil {
		mode = *opts.Mode
	}

	if ((mode == config.OnlineMode || mode == 0) && !isLive) || (mode == config.OfflineMode && isLive) {
		return false
	}

	text := msg.Message.Text.Text(message.LowerOption, message.RemovePunctuationOption, message.RemoveDuplicateLettersOption)
	words := msg.Message.Text.Words(message.LowerOption, message.RemovePunctuationOption, message.RemoveDuplicateLettersOption)
	if opts != nil {
		if opts.NoVip != nil && *opts.NoVip && msg.Chatter.IsVip {
			return false
		}
		if opts.NoSub != nil && *opts.NoSub && msg.Chatter.IsSubscriber {
			return false
		}
		if opts.IsFirst != nil && *opts.IsFirst && !msg.Message.IsFirst() {
			return false
		}
		if opts.OneWord != nil && *opts.OneWord && !t.CheckOneWord(words) {
			return false
		}

		textOpts := make([]message.TextOptionFuncWithID, 0, 3)
		if opts.SavePunctuation != nil && !*opts.SavePunctuation {
			textOpts = append(textOpts, message.RemovePunctuationOption)
		}
		if opts.NoRepeat == nil || !*opts.NoRepeat {
			textOpts = append(textOpts, message.RemoveDuplicateLettersOption)
		}
		if opts.CaseSensitive == nil || !*opts.CaseSensitive {
			textOpts = append(textOpts, message.LowerOption)
		}

		text = msg.Message.Text.Text(textOpts...)
		words = msg.Message.Text.Words(textOpts...)
	}

	if re != nil {
		return re.MatchString(text)
	}

	if (opts != nil && opts.Contains != nil && *opts.Contains) || strings.Contains(word, " ") {
		return strings.Contains(text, word)
	}
	return slices.Contains(words, word)
}

func (t *MwordTemplate) getCacheKey(msg *message.ChatMessage) string {
	return fmt.Sprintf("%s_%v_%v", msg.Message.Text.Text(message.RemovePunctuationOption),
		msg.Chatter.IsVip, msg.Chatter.IsSubscriber)
}

func (t *MwordTemplate) CheckOneWord(words []string) bool {
	if len(words) < 2 {
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
