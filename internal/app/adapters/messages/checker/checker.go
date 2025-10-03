package checker

import (
	"regexp"
	"slices"
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
	sevenTV  ports.SevenTVPort
	template ports.TemplatePort
	irc      ports.IRCPort
}

func NewCheck(log logger.Logger, cfg *config.Config, stream ports.StreamPort, template ports.TemplatePort, irc ports.IRCPort) *Checker {
	return &Checker{
		log:      log,
		cfg:      cfg,
		stream:   stream,
		sevenTV:  seventv.New(log, cfg, stream),
		template: template,
		irc:      irc,
	}
}

func (c *Checker) Check(msg *domain.ChatMessage) *ports.CheckerAction {
	if action := c.checkBypass(msg); action != nil {
		return action
	}

	if action := c.CheckBanwords(msg.Message.Text.Text(domain.Lower, domain.RemovePunctuation, domain.RemoveDuplicateLetters), msg.Message.Text.Words()); action != nil {
		return action
	}

	if action := c.CheckAds(msg.Message.Text.Text(domain.Lower), msg.Chatter.Username); action != nil {
		return action
	}

	if action := c.CheckMwords(msg); action != nil {
		return action
	}

	if action := c.checkSpam(msg); action != nil {
		return action
	}

	return &ports.CheckerAction{Type: None}
}

