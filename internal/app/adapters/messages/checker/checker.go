package checker

import (
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

	if action := c.CheckBanwords(msg.Message.Text.LowerNorm(), msg.Message.Text.Words()); action != nil {
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

func (c *Checker) CheckBanwords(textLower string, wordsOriginal []string) *ports.CheckerAction {
	if !c.template.CheckOnBanwords(textLower, wordsOriginal) {
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
	found, punishments, _ := c.template.MatchMwords(text, words)
	if !found {
		return nil
	}

	action, dur := domain.GetPunishment(punishments, c.timeouts.mwordGroup.Len(username))
	return &ports.CheckerAction{
		Type:     action,
		Reason:   "мворд",
		Duration: dur,
	}
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
	emoteOnly := c.sevenTV.IsOnlyEmotes(msg.Message.Text.Original) || msg.Message.EmoteOnly

	if !c.cfg.Spam.SettingsEmotes.Enabled && emoteOnly {
		return &ports.CheckerAction{Type: None}
	}

	if !emoteOnly {
		return nil
	}

	if action := c.handleEmotesExceptions(msg, countSpam); action != nil {
		return action
	}

	maxLen := c.cfg.Spam.SettingsEmotes.MaxEmotesLength
	if maxLen > 0 {
		emoteCount := max(len(msg.Message.Emotes), c.sevenTV.CountEmotes(msg.Message.Text.Words()))
		if emoteCount >= maxLen {
			p := c.cfg.Spam.SettingsEmotes.MaxEmotesPunishment
			return &ports.CheckerAction{
				Type:     p.Action,
				Reason:   "превышено максимальное кол-во эмоутов в сообщении",
				Duration: time.Duration(p.Duration) * time.Second,
			}
		}
	}

	if countSpam <= c.cfg.Spam.SettingsEmotes.MessageLimit-1 {
		return &ports.CheckerAction{Type: None}
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
