package checker

import (
	"net/http"
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
	log      logger.Logger
	cfg      *config.Config
	stream   ports.StreamPort
	sevenTV  ports.SevenTVPort
	template ports.TemplatePort

	messages ports.StorePort[storage.Message]
	timeouts ports.StorePort[int]
}

func NewCheck(log logger.Logger, cfg *config.Config, stream ports.StreamPort, template ports.TemplatePort, messages ports.StorePort[storage.Message], timeouts ports.StorePort[int], client *http.Client) *Checker {
	return &Checker{
		log:      log,
		cfg:      cfg,
		stream:   stream,
		sevenTV:  seventv.New(log, cfg, stream, client),
		template: template,
		messages: messages,
		timeouts: timeouts,
	}
}

func (c *Checker) Check(msg *domain.ChatMessage, checkSpam bool) *ports.CheckerAction {
	if action := c.checkBypass(msg); action != nil {
		return action
	}

	if action := c.template.Nuke().Check(&msg.Message.Text, false); action != nil {
		return action
	}

	if !c.cfg.Enabled {
		return &ports.CheckerAction{Type: None}
	}

	if action := c.checkBanwords(msg.Message.Text.Text(domain.LowerOption, domain.RemovePunctuationOption, domain.RemoveDuplicateLettersOption), msg.Message.Text.Words()); action != nil {
		return action
	}

	if action := c.checkAds(msg.Message.Text.Text(domain.LowerOption), msg.Chatter.Username); action != nil {
		return action
	}

	if action := c.checkMwords(msg); action != nil {
		return action
	}

	if checkSpam {
		if action := c.checkSpam(msg); action != nil {
			return action
		}
	}

	return &ports.CheckerAction{Type: None}
}

func (c *Checker) checkBypass(msg *domain.ChatMessage) *ports.CheckerAction {
	if msg.Chatter.IsBroadcaster || msg.Chatter.IsMod {
		return &ports.CheckerAction{Type: None}
	}

	if _, ok := c.cfg.Spam.WhitelistUsers[msg.Chatter.Username]; ok {
		return &ports.CheckerAction{Type: None}
	}

	return nil
}

func (c *Checker) checkBanwords(textLower string, wordsOriginal []string) *ports.CheckerAction {
	if !c.template.Banwords().CheckMessage(textLower, wordsOriginal) {
		return nil
	}

	return &ports.CheckerAction{
		Type:   Ban,
		Reason: "банворд",
	}
}

func (c *Checker) checkAds(text string, username string) *ports.CheckerAction {
	if !strings.Contains(text, "twitch.tv/") ||
		strings.Contains(text, "twitch.tv/"+strings.ToLower(c.stream.ChannelName())) {
		return nil
	}

	if !strings.Contains(text, "twitch.tv/"+strings.ToLower(username)) &&
		!strings.Contains(text, "подписывайтесь") &&
		!strings.Contains(text, "подпишитесь") {
		return nil
	}

	return &ports.CheckerAction{
		Type:   Ban,
		Reason: "реклама",
	}
}

