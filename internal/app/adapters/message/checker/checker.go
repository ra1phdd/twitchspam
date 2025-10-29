package checker

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/adapters/seventv"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/domain/message"
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

func (c *Checker) Check(msg *message.ChatMessage, checkSpam bool) *ports.CheckerAction {
	c.log.Trace("Checker.Check called",
		slog.String("user", msg.Chatter.Username),
		slog.String("message", msg.Message.Text.Text()),
		slog.Bool("check_spam", checkSpam),
	)

	if action := c.checkBypass(msg); action != nil {
		c.log.Debug("User bypassed check", slog.String("user", msg.Chatter.Username))
		return action
	}

	startProcessing := time.Now()
	if action := c.template.Nuke().Check(&msg.Message.Text, false); action != nil {
		c.log.Info("Message triggered nuke",
			slog.String("user", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
			slog.Any("action", action),
		)
		return action
	}
	endProcessing := time.Since(startProcessing).Seconds()
	metrics.ModulesProcessingTime.With(prometheus.Labels{"module": "nuke"}).Observe(endProcessing)

	if !c.cfg.Channels[msg.Broadcaster.Login].Enabled {
		c.log.Debug("Bot disabled, skipping message",
			slog.String("username", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
		)
		return &ports.CheckerAction{Type: None}
	}

	startProcessing = time.Now()
	if action := c.checkBanwords(msg); action != nil {
		c.log.Info("Message contains banword",
			slog.String("user", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
			slog.Any("action", action),
		)
		return action
	}
	endProcessing = time.Since(startProcessing).Seconds()
	metrics.ModulesProcessingTime.With(prometheus.Labels{"module": "banwords"}).Observe(endProcessing)

	startProcessing = time.Now()
	if action := c.checkAds(msg.Message.Text.Text(message.LowerOption), msg.Chatter.Username); action != nil {
		c.log.Info("Message detected as advertisement",
			slog.String("user", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
			slog.Any("action", action),
		)
		return action
	}
	endProcessing = time.Since(startProcessing).Seconds()
	metrics.ModulesProcessingTime.With(prometheus.Labels{"module": "ads"}).Observe(endProcessing)

	startProcessing = time.Now()
	if action := c.checkMwords(msg); action != nil {
		c.log.Info("Message contains muteword",
			slog.String("user", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
			slog.Any("action", action),
		)
		return action
	}
	endProcessing = time.Since(startProcessing).Seconds()
	metrics.ModulesProcessingTime.With(prometheus.Labels{"module": "mwords"}).Observe(endProcessing)

	if checkSpam {
		startProcessing = time.Now()
		if action := c.checkSpam(msg); action != nil {
			if action.Type != None {
				c.log.Info("Message flagged as spam",
					slog.String("user", msg.Chatter.Username),
					slog.String("message", msg.Message.Text.Text()),
					slog.Any("action", action),
				)
			}
			return action
		}
		endProcessing = time.Since(startProcessing).Seconds()
		metrics.ModulesProcessingTime.With(prometheus.Labels{"module": "spam"}).Observe(endProcessing)
	}

	c.log.Debug("No violations detected, skipping", slog.String("user", msg.Chatter.Username))
	return &ports.CheckerAction{Type: None}
}

func (c *Checker) checkBypass(msg *message.ChatMessage) *ports.CheckerAction {
	if msg.Chatter.IsBroadcaster || msg.Chatter.IsMod {
		c.log.Debug("Bypass: user is broadcaster or mod",
			slog.String("user", msg.Chatter.Username),
			slog.Bool("is_broadcaster", msg.Chatter.IsBroadcaster),
			slog.Bool("is_mod", msg.Chatter.IsMod),
		)
		return &ports.CheckerAction{Type: None}
	}

	if _, ok := c.cfg.Channels[msg.Broadcaster.Login].Spam.WhitelistUsers[msg.Chatter.Username]; ok {
		c.log.Debug("Bypass: user in whitelist", slog.String("user", msg.Chatter.Username))
		return &ports.CheckerAction{Type: None}
	}

	c.log.Trace("No bypass applied", slog.String("user", msg.Chatter.Username))
	return nil
}

func (c *Checker) checkBanwords(msg *message.ChatMessage) *ports.CheckerAction {
	if !c.template.Banwords().CheckMessage(
		msg.Message.Text.Words(message.RemovePunctuationOption, message.RemoveDuplicateLettersOption),
		msg.Message.Text.Words(message.LowerOption, message.RemovePunctuationOption, message.RemoveDuplicateLettersOption),
	) {
		c.log.Debug("No banwords detected", slog.String("user", msg.Chatter.Username), slog.String("message", msg.Message.Text.Text()))
		return nil
	}

	c.log.Debug("Banword detected", slog.String("user", msg.Chatter.Username), slog.String("message", msg.Message.Text.Text()))
	return &ports.CheckerAction{
		Type:   Ban,
		Reason: "банворд",
	}
}

func (c *Checker) checkAds(text string, username string) *ports.CheckerAction {
	if !strings.Contains(text, "twitch.tv/") ||
		strings.Contains(text, "twitch.tv/"+strings.ToLower(c.stream.ChannelName())) {
		c.log.Debug("No external Twitch link detected")
		return nil
	}

	if !strings.Contains(text, "twitch.tv/"+strings.ToLower(username)) &&
		!strings.Contains(text, "подписывайтесь") &&
		!strings.Contains(text, "подпишитесь") {
		c.log.Debug("No promotional keywords or self-link detected")
		return nil
	}

	c.log.Debug("Advertisement detected", slog.String("user", username), slog.String("text", text))
	return &ports.CheckerAction{
		Type:   Ban,
		Reason: "реклама",
	}
}

func (c *Checker) checkMwords(msg *message.ChatMessage) *ports.CheckerAction {
	trigger, punishments := c.template.Mword().Check(msg, c.stream.IsLive())
	if len(punishments) == 0 {
		c.log.Debug("No muteword violations found", slog.String("message", msg.Message.Text.Text()))
		return nil
	}

	countTimeouts, ok := c.timeouts.Get(msg.Chatter.Username, "mword")
	if !ok {
		c.log.Debug("Initializing muteword punishment counter", slog.String("user", msg.Chatter.Username), slog.String("message", msg.Message.Text.Text()))
		c.timeouts.Push(msg.Chatter.Username, "mword", 0, storage.WithTTL(
			time.Duration(c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsDefault.DurationResetPunishments)*time.Second),
		)
	}

	action, dur := c.template.Punishment().Get(punishments, countTimeouts)
	c.timeouts.Update(msg.Chatter.Username, "mword", func(cur int, exists bool) int {
		if !exists {
			return 1
		}
		return cur + 1
	})

	c.log.Info("Applying muteword punishment",
		slog.String("user", msg.Chatter.Username),
		slog.String("message", msg.Message.Text.Text()),
		slog.Any("punishments", punishments),
		slog.String("action", action),
		slog.Duration("duration", dur),
		slog.Int("previous_count", countTimeouts),
	)

	return &ports.CheckerAction{
		Type:     action,
		Reason:   fmt.Sprintf("мворд (%s)", trigger),
		Duration: dur,
	}
}

func (c *Checker) checkSpam(msg *message.ChatMessage) *ports.CheckerAction {
	settings := c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsDefault
	if msg.Chatter.IsVip {
		c.log.Trace("Applied VIP spam settings", slog.String("user", msg.Chatter.Username))
		settings = c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsVIP
	}

	if !settings.Enabled || !c.template.SpamPause().CanProcess() {
		c.log.Debug("Spam check skipped (disabled or paused)",
			slog.String("user", msg.Chatter.Username),
			slog.Bool("enabled", settings.Enabled),
			slog.Bool("can_process", c.template.SpamPause().CanProcess()),
		)
		return nil
	}

	if ((c.cfg.Channels[msg.Broadcaster.Login].Spam.Mode == config.OnlineMode || c.cfg.Channels[msg.Broadcaster.Login].Spam.Mode == 0) && !c.stream.IsLive()) ||
		(c.cfg.Channels[msg.Broadcaster.Login].Spam.Mode == config.OfflineMode && c.stream.IsLive()) {
		c.log.Debug("Spam check skipped due to mode mismatch",
			slog.String("user", msg.Chatter.Username),
			slog.Int("spam_mode", c.cfg.Channels[msg.Broadcaster.Login].Spam.Mode),
			slog.Bool("is_live", c.stream.IsLive()),
		)
		return nil
	}

	if action := c.handleWordLength(msg.Message.Text.Words(message.LowerOption, message.RemovePunctuationOption, message.RemoveDuplicateLettersOption), settings); action != nil {
		c.log.Info("Message exceeded max word length",
			slog.String("user", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
			slog.String("reason", action.Reason),
			slog.Any("action", action),
		)
		return action
	}

	countSpam, avgGap := c.calculateSpamMessages(msg, settings)
	c.log.Debug("Calculated spam stats",
		slog.String("user", msg.Chatter.Username),
		slog.String("message", msg.Message.Text.Text()),
		slog.Int("spam_count", countSpam),
		slog.Duration("avg_gap", avgGap),
	)

	if action := c.handleEmotes(msg, countSpam); action != nil {
		c.log.Debug("Emote spam check triggered",
			slog.String("user", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
			slog.String("reason", action.Reason),
			slog.Any("action", action),
		)

		if action.Type != None {
			c.messages.ClearKey(msg.Chatter.Username)
		}
		return action
	}

	if action := c.handleExceptions(msg, countSpam, "default"); action != nil {
		c.log.Debug("Spam exception triggered",
			slog.String("user", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
			slog.Any("action", action),
		)

		if action.Type != None {
			c.messages.ClearKey(msg.Chatter.Username)
		}
		return action
	}

	if countSpam < settings.MessageLimit {
		c.log.Trace("Spam threshold not reached",
			slog.String("user", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
			slog.Int("spam_count", countSpam),
			slog.Int("limit", settings.MessageLimit),
		)
		return nil
	}

	cacheKey := "spam_default"
	cacheTTL := time.Duration(c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsDefault.DurationResetPunishments) * time.Second
	if msg.Chatter.IsVip {
		cacheKey = "spam_vip"
		cacheTTL = time.Duration(c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsVIP.DurationResetPunishments) * time.Second
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

	c.log.Warn("Spam detected and punishment applied",
		slog.String("user", msg.Chatter.Username),
		slog.String("message", msg.Message.Text.Text()),
		slog.Int("spam_count", countSpam),
		slog.Int("previous_timeouts", countTimeouts),
		slog.Duration("duration", dur),
		slog.String("action", action),
	)

	c.messages.ClearKey(msg.Chatter.Username)
	return &ports.CheckerAction{
		Type:     action,
		Reason:   "спам",
		Duration: dur,
	}
}

func (c *Checker) calculateSpamMessages(msg *message.ChatMessage, settings config.SpamSettings) (int, time.Duration) {
	var countSpam, gap int
	var timestamps []time.Time

	c.messages.ForEach(msg.Chatter.Username, func(item *storage.Message) {
		c.log.Trace("Processing previous message",
			slog.String("user", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
			slog.String("old_message", item.Data.Message.Text.Text()),
			slog.Time("timestamp", item.Time),
		)

		if item.IgnoreAntispam {
			c.log.Trace("Skipping message because IgnoreAntispam enabled", slog.String("message", item.Data.Message.Text.Text()))
			return
		}

		similarity := domain.JaccardHashSimilarity(msg.Message.Text.Words(message.RemovePunctuationOption), item.Data.Message.Text.Words(message.RemovePunctuationOption))
		c.log.Trace("Calculated similarity with previous message",
			slog.String("user", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
			slog.String("old_message", item.Data.Message.Text.Text()),
			slog.Float64("similarity", similarity),
		)

		if similarity >= settings.SimilarityThreshold {
			c.log.Trace("Message is similar enough to be considered for spam",
				slog.String("user", msg.Chatter.Username),
				slog.String("message", msg.Message.Text.Text()),
				slog.String("old_message", item.Data.Message.Text.Text()),
				slog.Float64("similarity", similarity),
			)

			if !c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsEmotes.Enabled {
				_, isOnlyEmotes := c.sevenTV.EmoteStats(item.Data.Message.Text.Words(message.RemovePunctuationOption))
				if isOnlyEmotes || item.Data.Message.EmoteOnly {
					c.log.Trace("Skipping message because it contains only emotes",
						slog.String("user", msg.Chatter.Username),
						slog.String("message", msg.Message.Text.Text()),
						slog.String("old_message", item.Data.Message.Text.Text()),
					)
					return
				}
			}

			if gap < settings.MinGapMessages {
				countSpam++
				timestamps = append(timestamps, item.Time)
				c.log.Trace("Incremented spam count",
					slog.String("user", msg.Chatter.Username),
					slog.String("message", msg.Message.Text.Text()),
					slog.String("old_message", item.Data.Message.Text.Text()),
					slog.Int("current_spam_count", countSpam),
					slog.Int("gap", gap),
				)
			} else {
				c.log.Trace("Gap between messages is too large, not counting as spam",
					slog.Int("gap", gap),
					slog.Int("min_gap_required", settings.MinGapMessages),
				)
			}
			gap = 0
		} else {
			gap++
			c.log.Trace("Messages are not similar, incrementing gap", slog.Int("gap", gap))
		}
	})

	if len(timestamps) < 2 {
		c.log.Debug("Finished calculating spam messages",
			slog.String("user", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
			slog.Int("spam_count", countSpam),
		)
		return countSpam, 0
	}

	var total time.Duration
	for i := 1; i < len(timestamps); i++ {
		diff := timestamps[i].Sub(timestamps[i-1])
		if diff < 0 {
			diff = -diff
		}

		total += diff
		c.log.Trace("Adding interval between spam messages",
			slog.Time("prev_timestamp", timestamps[i-1]),
			slog.Time("current_timestamp", timestamps[i]),
			slog.Duration("interval", diff),
		)
	}

	avgGap := total / time.Duration(len(timestamps)-1)
	c.log.Debug("Finished calculating spam messages",
		slog.String("user", msg.Chatter.Username),
		slog.String("message", msg.Message.Text.Text()),
		slog.Int("spam_count", countSpam),
		slog.Duration("average_gap_between_spam", avgGap),
	)

	return countSpam, avgGap
}

func (c *Checker) handleWordLength(words []string, settings config.SpamSettings) *ports.CheckerAction {
	for _, word := range words {
		c.log.Trace("Checking word length", slog.String("word", word), slog.Int("length", len([]rune(word))))
		if settings.MaxWordLength > 0 && len([]rune(word)) >= settings.MaxWordLength {
			c.log.Warn("Word exceeds maximum length",
				slog.String("word", word),
				slog.Int("length", len([]rune(word))),
				slog.Int("max_length", settings.MaxWordLength),
			)
			return &ports.CheckerAction{
				Type:     settings.MaxWordPunishment.Action,
				Reason:   "превышена максимальная длина слова",
				Duration: time.Duration(settings.MaxWordPunishment.Duration) * time.Second,
			}
		}
	}
	return nil
}

func (c *Checker) handleEmotes(msg *message.ChatMessage, countSpam int) *ports.CheckerAction {
	count, isOnlyEmotes := c.sevenTV.EmoteStats(msg.Message.Text.Words(message.RemovePunctuationOption))
	emoteOnly := msg.Message.EmoteOnly || isOnlyEmotes

	c.log.Debug("Checking emotes in message",
		slog.String("message", msg.Message.Text.Text()),
		slog.Int("emote_count", count),
		slog.Bool("emote_only", emoteOnly),
		slog.Int("countSpam", countSpam),
	)

	if !emoteOnly {
		c.log.Debug("Message contains non-emote characters, skipping emote handling")
		return nil
	}

	if !c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsEmotes.Enabled {
		c.log.Debug("Emote spam checking disabled in config")
		return &ports.CheckerAction{Type: None}
	}

	if action := c.handleExceptions(msg, countSpam, "emote"); action != nil {
		c.log.Debug("Exception rule applied for emote spam",
			slog.String("user", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
			slog.Any("action", action),
		)

		return action
	}

	if c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsEmotes.MaxEmotesLength > 0 {
		emoteCount := max(len(msg.Message.Emotes), count)
		if emoteCount >= c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsEmotes.MaxEmotesLength {
			c.log.Debug("Message exceeds maximum emote count",
				slog.String("message", msg.Message.Text.Text()),
				slog.Int("emote_count", emoteCount),
				slog.Int("max_allowed", c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsEmotes.MaxEmotesLength),
			)
			return &ports.CheckerAction{
				Type:     c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsEmotes.MaxEmotesPunishment.Action,
				Reason:   "превышено максимальное кол-во эмоутов в сообщении",
				Duration: time.Duration(c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsEmotes.MaxEmotesPunishment.Duration) * time.Second,
			}
		}
	}

	if countSpam < c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsEmotes.MessageLimit {
		c.log.Debug("Spam emote message limit not reached yet",
			slog.String("message", msg.Message.Text.Text()),
			slog.Int("countSpam", countSpam),
			slog.Int("message_limit", c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsEmotes.MessageLimit),
		)
		return &ports.CheckerAction{Type: None}
	}

	countTimeouts, ok := c.timeouts.Get(msg.Chatter.Username, "spam_emote")
	if !ok {
		c.timeouts.Push(msg.Chatter.Username, "spam_emote", 0, storage.WithTTL(
			time.Duration(c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsDefault.DurationResetPunishments)*time.Second),
		)
	}

	action, dur := c.template.Punishment().Get(c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsEmotes.Punishments, countTimeouts)
	c.timeouts.Update(msg.Chatter.Username, "spam_emote", func(cur int, exists bool) int {
		if !exists {
			return 1
		}
		return cur + 1
	})

	c.log.Info("Emote spam action applied",
		slog.String("user", msg.Chatter.Username),
		slog.String("message", msg.Message.Text.Text()),
		slog.String("action", action),
		slog.Int("previous_timeouts", countTimeouts),
		slog.Duration("duration", dur),
	)

	return &ports.CheckerAction{
		Type:     action,
		Reason:   "спам эмоутов",
		Duration: dur,
	}
}

func (c *Checker) handleExceptions(msg *message.ChatMessage, countSpam int, typeSpam string) *ports.CheckerAction {
	exceptions, subKey := c.cfg.Channels[msg.Broadcaster.Login].Spam.Exceptions, "except_spam"
	if typeSpam == "emote" {
		exceptions, subKey = c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsEmotes.Exceptions, "except_emote"
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
				time.Duration(c.cfg.Channels[msg.Broadcaster.Login].Spam.SettingsDefault.DurationResetPunishments)*time.Second),
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

func (c *Checker) matchExceptRule(msg *message.ChatMessage, word string, re *regexp.Regexp, opts *config.ExceptOptions) bool {
	if word == "" {
		return false
	}

	text := msg.Message.Text.Text(message.LowerOption, message.RemovePunctuationOption, message.RemoveDuplicateLettersOption)
	words := msg.Message.Text.Words(message.LowerOption, message.RemovePunctuationOption, message.RemoveDuplicateLettersOption)

	if (opts == nil || opts.OneWord == nil || *opts.OneWord) && !c.template.Mword().CheckOneWord(words) {
		return false
	}

	if opts != nil {
		if opts.NoVip != nil && *opts.NoVip && msg.Chatter.IsVip {
			return false
		}
		if opts.NoSub != nil && *opts.NoSub && msg.Chatter.IsSubscriber {
			return false
		}

		textOpts := make([]message.TextOption, 0, 3)
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
