package config

import (
	"errors"
	"fmt"
	"regexp"
)

func (m *Manager) validate(cfg *Config) error {
	// app
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if cfg.App.LogLevel != "" && !validLevels[cfg.App.LogLevel] {
		return fmt.Errorf("app.log_level must be one of debug, info, warn, error; got %s", cfg.App.LogLevel)
	}

	if cfg.App.OAuth == "" {
		return errors.New("app.oauth is required")
	}
	if cfg.App.ClientID == "" {
		return errors.New("app.client_id is required")
	}
	if cfg.App.Username == "" {
		return errors.New("app.username is required")
	}
	if cfg.App.UserID == "" {
		return errors.New("app.user_id is required")
	}
	if len(cfg.App.ModChannels) == 0 {
		return errors.New("app.mod_channels is required")
	}

	cfg.WindowSecs = 180

	// limiter
	if (cfg.Limiter.Requests != 0 && cfg.Limiter.Per == 0) || (cfg.Limiter.Requests == 0 && cfg.Limiter.Per != 0) {
		return errors.New("limiter.requests and limiter.per must both be set or both be zero")
	}

	validPunishments := map[string]bool{"delete": true, "timeout": true, "warn": true, "ban": true}

	// spam
	if cfg.Spam.Mode != "online" && cfg.Spam.Mode != "always" {
		return errors.New("spam.mode must be 'online' or 'always'")
	}
	if cfg.Spam.WhitelistUsers == nil {
		cfg.Spam.WhitelistUsers = make(map[string]struct{})
	}
	if cfg.Spam.Exceptions == nil {
		cfg.Spam.Exceptions = make(map[string]*ExceptionsSettings)
	}
	for _, except := range cfg.Spam.Exceptions {
		if except == nil {
			return errors.New("spam.exceptions.value is required")
		}

		if except.MessageLimit < 2 || except.MessageLimit > 15 {
			return errors.New("spam.exceptions.message_limit must be [2,15]")
		}

		if len(except.Punishments) == 0 {
			return errors.New("spam.exceptions.punishments is required")
		}
		for _, punishment := range except.Punishments {
			if !validPunishments[punishment.Action] {
				return fmt.Errorf("spam.exceptions.punishments must be on of delete, warn, timeout, ban; got %s", punishment.Action)
			}

			if punishment.Duration < 0 || punishment.Duration > 1209600 {
				return errors.New("spam.exceptions.duration must be [0,1209600]")
			}
		}
	}

	// spam settings default
	if cfg.Spam.SettingsDefault.SimilarityThreshold < 0.1 || cfg.Spam.SettingsDefault.SimilarityThreshold > 1 {
		return errors.New("spam.settings_default.similarity_threshold must be in [0.1,1.0]")
	}
	if cfg.Spam.SettingsDefault.MessageLimit < 2 || cfg.Spam.SettingsDefault.MessageLimit > 15 {
		return errors.New("spam.settings_default.message_limit must be [2,15]")
	}
	if len(cfg.Spam.SettingsDefault.Punishments) == 0 {
		return errors.New("spam.settings_default.punishments is required")
	}
	for _, punishment := range cfg.Spam.SettingsDefault.Punishments {
		if !validPunishments[punishment.Action] {
			return fmt.Errorf("spam.settings_default.punishments must be on of delete, warn, timeout, ban; got %s", punishment.Action)
		}

		if punishment.Duration < 0 || punishment.Duration > 1209600 {
			return errors.New("spam.exceptions.duration must be [0,1209600]")
		}
	}
	if cfg.Spam.SettingsDefault.DurationResetPunishments < 0 || cfg.Spam.SettingsDefault.DurationResetPunishments > 86400 {
		return errors.New("spam.settings_default.reset_timeout_seconds must be [0,86400]")
	}
	if cfg.Spam.SettingsDefault.MaxWordLength < 0 || cfg.Spam.SettingsDefault.MaxWordLength > 500 {
		return errors.New("spam.settings_default.max_word_length must be [0,500]")
	}
	if cfg.Spam.SettingsDefault.MaxWordLength != 0 && !validPunishments[cfg.Spam.SettingsDefault.MaxWordPunishment.Action] {
		return fmt.Errorf("spam.settings_default.max_word_punishment must be on of delete, warn, timeout, ban; got %s", cfg.Spam.SettingsDefault.MaxWordPunishment.Action)
	}
	if cfg.Spam.SettingsDefault.MinGapMessages < 0 || cfg.Spam.SettingsDefault.MinGapMessages > 15 {
		return errors.New("spam.settings_default.min_gap_messages must be in 0..15")
	}

	// spam settings vip
	if cfg.Spam.SettingsVIP.SimilarityThreshold < 0.1 || cfg.Spam.SettingsVIP.SimilarityThreshold > 1 {
		return errors.New("spam.settings_vip.similarity_threshold must be in [0.1,1.0]")
	}
	if cfg.Spam.SettingsVIP.MessageLimit < 2 || cfg.Spam.SettingsVIP.MessageLimit > 15 {
		return errors.New("spam.settings_vip.message_limit must be [2,15]")
	}
	if len(cfg.Spam.SettingsVIP.Punishments) == 0 {
		return errors.New("spam.settings_vip.punishments is required")
	}
	for _, punishment := range cfg.Spam.SettingsVIP.Punishments {
		if !validPunishments[punishment.Action] {
			return fmt.Errorf("spam.settings_vip.punishments must be on of delete, warn, timeout, ban; got %s", punishment.Action)
		}

		if punishment.Duration < 0 || punishment.Duration > 1209600 {
			return errors.New("spam.exceptions.duration must be [0,1209600]")
		}
	}
	if cfg.Spam.SettingsVIP.DurationResetPunishments < 0 || cfg.Spam.SettingsVIP.DurationResetPunishments > 86400 {
		return errors.New("spam.settings_vip.reset_timeout_seconds must be [0,86400]")
	}
	if cfg.Spam.SettingsVIP.MaxWordLength < 0 || cfg.Spam.SettingsVIP.MaxWordLength > 500 {
		return errors.New("spam.settings_vip.max_word_length must be [0,500]")
	}
	if cfg.Spam.SettingsVIP.MaxWordLength != 0 && !validPunishments[cfg.Spam.SettingsVIP.MaxWordPunishment.Action] {
		return fmt.Errorf("spam.settings_vip.max_word_punishment must be on of delete, warn, timeout, ban; got %s", cfg.Spam.SettingsVIP.MaxWordPunishment.Action)
	}
	if cfg.Spam.SettingsVIP.MinGapMessages < 0 || cfg.Spam.SettingsVIP.MinGapMessages > 15 {
		return errors.New("spam.settings_vip.min_gap_messages must be in 0..15")
	}

	// spam settings emote
	if cfg.Spam.SettingsEmotes.EmoteThreshold < 0.1 || cfg.Spam.SettingsEmotes.EmoteThreshold > 1 {
		return errors.New("spam.settings_emote.emote_threshold must be in [0.1,1.0]")
	}
	if cfg.Spam.SettingsEmotes.MessageLimit < 2 || cfg.Spam.SettingsEmotes.MessageLimit > 15 {
		return errors.New("spam.settings_emote.message_limit must be [2,15]")
	}
	if len(cfg.Spam.SettingsEmotes.Punishments) == 0 {
		return errors.New("spam.settings_emote.punishments is required")
	}
	for _, punishment := range cfg.Spam.SettingsEmotes.Punishments {
		if !validPunishments[punishment.Action] {
			return fmt.Errorf("spam.settings_emote.punishments must be on of delete, warn, timeout, ban; got %s", punishment.Action)
		}

		if punishment.Duration < 0 || punishment.Duration > 1209600 {
			return errors.New("spam.exceptions.duration must be [0,1209600]")
		}
	}
	if cfg.Spam.SettingsEmotes.DurationResetPunishments < 0 || cfg.Spam.SettingsEmotes.DurationResetPunishments > 86400 {
		return errors.New("spam.settings_emote.reset_timeout_seconds must be [0,86400]")
	}
	if cfg.Spam.SettingsEmotes.MaxEmotesLength < 0 || cfg.Spam.SettingsEmotes.MaxEmotesLength > 500 {
		return errors.New("spam.settings_emote.max_emote_length must be [0,500]")
	}
	if cfg.Spam.SettingsEmotes.MaxEmotesLength != 0 && !validPunishments[cfg.Spam.SettingsEmotes.MaxEmotesPunishment.Action] {
		return fmt.Errorf("spam.settings_emote.max_emotes_punishments must be on of delete, warn, timeout, ban; got %s", cfg.Spam.SettingsEmotes.MaxEmotesPunishment.Action)
	}
	if cfg.Spam.SettingsEmotes.Exceptions == nil {
		cfg.Spam.SettingsEmotes.Exceptions = make(map[string]*ExceptionsSettings)
	}
	for _, except := range cfg.Spam.SettingsEmotes.Exceptions {
		if except == nil {
			return errors.New("spam.settings_emote.exceptions.value is required")
		}

		if except.MessageLimit < 2 || except.MessageLimit > 15 {
			return errors.New("spam.settings_emote.exceptions.message_limit must be [2,15]")
		}

		if len(except.Punishments) == 0 {
			return errors.New("spam.settings_emote.exceptions.punishments is required")
		}
		for _, punishment := range except.Punishments {
			if !validPunishments[punishment.Action] {
				return fmt.Errorf("spam.settings_emote.exceptions.punishments must be on of delete, warn, timeout, ban; got %s", punishment.Action)
			}

			if punishment.Duration < 0 || punishment.Duration > 1209600 {
				return errors.New("spam.exceptions.duration must be [0,1209600]")
			}
		}
	}

	// automod
	if cfg.Automod.Delay < 0 || cfg.Automod.Delay > 10 {
		return errors.New("automod.delay must be [0,10]")
	}

	// мворды
	if cfg.Mword == nil {
		cfg.Mword = make(map[string]*Mword)
	}
	for _, mw := range cfg.Mword {
		if mw == nil {
			return errors.New("mword.value is required")
		}

		if len(mw.Punishments) == 0 {
			return errors.New("mword.exceptions.punishments is required")
		}
		for _, punishment := range mw.Punishments {
			if !validPunishments[punishment.Action] {
				return fmt.Errorf("mword.punishments must be on of delete, warn, timeout, ban; got %s", punishment.Action)
			}

			if punishment.Duration < 0 || punishment.Duration > 1209600 {
				return errors.New("mword.duration must be [0,1209600]")
			}
		}
	}

	if cfg.MwordGroup == nil {
		cfg.MwordGroup = make(map[string]*MwordGroup)
	}
	for _, mwg := range cfg.MwordGroup {
		if mwg == nil {
			return errors.New("mword_group.value is required")
		}

		if len(mwg.Punishments) == 0 {
			return errors.New("mword_group.punishments is required")
		}
		for _, punishment := range mwg.Punishments {
			if !validPunishments[punishment.Action] {
				return fmt.Errorf("mword_group.punishments must be on of delete, warn, timeout, ban; got %s", punishment.Action)
			}

			if punishment.Duration < 0 || punishment.Duration > 1209600 {
				return errors.New("mword_group.duration must be [0,1209600]")
			}
		}

		if mwg.Regexp == nil {
			mwg.Regexp = make(map[string]*regexp.Regexp)
		}
	}

	if cfg.Markers == nil {
		cfg.Markers = make(map[string]map[string][]*Markers)
	}

	if cfg.Commands == nil {
		cfg.Commands = make(map[string]*Commands)
	}

	if cfg.Aliases == nil {
		cfg.Aliases = make(map[string]string)
	}

	if cfg.AliasGroups == nil {
		cfg.AliasGroups = make(map[string]*AliasGroups)
	}
	for _, alg := range cfg.AliasGroups {
		if alg == nil {
			return errors.New("mword_group.value is required")
		}

		if alg.Aliases == nil {
			alg.Aliases = make(map[string]struct{})
		}

		if alg.Original == "" {
			return errors.New("mword_group.original must not be empty")
		}
	}

	return nil
}