func (c *Checker) checkMwords(msg *domain.ChatMessage) *ports.CheckerAction {
	punishments := c.template.Mword().Check(msg, c.stream.IsLive())
	if len(punishments) == 0 {
		return nil
	}

	countTimeouts, ok := c.timeouts.Get(msg.Chatter.Username, "mword")
	if !ok {
		c.timeouts.Push(msg.Chatter.Username, "mword", 0, storage.WithTTL(
			time.Duration(c.cfg.Spam.SettingsDefault.DurationResetPunishments)*time.Second),
		)
	}
	action, dur := c.template.Punishment().Get(punishments, countTimeouts)
	c.timeouts.Update(msg.Chatter.Username, "mword", func(cur int, exists bool) int {
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

	if ((c.cfg.Spam.Mode == config.OnlineMode || c.cfg.Spam.Mode == 0) && !c.stream.IsLive()) ||
		(c.cfg.Spam.Mode == config.OfflineMode && c.stream.IsLive()) {
		return nil
	}

	if action := c.handleWordLength(msg.Message.Text.Words(domain.LowerOption, domain.RemovePunctuationOption, domain.RemoveDuplicateLettersOption), settings); action != nil {
		return action
	}
	countSpam, _ := c.calculateSpamMessages(msg, settings)

	if action := c.handleEmotes(msg, countSpam); action != nil {
		if action.Type != None {
			c.messages.ClearKey(msg.Chatter.Username)
		}
		return action
	}

	if action := c.handleExceptions(msg, countSpam, "default"); action != nil {
		if action.Type != None {
			c.messages.ClearKey(msg.Chatter.Username)
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

	countTimeouts, ok := c.timeouts.Get(msg.Chatter.Username, cacheKey)
	if !ok {
		c.timeouts.Push(msg.Chatter.Username, cacheKey, 0, storage.WithTTL(cacheTTL))
	}
	action, dur := c.template.Punishment().Get(settings.Punishments, countTimeouts)
	c.timeouts.Update(msg.Chatter.Username, cacheKey, func(cur int, exists bool) int {
		if !exists {
			return 1
		}
		return cur + 1
	})

	c.messages.ClearKey(msg.Chatter.Username)
	return &ports.CheckerAction{
		Type:     action,
		Reason:   "спам",
		Duration: dur,
	}
}

func (c *Checker) calculateSpamMessages(msg *domain.ChatMessage, settings config.SpamSettings) (int, time.Duration) {
	var countSpam, gap int
	var timestamps []time.Time

	hash := domain.WordsToHashes(msg.Message.Text.Words(domain.RemovePunctuationOption))
	c.messages.ForEach(msg.Chatter.Username, func(item *storage.Message) {
		if item.IgnoreAntispam {
			return
		}

		similarity := domain.JaccardHashSimilarity(hash, item.HashWordsLowerNorm)
		if similarity >= settings.SimilarityThreshold {
			_, isOnlyEmotes := c.sevenTV.EmoteStats(item.Data.Message.Text.Words(domain.RemovePunctuationOption))
			if isOnlyEmotes || item.Data.Message.EmoteOnly {
				return
			}

			if gap < settings.MinGapMessages {
				countSpam++
				timestamps = append(timestamps, item.Time)
			}
			gap = 0
		} else {
			gap++
		}
	})

	if len(timestamps) < 2 {
		return countSpam, 0
	}

	var total time.Duration
	for i := 1; i < len(timestamps); i++ {
		total += timestamps[i].Sub(timestamps[i-1])
	}

	avgGap := total / time.Duration(len(timestamps)-1)
	return countSpam, avgGap
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
	count, isOnlyEmotes := c.sevenTV.EmoteStats(msg.Message.Text.Words(domain.RemovePunctuationOption))

	emoteOnly := msg.Message.EmoteOnly || isOnlyEmotes
	if !emoteOnly {
		return nil
	}

	if !c.cfg.Spam.SettingsEmotes.Enabled {
		return &ports.CheckerAction{Type: None}
	}

	if action := c.handleExceptions(msg, countSpam, "emote"); action != nil {
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

	countTimeouts, ok := c.timeouts.Get(msg.Chatter.Username, "spam_emote")
	if !ok {
		c.timeouts.Push(msg.Chatter.Username, "spam_emote", 0, storage.WithTTL(
			time.Duration(c.cfg.Spam.SettingsDefault.DurationResetPunishments)*time.Second),
		)
	}
	action, dur := c.template.Punishment().Get(c.cfg.Spam.SettingsEmotes.Punishments, countTimeouts)
	c.timeouts.Update(msg.Chatter.Username, "spam_emote", func(cur int, exists bool) int {
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

func (c *Checker) handleExceptions(msg *domain.ChatMessage, countSpam int, typeSpam string) *ports.CheckerAction {
	exceptions, subKey := c.cfg.Spam.Exceptions, "except_spam"
	if typeSpam == "emote" {
		exceptions, subKey = c.cfg.Spam.SettingsEmotes.Exceptions, "except_emote"
	}

	for word, ex := range exceptions {
		if !c.matchExceptRule(msg, word, ex.Regexp, ex.Options) {
			continue
		}

		if !ex.Enabled || countSpam < ex.MessageLimit {
			return &ports.CheckerAction{Type: None}
		}

		countTimeouts, ok := c.timeouts.Get(msg.Chatter.Username, subKey)
		if !ok {
			c.timeouts.Push(msg.Chatter.Username, subKey, 0, storage.WithTTL(
				time.Duration(c.cfg.Spam.SettingsDefault.DurationResetPunishments)*time.Second),
			)
		}
		action, dur := c.template.Punishment().Get(ex.Punishments, countTimeouts)
		c.timeouts.Update(msg.Chatter.Username, subKey, func(cur int, exists bool) int {
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
	if opts.OneWord && len(msg.Message.Text.Words(domain.LowerOption, domain.RemovePunctuationOption, domain.RemoveDuplicateLettersOption)) > 1 {
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
		text = msg.Message.Text.Text(domain.RemovePunctuationOption, domain.RemoveDuplicateLettersOption)
		words = msg.Message.Text.Words(domain.RemovePunctuationOption, domain.RemoveDuplicateLettersOption)
	default:
		text = msg.Message.Text.Text(domain.LowerOption, domain.RemovePunctuationOption, domain.RemoveDuplicateLettersOption)
		words = msg.Message.Text.Words(domain.LowerOption, domain.RemovePunctuationOption, domain.RemoveDuplicateLettersOption)
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
