package checker

import (
	"fmt"
	"runtime"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/seventv"
	"twitchspam/internal/app/domain"
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
	timeouts struct {
		spam             ports.StorePort[Empty]
		emote            ports.StorePort[Empty]
		exceptionsSpam   ports.StorePort[Empty]
		exceptionsEmotes ports.StorePort[Empty]
		mword            ports.StorePort[Empty]
		mwordGroup       ports.StorePort[Empty]
	}
	messages ports.StorePort[string]
	sevenTV  ports.SevenTVPort
	irc      ports.IRCPort
	template ports.TemplatePort
}

func NewCheck(log logger.Logger, cfg *config.Config, stream ports.StreamPort, stats ports.StatsPort, irc ports.IRCPort, template ports.TemplatePort) *Checker {
	return &Checker{
		log:    log,
		cfg:    cfg,
		stream: stream,
		stats:  stats,
		timeouts: struct {
			spam             ports.StorePort[Empty]
			emote            ports.StorePort[Empty]
			exceptionsSpam   ports.StorePort[Empty]
			exceptionsEmotes ports.StorePort[Empty]
			mword            ports.StorePort[Empty]
			mwordGroup       ports.StorePort[Empty]
		}{
			spam:             storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int { return 15 }),
			emote:            storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int { return 15 }),
			exceptionsSpam:   storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int { return 15 }),
			exceptionsEmotes: storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int { return 15 }),
			mword:            storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int { return 15 }),
			mwordGroup:       storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int { return 15 }),
		},
		messages: storage.New[string](runtime.NumCPU(), 500*time.Millisecond, func() int {
			defLimit := float64(cfg.Spam.SettingsDefault.MessageLimit*cfg.Spam.SettingsDefault.MinGapMessages) / cfg.Spam.SettingsDefault.SimilarityThreshold
			vipLimit := float64(cfg.Spam.SettingsVIP.MessageLimit*cfg.Spam.SettingsVIP.MinGapMessages) / cfg.Spam.SettingsVIP.SimilarityThreshold

			return int(max(defLimit, vipLimit))
		}),
		sevenTV:  seventv.New(log, stream),
		irc:      irc,
		template: template,
	}
}

func (c *Checker) Check(msg *ports.ChatMessage) *ports.CheckerAction {
	if action := c.checkBypass(msg); action != nil {
		return action
	}

	if action := c.CheckBanwords(msg.Message.Text.LowerNorm(), msg.Message.Text.Original); action != nil {
		return action
	}

	if action := c.CheckAds(msg.Message.Text.Lower(), msg.Chatter.Username); action != nil {
		return action
	}

	if action := c.CheckMwords(msg.Message.Text.LowerNorm(), msg.Chatter.Username, msg.Message.Text.WordsLowerNorm()); action != nil {
		return action
	}

	if action := c.CheckMwords(msg.Message.Text.Original, msg.Chatter.Username, msg.Message.Text.Words()); action != nil {
		return action
	}

	c.messages.Push(msg.Chatter.Username, msg.Message.Text.LowerNorm(), time.Duration(c.cfg.Spam.CheckWindowSeconds)*time.Second)
	if action := c.checkSpam(msg); action != nil {
		return action
	}

	return &ports.CheckerAction{Type: None}
}

func (c *Checker) checkBypass(msg *ports.ChatMessage) *ports.CheckerAction {
	if !c.cfg.Enabled || msg.Chatter.IsBroadcaster || msg.Chatter.IsMod ||
		(c.cfg.Spam.Mode == "online" && !c.stream.IsLive()) {
		return &ports.CheckerAction{Type: None}
	}

	if _, ok := c.cfg.Spam.WhitelistUsers[msg.Chatter.Username]; ok {
		return &ports.CheckerAction{Type: None}
	}

	return nil
}

func (c *Checker) CheckBanwords(text, textOriginal string) *ports.CheckerAction {
	if !c.template.CheckOnBanwords(text, textOriginal) {
		return nil
	}

	return &ports.CheckerAction{
		Type:   Ban,
		Reason: "банворд",
	}
}