func (c *Checker) checkBypass(msg *domain.ChatMessage) *ports.CheckerAction {
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
	if !c.template.Banwords().CheckMessage(textLower, wordsOriginal) {
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

func (c *Checker) CheckMwords(msg *domain.ChatMessage) *ports.CheckerAction {
	punishments := c.template.Mword().Check(msg)
	if punishments == nil || len(punishments) == 0 {
		return nil
	}

	countTimeouts, ok := c.template.Store().Timeouts().Get(msg.Chatter.Username, "mword")
	if !ok {
		c.template.Store().Timeouts().Push(msg.Chatter.Username, "mword", 0, storage.WithTTL(
			time.Duration(c.cfg.Spam.SettingsDefault.DurationResetPunishments)*time.Second),
		)
	}
	action, dur := domain.GetPunishment(punishments, countTimeouts)
	c.template.Store().Timeouts().Update(msg.Chatter.Username, "mword", func(cur int, exists bool) int {
		if !exists {
			return 1
		}
		return cur + 1
	})

	return &ports.CheckerAction{
		Type:     action,
		Reason:   "мворд",
		Duration: dur,
	}
}

func (c *Checker) checkSpam(msg *domain.ChatMessage) *ports.CheckerAction {
	settings := c.cfg.Spam.SettingsDefault
	if msg.Chatter.IsVip {
		settings = c.cfg.Spam.SettingsVIP
	}

	if !settings.Enabled || !c.template.SpamPause().CanProcess() {
		return nil
	}

	if action := c.handleWordLength(msg.Message.Text.Words(domain.Lower, domain.RemovePunctuation, domain.RemoveDuplicateLetters), settings); action != nil {
		return action
	}
	countSpam := c.countSpamMessages(msg, settings)

	if action := c.handleEmotes(msg, countSpam); action != nil {
		if action.Type != None {
			c.template.Store().Messages().ClearKey(msg.Chatter.Username)
		}
		return action
	}

	if action := c.handleExceptions(msg, countSpam); action != nil {
		if action.Type != None {
			c.template.Store().Messages().ClearKey(msg.Chatter.Username)
		}
		return action
	}

	if countSpam < settings.MessageLimit {
		return nil
	}

	cacheKey := "spam_default"
	cacheTTL := time.Duration(c.cfg.Spam.SettingsDefault.DurationResetPunishments) * time.Second
	if msg.Chatter.IsVip {
		cacheKey = "spam_vip"
		cacheTTL = time.Duration(c.cfg.Spam.SettingsVIP.DurationResetPunishments) * time.Second
	}

	countTimeouts, ok := c.template.Store().Timeouts().Get(msg.Chatter.Username, cacheKey)
	if !ok {
		c.template.Store().Timeouts().Push(msg.Chatter.Username, cacheKey, 0, storage.WithTTL(cacheTTL))
	}
	action, dur := domain.GetPunishment(settings.Punishments, countTimeouts)
	c.template.Store().Timeouts().Update(msg.Chatter.Username, cacheKey, func(cur int, exists bool) int {
		if !exists {
			return 1
		}
		return cur + 1
	})

	c.template.Store().Messages().ClearKey(msg.Chatter.Username)
	return &ports.CheckerAction{
		Type:     action,
		Reason:   "спам",
		Duration: dur,
	}
}

func (c *Checker) countSpamMessages(msg *domain.ChatMessage, settings config.SpamSettings) int {
	var countSpam, gap int
	hash := domain.WordsToHashes(msg.Message.Text.Words(domain.Lower, domain.RemovePunctuation, domain.RemoveDuplicateLetters))
	c.template.Store().Messages().ForEach(msg.Chatter.Username, func(item *storage.Message) {
		if item.IgnoreAntispam {
			return
		}

		similarity := domain.JaccardHashSimilarity(hash, item.HashWordsLowerNorm)
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

func (c *Checker) handleEmotes(msg *domain.ChatMessage, countSpam int) *ports.CheckerAction {
	count, isOnlyEmotes := c.sevenTV.EmoteStats(msg.Message.Text.Words())

	emoteOnly := msg.Message.EmoteOnly || isOnlyEmotes
	if !emoteOnly {
		return nil
	}

	if !c.cfg.Spam.SettingsEmotes.Enabled {
		return &ports.CheckerAction{Type: None}
	}

	if action := c.handleEmotesExceptions(msg, countSpam); action != nil {
		return action
	}

	if c.cfg.Spam.SettingsEmotes.MaxEmotesLength > 0 {
		emoteCount := max(len(msg.Message.Emotes), count)
		if emoteCount >= c.cfg.Spam.SettingsEmotes.MaxEmotesLength {
			return &ports.CheckerAction{
				Type:     c.cfg.Spam.SettingsEmotes.MaxEmotesPunishment.Action,
				Reason:   "превышено максимальное кол-во эмоутов в сообщении",
				Duration: time.Duration(c.cfg.Spam.SettingsEmotes.MaxEmotesPunishment.Duration) * time.Second,
			}
		}
	}

	if countSpam < c.cfg.Spam.SettingsEmotes.MessageLimit {
		return &ports.CheckerAction{Type: None}
	}

	countTimeouts, ok := c.template.Store().Timeouts().Get(msg.Chatter.Username, "spam_emote")
	if !ok {
		c.template.Store().Timeouts().Push(msg.Chatter.Username, "spam_emote", 0, storage.WithTTL(
			time.Duration(c.cfg.Spam.SettingsDefault.DurationResetPunishments)*time.Second),
		)
	}
	action, dur := domain.GetPunishment(c.cfg.Spam.SettingsEmotes.Punishments, countTimeouts)
	c.template.Store().Timeouts().Update(msg.Chatter.Username, "spam_emote", func(cur int, exists bool) int {
		if !exists {
			return 1
		}
		return cur + 1
	})

	return &ports.CheckerAction{
		Type:     action,
		Reason:   "спам эмоутов",
		Duration: dur,
	}
}

func (c *Checker) handleEmotesExceptions(msg *domain.ChatMessage, countSpam int) *ports.CheckerAction {
	for word, ex := range c.cfg.Spam.SettingsEmotes.Exceptions {
		if !c.matchExceptRule(msg, word, ex.Regexp, ex.Options) {
			continue
		}

		if !ex.Enabled || countSpam < ex.MessageLimit {
			return &ports.CheckerAction{Type: None}
		}

		countTimeouts, ok := c.template.Store().Timeouts().Get(msg.Chatter.Username, "except_emote")
		if !ok {
			c.template.Store().Timeouts().Push(msg.Chatter.Username, "except_emote", 0, storage.WithTTL(
				time.Duration(c.cfg.Spam.SettingsEmotes.DurationResetPunishments)*time.Second),
			)
		}
		action, dur := domain.GetPunishment(ex.Punishments, countTimeouts)
		c.template.Store().Timeouts().Update(msg.Chatter.Username, "except_emote", func(cur int, exists bool) int {
			if !exists {
				return 1
			}
			return cur + 1
		})

		return &ports.CheckerAction{
			Type:     action,
			Reason:   "спам",
			Duration: dur,
		}
	}

	return nil
}

func (c *Checker) handleExceptions(msg *domain.ChatMessage, countSpam int) *ports.CheckerAction {
	for word, ex := range c.cfg.Spam.Exceptions {
		if !c.matchExceptRule(msg, word, ex.Regexp, ex.Options) {
			continue
		}

		if !ex.Enabled || countSpam < ex.MessageLimit {
			return &ports.CheckerAction{Type: None}
		}

		countTimeouts, ok := c.template.Store().Timeouts().Get(msg.Chatter.Username, "except_spam")
		if !ok {
			c.template.Store().Timeouts().Push(msg.Chatter.Username, "except_spam", 0, storage.WithTTL(
				time.Duration(c.cfg.Spam.SettingsDefault.DurationResetPunishments)*time.Second),
			)
		}
		action, dur := domain.GetPunishment(ex.Punishments, countTimeouts)
		c.template.Store().Timeouts().Update(msg.Chatter.Username, "except_spam", func(cur int, exists bool) int {
			if !exists {
				return 1
			}
			return cur + 1
		})

		return &ports.CheckerAction{
			Type:     action,
			Reason:   "спам",
			Duration: dur,
		}
	}

	return nil
}

func (c *Checker) matchExceptRule(msg *domain.ChatMessage, word string, re *regexp.Regexp, opts config.ExceptOptions) bool {
	if opts.NoVip && msg.Chatter.IsVip {
		return false
	}
	if opts.NoSub && msg.Chatter.IsSubscriber {
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
