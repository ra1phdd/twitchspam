package checker

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/seventv"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/domain/regex"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

const (
	None    string = "none"
	Ban     string = "ban"
	Timeout string = "timeout"
	Delete  string = "delete"
)

type Empty struct{}

type Checker struct {
	log logger.Logger
	cfg *config.Config

	stream   ports.StreamPort
	stats    ports.StatsPort
	timeouts ports.StorePort[Empty]
	messages ports.StorePort[string]
	bwords   ports.BanwordsPort
	sevenTV  ports.SevenTVPort
	regexp   ports.RegexPort
}

func NewCheck(log logger.Logger, cfg *config.Config, stream ports.StreamPort, stats ports.StatsPort, bwords ports.BanwordsPort, regexp *regex.Regex) *Checker {
	return &Checker{
		log:    log,
		cfg:    cfg,
		stream: stream,
		stats:  stats,
		timeouts: storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int {
			return 15
		}),
		messages: storage.New[string](runtime.NumCPU(), 500*time.Millisecond, func() int {
			defLimit := float64(cfg.Spam.SettingsDefault.MessageLimit*cfg.Spam.SettingsDefault.MinGapMessages) / cfg.Spam.SettingsDefault.SimilarityThreshold
			vipLimit := float64(cfg.Spam.SettingsVIP.MessageLimit*cfg.Spam.SettingsVIP.MinGapMessages) / cfg.Spam.SettingsVIP.SimilarityThreshold

			return int(max(defLimit, vipLimit))
		}),
		bwords:  bwords,
		sevenTV: seventv.New(log, stream),
		regexp:  regexp,
	}
}

func (c *Checker) Check(msg *ports.ChatMessage) *ports.CheckerAction {
	if action := c.checkBypass(msg); action != nil {
		return action
	}

	text := strings.ToLower(domain.NormalizeText(msg.Message.Text))
	words := strings.Fields(text)

	if !c.cfg.Spam.SettingsEmotes.Enabled && (msg.Message.EmoteOnly || c.sevenTV.IsOnlyEmotes(msg.Message.Text)) {
		return &ports.CheckerAction{Type: None}
	}

	if action := c.CheckBanwords(text, msg.Message.Text); action != nil {
		return action
	}

	if action := c.CheckAds(msg.Message.Text, msg.Chatter.Username); action != nil {
		return action
	}

	for _, t := range []string{text, msg.Message.Text} {
		if action := c.CheckMwords(t); action != nil {
			return action
		}
	}

	if action := c.checkWordLength(msg, words); action != nil {
		return action
	}

	if action := c.checkSpam(msg, text); action != nil {
		return action
	}

	return &ports.CheckerAction{Type: None}
}

func (c *Checker) checkBypass(msg *ports.ChatMessage) *ports.CheckerAction {
	if !c.cfg.Enabled || msg.Chatter.IsBroadcaster || msg.Chatter.IsMod || !c.cfg.Spam.SettingsDefault.Enabled ||
		(c.cfg.Spam.Mode == "online" && !c.stream.IsLive()) {
		return &ports.CheckerAction{Type: None}
	}

	if slices.Contains(c.cfg.Spam.WhitelistUsers, msg.Chatter.Username) {
		return &ports.CheckerAction{Type: None}
	}

	return nil
}

func (c *Checker) CheckBanwords(text, textOriginal string) *ports.CheckerAction {
	if !c.bwords.CheckMessage(text, textOriginal) {
		return nil
	}

	return &ports.CheckerAction{
		Type:   Ban,
		Reason: "банворд",
	}
}

func (c *Checker) CheckAds(text string, username string) *ports.CheckerAction {
	if !strings.Contains(text, "twitch.tv/") {
		return nil
	}

	if !strings.Contains(text, "twitch.tv/"+strings.ToLower(username)) &&
		!(strings.Contains(text, "подписывайтесь") || strings.Contains(text, "подпишитесь")) {
		return nil
	}

	return &ports.CheckerAction{
		Type:   Ban,
		Reason: "реклама",
	}
}