func (c *Checker) CheckAds(text string, username string) *ports.CheckerAction {
	if !strings.Contains(text, "twitch.tv/") ||
		strings.Contains(text, "twitch.tv/"+strings.ToLower(c.stream.ChannelName())) {
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

func (c *Checker) CheckMwords(text, username string, words []string) *ports.CheckerAction {
	makeAction := func(punishments []config.Punishment, reason string, countPunishments int) *ports.CheckerAction {
		action, dur := domain.GetPunishment(punishments, countPunishments)
		return &ports.CheckerAction{
			Type:     action,
			Reason:   fmt.Sprintf("мворд (%s)", reason),
			Duration: dur,
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

			if c.template.MatchPhrase(words, phrase) {
				return makeAction(group.Punishments, phrase, c.timeouts.mwordGroup.Len(username))
			}
		}

		for _, re := range group.Regexp {
			if re == nil {
				continue
			}

			if isMatch, _ := re.MatchString(text); isMatch {
				return makeAction(group.Punishments, "регулярное выражение", c.timeouts.mwordGroup.Len(username))
			}
		}
	}

	for phrase, mw := range c.cfg.Mword {
		if phrase == "" {
			continue
		}

		if mw.Regexp != nil {
			if isMatch, _ := mw.Regexp.MatchString(text); isMatch {
				return makeAction(mw.Punishments, "регулярное выражение", c.timeouts.mword.Len(username))
			}
		}

		if c.template.MatchPhrase(words, strings.ToLower(phrase)) {
			return makeAction(mw.Punishments, phrase, c.timeouts.mword.Len(username))
		}
	}
	return nil
}

func (c *Checker) checkSpam(msg *ports.ChatMessage) *ports.CheckerAction {
	settings := c.cfg.Spam.SettingsDefault
	if msg.Chatter.IsVip {
		settings = c.cfg.Spam.SettingsVIP
	}

	if !settings.Enabled {
		return nil
	}

	if action := c.handleWordLength(msg.Message.Text.WordsLowerNorm(), settings); action != nil {
		return action
	}
	countSpam := c.countSpamMessages(msg, settings)

	if action := c.handleEmotes(msg, countSpam); action != nil {
		if action.Type != None {
			c.messages.CleanupUser(msg.Chatter.Username)
		}
		return action
	}

	if action := c.handleExceptions(msg, countSpam); action != nil {
		if action.Type != None {
			c.messages.CleanupUser(msg.Chatter.Username)
		}
		return action
	}

	if countSpam <= settings.MessageLimit-1 {
		return nil
	}

	action, dur := domain.GetPunishment(settings.Punishments, c.timeouts.spam.Len(msg.Chatter.Username))
	c.timeouts.spam.Push(msg.Chatter.Username, Empty{}, time.Duration(settings.DurationResetPunishments)*time.Second)

	c.messages.CleanupUser(msg.Chatter.Username)
	return &ports.CheckerAction{
		Type:     action,
		Reason:   "спам",
		Duration: dur,
	}
}

func (c *Checker) countSpamMessages(msg *ports.ChatMessage, settings config.SpamSettings) int {
	var countSpam, gap int
	c.messages.ForEach(msg.Chatter.Username, func(item *storage.Item[string]) {
		similarity := domain.JaccardSimilarity(msg.Message.Text.LowerNorm(), item.Data)
		if similarity >= settings.SimilarityThreshold {
			if gap < settings.MinGapMessages {
				countSpam++
			}
			gap = 0
		} else {
			gap++
		}
	})
	return countSpam
}

func (c *Checker) handleWordLength(words []string, settings config.SpamSettings) *ports.CheckerAction {
	for _, word := range words {
		if settings.MaxWordLength > 0 && len([]rune(word)) >= settings.MaxWordLength {
			return &ports.CheckerAction{
				Type:     settings.MaxWordPunishment.Action,
				Reason:   "превышена максимальная длина слова",
				Duration: time.Duration(settings.MaxWordPunishment.Duration) * time.Second,
			}
		}
	}

	return nil
}

func (c *Checker) handleEmotes(msg *ports.ChatMessage, countSpam int) *ports.CheckerAction {
	if !c.cfg.Spam.SettingsEmotes.Enabled && (c.sevenTV.IsOnlyEmotes(msg.Message.Text.Original) || msg.Message.EmoteOnly) {
		return &ports.CheckerAction{Type: None}
	}

	if !c.sevenTV.IsOnlyEmotes(msg.Message.Text.Original) || !msg.Message.EmoteOnly {
		return nil
	}
	words := msg.Message.Text.WordsLowerNorm()

	if action := c.handleEmotesExceptions(msg, countSpam); action != nil {
		return action
	}

	if countSpam <= c.cfg.Spam.SettingsEmotes.MessageLimit-1 {
		return &ports.CheckerAction{Type: None}
	}

	if c.cfg.Spam.SettingsEmotes.MaxEmotesLength > 0 && c.sevenTV.CountEmotes(words) >= c.cfg.Spam.SettingsEmotes.MaxEmotesLength {
		return &ports.CheckerAction{
			Type:     c.cfg.Spam.SettingsEmotes.MaxEmotesPunishment.Action,
			Reason:   "превышено максимальное кол-во эмоутов в сообщении",
			Duration: time.Duration(c.cfg.Spam.SettingsEmotes.MaxEmotesPunishment.Duration) * time.Second,
		}
	}

	action, dur := domain.GetPunishment(c.cfg.Spam.SettingsEmotes.Punishments, c.timeouts.emote.Len(msg.Chatter.Username))
	c.timeouts.emote.Push(msg.Chatter.Username, Empty{}, time.Duration(c.cfg.Spam.SettingsEmotes.DurationResetPunishments)*time.Second)

	return &ports.CheckerAction{
		Type:     action,
		Reason:   "спам эмоутов",
		Duration: dur,
	}
}

func (c *Checker) handleEmotesExceptions(msg *ports.ChatMessage, countSpam int) *ports.CheckerAction {
	for phrase, ex := range c.cfg.Spam.SettingsEmotes.Exceptions {
		if ex.Regexp != nil {
			if isMatch, _ := ex.Regexp.MatchString(msg.Message.Text.LowerNorm()); !isMatch {
				continue
			}
		}

		if !c.template.MatchPhrase(msg.Message.Text.WordsLowerNorm(), strings.ToLower(phrase)) {
			continue
		}

		if countSpam <= ex.MessageLimit-1 {
			return &ports.CheckerAction{Type: None}
		}

		action, dur := domain.GetPunishment(ex.Punishments, c.timeouts.exceptionsEmotes.Len(msg.Chatter.Username))
		c.timeouts.exceptionsEmotes.Push(msg.Chatter.Username, Empty{}, time.Duration(c.cfg.Spam.SettingsEmotes.DurationResetPunishments)*time.Second)

		return &ports.CheckerAction{
			Type:     action,
			Reason:   "спам",
			Duration: dur,
		}
	}

	return nil
}

func (c *Checker) handleExceptions(msg *ports.ChatMessage, countSpam int) *ports.CheckerAction {
	for phrase, ex := range c.cfg.Spam.Exceptions {
		if ex.Regexp != nil {
			if isMatch, _ := ex.Regexp.MatchString(msg.Message.Text.LowerNorm()); !isMatch {
				continue
			}
		}

		if !c.template.MatchPhrase(msg.Message.Text.WordsLowerNorm(), strings.ToLower(phrase)) {
			continue
		}

		if countSpam <= ex.MessageLimit-1 {
			return &ports.CheckerAction{Type: None}
		}

		action, dur := domain.GetPunishment(ex.Punishments, c.timeouts.exceptionsSpam.Len(msg.Chatter.Username))
		c.timeouts.exceptionsSpam.Push(msg.Chatter.Username, Empty{}, time.Duration(c.cfg.Spam.SettingsDefault.DurationResetPunishments)*time.Second)

		return &ports.CheckerAction{
			Type:     action,
			Reason:   "спам",
			Duration: dur,
		}
	}

	return nil
}
