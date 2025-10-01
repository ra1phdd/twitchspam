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
	timeouts StoreTimeouts
	messages ports.StorePort[storage.Message]
	sevenTV  ports.SevenTVPort
	template ports.TemplatePort
	irc      ports.IRCPort
}

type StoreTimeouts struct {
	spam             ports.StorePort[storage.Empty]
	emote            ports.StorePort[storage.Empty]
	exceptionsSpam   ports.StorePort[storage.Empty]
	exceptionsEmotes ports.StorePort[storage.Empty]
	mword            ports.StorePort[storage.Empty]
	mwordGroup       ports.StorePort[storage.Empty]
}

func NewCheck(log logger.Logger, cfg *config.Config, stream ports.StreamPort, template ports.TemplatePort, irc ports.IRCPort) *Checker {
	capacity := func() int {
		defLimit := float64(cfg.Spam.SettingsDefault.MessageLimit*cfg.Spam.SettingsDefault.MinGapMessages) / cfg.Spam.SettingsDefault.SimilarityThreshold
		vipLimit := float64(cfg.Spam.SettingsVIP.MessageLimit*cfg.Spam.SettingsVIP.MinGapMessages) / cfg.Spam.SettingsVIP.SimilarityThreshold

		return int(max(defLimit, vipLimit))
	}()

	return &Checker{
		log:    log,
		cfg:    cfg,
		stream: stream,
		timeouts: StoreTimeouts{
			spam:             storage.New[storage.Empty](15, time.Duration(cfg.Spam.CheckWindowSeconds)*time.Second),
			emote:            storage.New[storage.Empty](15, time.Duration(cfg.Spam.CheckWindowSeconds)*time.Second),
			exceptionsSpam:   storage.New[storage.Empty](15, time.Duration(cfg.Spam.CheckWindowSeconds)*time.Second),
			exceptionsEmotes: storage.New[storage.Empty](15, time.Duration(cfg.Spam.CheckWindowSeconds)*time.Second),
			mword:            storage.New[storage.Empty](15, time.Duration(cfg.Spam.CheckWindowSeconds)*time.Second),
			mwordGroup:       storage.New[storage.Empty](15, time.Duration(cfg.Spam.CheckWindowSeconds)*time.Second),
		},
		messages: storage.New[storage.Message](capacity, time.Duration(cfg.Spam.CheckWindowSeconds)*time.Second),
		sevenTV:  seventv.New(log, cfg, stream),
		template: template,
		irc:      irc,
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

	if action := c.CheckMwords(msg); action != nil {
		return action
	}

	c.messages.Push(msg.Chatter.Username, storage.Message{HashWords: domain.WordsToHashes(msg.Message.Text.WordsLowerNorm())})
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

func (c *Checker) CheckMwords(msg *ports.ChatMessage) *ports.CheckerAction {
	for word, mw := range c.cfg.Mword {
		if !c.matchMwordRule(msg, word, mw.Regexp, mw.Options) {
			continue
		}

		action, dur := domain.GetPunishment(mw.Punishments, c.timeouts.mword.Len(msg.Chatter.Username))
		c.timeouts.mword.Push(msg.Chatter.Username, storage.Empty{})

		return &ports.CheckerAction{
			Type:     action,
			Reason:   "мворд",
			Duration: dur,
		}
	}

	for _, mwg := range c.cfg.MwordGroup {
		if !mwg.Enabled {
			continue
		}

		var isMatch bool
		for _, word := range mwg.Words {
			if !c.matchMwordRule(msg, word, nil, mwg.Options) {
				continue
			}

			isMatch = true
			break
		}

		if !isMatch {
			for _, re := range mwg.Regexp {
				if !c.matchMwordRule(msg, "", re, mwg.Options) {
					continue
				}

				isMatch = true
				break
			}
		}

		if !isMatch {
			continue
		}

		action, dur := domain.GetPunishment(mwg.Punishments, c.timeouts.mwordGroup.Len(msg.Chatter.Username))
		c.timeouts.mwordGroup.Push(msg.Chatter.Username, storage.Empty{})

		return &ports.CheckerAction{
			Type:     action,
			Reason:   "мворд",
			Duration: dur,
		}
	}

	return nil
}

func (c *Checker) checkSpam(msg *ports.ChatMessage) *ports.CheckerAction {
	settings := c.cfg.Spam.SettingsDefault
	if msg.Chatter.IsVip {
		settings = c.cfg.Spam.SettingsVIP
	}

	if !settings.Enabled || !c.template.SpamPause().CanProcess() {
		return nil
	}

	if action := c.handleWordLength(msg.Message.Text.WordsLowerNorm(), settings); action != nil {
		return action
	}
	countSpam := c.countSpamMessages(msg, settings)

	if action := c.handleEmotes(msg, countSpam); action != nil {
		if action.Type != None {
			c.messages.ClearKey(msg.Chatter.Username)
		}
		return action
	}

	if action := c.handleExceptions(msg, countSpam); action != nil {
		if action.Type != None {
			c.messages.ClearKey(msg.Chatter.Username)
		}
		return action
	}

	if countSpam < settings.MessageLimit {
		return nil
	}

	action, dur := domain.GetPunishment(settings.Punishments, c.timeouts.spam.Len(msg.Chatter.Username))
	c.timeouts.spam.Push(msg.Chatter.Username, storage.Empty{})

	c.messages.ClearKey(msg.Chatter.Username)
	return &ports.CheckerAction{
		Type:     action,
		Reason:   "спам",
		Duration: dur,
	}
}

func (c *Checker) countSpamMessages(msg *ports.ChatMessage, settings config.SpamSettings) int {
	var countSpam, gap int
	hash := domain.WordsToHashes(msg.Message.Text.WordsLowerNorm())
	c.messages.ForEach(msg.Chatter.Username, func(item *storage.Message) {
		similarity := domain.JaccardHashSimilarity(hash, item.HashWords)
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

	action, dur := domain.GetPunishment(c.cfg.Spam.SettingsEmotes.Punishments, c.timeouts.emote.Len(msg.Chatter.Username))
	c.timeouts.emote.Push(msg.Chatter.Username, storage.Empty{})

	return &ports.CheckerAction{
		Type:     action,
		Reason:   "спам эмоутов",
		Duration: dur,
	}
}

func (c *Checker) handleEmotesExceptions(msg *ports.ChatMessage, countSpam int) *ports.CheckerAction {
	for word, ex := range c.cfg.Spam.SettingsEmotes.Exceptions {
		if !c.matchExceptRule(msg, word, ex.Regexp, ex.Options) {
			continue
		}

		if !ex.Enabled || countSpam < ex.MessageLimit {
			return &ports.CheckerAction{Type: None}
		}

		action, dur := domain.GetPunishment(ex.Punishments, c.timeouts.exceptionsEmotes.Len(msg.Chatter.Username))
		c.timeouts.exceptionsEmotes.Push(msg.Chatter.Username, storage.Empty{})

		return &ports.CheckerAction{
			Type:     action,
			Reason:   "спам",
			Duration: dur,
		}
	}

	return nil
}

func (c *Checker) handleExceptions(msg *ports.ChatMessage, countSpam int) *ports.CheckerAction {
	for word, ex := range c.cfg.Spam.Exceptions {
		if !c.matchExceptRule(msg, word, ex.Regexp, ex.Options) {
			continue
		}

		if !ex.Enabled || countSpam < ex.MessageLimit {
			return &ports.CheckerAction{Type: None}
		}

		action, dur := domain.GetPunishment(ex.Punishments, c.timeouts.exceptionsSpam.Len(msg.Chatter.Username))
		c.timeouts.exceptionsSpam.Push(msg.Chatter.Username, storage.Empty{})

		return &ports.CheckerAction{
			Type:     action,
			Reason:   "спам",
			Duration: dur,
		}
	}

	return nil
}

func (c *Checker) matchMwordRule(msg *ports.ChatMessage, word string, re *regexp.Regexp, opts config.MwordOptions) bool {
	if opts.NoVip && msg.Chatter.IsVip {
		return false
	}
	if opts.NoSub && msg.Chatter.IsSubscriber {
		return false
	}
	if opts.IsFirst {
		if isFirst, _ := c.irc.WaitForIRC(msg.Message.ID, 250*time.Millisecond); !isFirst {
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
		text = msg.Message.Text.Original
		words = msg.Message.Text.Words()
	case opts.NoRepeat:
		text = msg.Message.Text.Original
		words = msg.Message.Text.Words()
	case opts.CaseSensitive:
		text = msg.Message.Text.Normalized()
		words = msg.Message.Text.WordsNorm()
	default:
		text = msg.Message.Text.LowerNorm()
		words = msg.Message.Text.WordsLowerNorm()
	}

	return c.matchRule(word, re, text, words, opts.Contains)
}

func (c *Checker) matchExceptRule(msg *ports.ChatMessage, word string, re *regexp.Regexp, opts config.ExceptOptions) bool {
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
		text = msg.Message.Text.Original
		words = msg.Message.Text.Words()
	case opts.NoRepeat:
		text = msg.Message.Text.Original
		words = msg.Message.Text.Words()
	case opts.CaseSensitive:
		text = msg.Message.Text.Normalized()
		words = msg.Message.Text.WordsNorm()
	default:
		text = msg.Message.Text.LowerNorm()
		words = msg.Message.Text.WordsLowerNorm()
	}

	return c.matchRule(word, re, text, words, opts.Contains)
}

func (c *Checker) matchRule(word string, re *regexp.Regexp, text string, words []string, contains bool) bool {
	if re != nil {
		return re.MatchString(text)
	}

	if word == "" {
		return false
	}

	if contains || strings.Contains(word, " ") {
		return strings.Contains(text, word)
	}
	return slices.Contains(words, word)
}