func (c *Checker) CheckMwords(text string) *ports.CheckerAction {
	words := c.regexp.SplitWordsBySpace(text)
	makeAction := func(action string, reason string, duration int) *ports.CheckerAction {
		return &ports.CheckerAction{
			Type:     action,
			Reason:   fmt.Sprintf("мворд (%s)", reason),
			Duration: time.Duration(duration) * time.Second,
		}
	}

	for _, group := range c.cfg.MwordGroup {
		if !group.Enabled {
			continue
		}

		for _, phrase := range group.Words {
			if phrase == "" {
				continue
			}

			if matchPhrase(words, phrase) {
				return makeAction(group.Action, phrase, group.Duration)
			}
		}

		for _, re := range group.Regexp {
			if re == nil {
				continue
			}

			if isMatch, _ := re.MatchString(text); isMatch {
				return makeAction(group.Action, re.String(), group.Duration)
			}
		}
	}

	for phrase, mw := range c.cfg.Mword {
		if phrase == "" {
			continue
		}

		if mw.Regexp != nil {
			if isMatch, _ := mw.Regexp.MatchString(text); isMatch {
				return makeAction(mw.Action, mw.Regexp.String(), mw.Duration)
			}
		}

		if matchPhrase(words, strings.ToLower(phrase)) {
			return makeAction(mw.Action, phrase, mw.Duration)
		}
	}
	return nil
}

func (c *Checker) checkWordLength(msg *ports.ChatMessage, words []string) *ports.CheckerAction {
	settings := c.cfg.Spam.SettingsDefault
	if msg.Chatter.IsVip {
		settings = c.cfg.Spam.SettingsVIP
		if !settings.Enabled {
			return &ports.CheckerAction{Type: None}
		}
	}

	for _, word := range words {
		if settings.MaxWordLength > 0 && len(word) >= settings.MaxWordLength && settings.MaxWordTimeoutTime > 0 {
			return &ports.CheckerAction{
				Type:     Timeout,
				Reason:   "превышена максимальная длина слова",
				Duration: time.Duration(settings.MaxWordTimeoutTime) * time.Second,
			}
		}
	}
	return nil
}

func (c *Checker) checkSpam(msg *ports.ChatMessage, text string) *ports.CheckerAction {
	settings := c.cfg.Spam.SettingsDefault
	if msg.Chatter.IsVip {
		settings = c.cfg.Spam.SettingsVIP
		if !settings.Enabled {
			return nil
		}
	}

	var countSpam, gap int
	c.messages.ForEach(msg.Chatter.Username, func(item *storage.Item[string]) {
		similarity := domain.JaccardSimilarity(text, item.Data)
		if similarity >= settings.SimilarityThreshold {
			if gap < settings.MinGapMessages {
				countSpam++
			}
			gap = 0
		} else {
			gap++
		}
	})

	if countSpam < settings.MessageLimit-1 {
		c.messages.Push(msg.Chatter.Username, text, time.Duration(c.cfg.Spam.CheckWindowSeconds)*time.Second)
		return nil
	}

	words := c.regexp.SplitWordsBySpace(text)
	var dur time.Duration
	isException := false

	for phrase, ex := range c.cfg.Spam.Exceptions {
		if ex.Regexp != nil {
			if isMatch, _ := ex.Regexp.MatchString(text); !isMatch {
				continue
			}
		}

		if !matchPhrase(words, strings.ToLower(phrase)) {
			continue
		}

		if countSpam < ex.MessageLimit-1 {
			c.messages.Push(msg.Chatter.Username, text, time.Duration(c.cfg.Spam.CheckWindowSeconds)*time.Second)
			return nil
		}
		dur = time.Duration(ex.Timeout) * time.Second
		isException = true
		break
	}

	if !isException {
		sec := domain.GetByIndexOrLast(settings.Timeouts, c.timeouts.Len(msg.Chatter.Username))
		dur = time.Duration(sec) * time.Second
		c.timeouts.Push(msg.Chatter.Username, Empty{}, time.Duration(settings.ResetTimeoutSeconds)*time.Second)
	}

	c.messages.CleanupUser(msg.Chatter.Username)
	return &ports.CheckerAction{
		Type:     Timeout,
		Reason:   "спам",
		Duration: dur,
	}
}

func matchPhrase(words []string, phrase string) bool {
	phraseParts := strings.Split(phrase, " ")
	if len(phraseParts) == 1 {
		for _, w := range words {
			if w == phrase {
				return true
			}
		}
		return false
	}

	for i := 0; i <= len(words)-len(phraseParts); i++ {
		match := true
		for j := 0; j < len(phraseParts); j++ {
			if words[i+j] != phraseParts[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
