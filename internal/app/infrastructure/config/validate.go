package config

import (
	"errors"
	"fmt"
)

func (m *Manager) validate(cfg *Config) error {
	// app
	validLevels := map[string]bool{"trace": true, "debug": true, "info": true, "warn": true, "error": true}
	if cfg.App.LogLevel != "" && !validLevels[cfg.App.LogLevel] {
		return fmt.Errorf("app.log_level must be one of trace, debug, info, warn, error; got %s", cfg.App.LogLevel)
	}

	validGinModes := map[string]bool{"debug": true, "release": true}
	if cfg.App.GinMode != "" && !validGinModes[cfg.App.GinMode] {
		return fmt.Errorf("app.gin_mode must be one of debug, release; got %s", cfg.App.GinMode)
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
	if cfg.App.AuthToken == "" {
		return errors.New("app.auth_token is required")
	}

	if cfg.UsersTokens == nil {
		cfg.UsersTokens = make(map[string]*UserTokens)
	}

	// limiter
	if (cfg.Limiter.Requests != 0 && cfg.Limiter.Per == 0) || (cfg.Limiter.Requests == 0 && cfg.Limiter.Per != 0) {
		return errors.New("limiter.requests and limiter.per must both be set or both be zero")
	}

	validPunishments := map[string]bool{"none": true, "delete": true, "timeout": true, "warn": true, "ban": true}
	for _, channel := range cfg.Channels {
		channel.WindowSecs = 180

		// spam
		if channel.Spam.Mode < 0 || channel.Spam.Mode > 3 {
			return fmt.Errorf("spam.mode must be one of always (0), online (1), offline (2); got %d", channel.Spam.Mode)
		}
		if channel.Spam.WhitelistUsers == nil {
			channel.Spam.WhitelistUsers = make(map[string]struct{})
		}
		if channel.Spam.Exceptions == nil {
			channel.Spam.Exceptions = make(map[string]*ExceptionsSettings)
		}
		for _, except := range channel.Spam.Exceptions {
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
		if channel.Spam.SettingsDefault.SimilarityThreshold < 0.1 || channel.Spam.SettingsDefault.SimilarityThreshold > 1 {
			return errors.New("spam.settings_default.similarity_threshold must be in [0.1,1.0]")
		}
		if channel.Spam.SettingsDefault.MessageLimit < 2 || channel.Spam.SettingsDefault.MessageLimit > 15 {
			return errors.New("spam.settings_default.message_limit must be [2,15]")
		}
		if len(channel.Spam.SettingsDefault.Punishments) == 0 {
			return errors.New("spam.settings_default.punishments is required")
		}
		for _, punishment := range channel.Spam.SettingsDefault.Punishments {
			if !validPunishments[punishment.Action] {
				return fmt.Errorf("spam.settings_default.punishments must be on of delete, warn, timeout, ban; got %s", punishment.Action)
			}

			if punishment.Duration < 0 || punishment.Duration > 1209600 {
				return errors.New("spam.exceptions.duration must be [0,1209600]")
			}
		}
		if channel.Spam.SettingsDefault.DurationResetPunishments < 0 || channel.Spam.SettingsDefault.DurationResetPunishments > 86400 {
			return errors.New("spam.settings_default.reset_timeout_seconds must be [0,86400]")
		}
		if channel.Spam.SettingsDefault.MaxWordLength < 0 || channel.Spam.SettingsDefault.MaxWordLength > 500 {
			return errors.New("spam.settings_default.max_word_length must be [0,500]")
		}
		if channel.Spam.SettingsDefault.MaxWordLength != 0 && !validPunishments[channel.Spam.SettingsDefault.MaxWordPunishment.Action] {
			return fmt.Errorf("spam.settings_default.max_word_punishment must be on of delete, warn, timeout, ban; got %s", channel.Spam.SettingsDefault.MaxWordPunishment.Action)
		}
		if channel.Spam.SettingsDefault.MinGapMessages < 0 || channel.Spam.SettingsDefault.MinGapMessages > 15 {
			return errors.New("spam.settings_default.min_gap_messages must be in 0..15")
		}

		// spam settings vip
		if channel.Spam.SettingsVIP.SimilarityThreshold < 0.1 || channel.Spam.SettingsVIP.SimilarityThreshold > 1 {
			return errors.New("spam.settings_vip.similarity_threshold must be in [0.1,1.0]")
		}
		if channel.Spam.SettingsVIP.MessageLimit < 2 || channel.Spam.SettingsVIP.MessageLimit > 15 {
			return errors.New("spam.settings_vip.message_limit must be [2,15]")
		}
		if len(channel.Spam.SettingsVIP.Punishments) == 0 {
			return errors.New("spam.settings_vip.punishments is required")
		}
		for _, punishment := range channel.Spam.SettingsVIP.Punishments {
			if !validPunishments[punishment.Action] {
				return fmt.Errorf("spam.settings_vip.punishments must be on of delete, warn, timeout, ban; got %s", punishment.Action)
			}

			if punishment.Duration < 0 || punishment.Duration > 1209600 {
				return errors.New("spam.exceptions.duration must be [0,1209600]")
			}
		}
		if channel.Spam.SettingsVIP.DurationResetPunishments < 0 || channel.Spam.SettingsVIP.DurationResetPunishments > 86400 {
			return errors.New("spam.settings_vip.reset_timeout_seconds must be [0,86400]")
		}
		if channel.Spam.SettingsVIP.MaxWordLength < 0 || channel.Spam.SettingsVIP.MaxWordLength > 500 {
			return errors.New("spam.settings_vip.max_word_length must be [0,500]")
		}
		if channel.Spam.SettingsVIP.MaxWordLength != 0 && !validPunishments[channel.Spam.SettingsVIP.MaxWordPunishment.Action] {
			return fmt.Errorf("spam.settings_vip.max_word_punishment must be on of delete, warn, timeout, ban; got %s", channel.Spam.SettingsVIP.MaxWordPunishment.Action)
		}
		if channel.Spam.SettingsVIP.MinGapMessages < 0 || channel.Spam.SettingsVIP.MinGapMessages > 15 {
			return errors.New("spam.settings_vip.min_gap_messages must be in 0..15")
		}

		// spam settings emote
		if channel.Spam.SettingsEmotes.EmoteThreshold < 0.1 || channel.Spam.SettingsEmotes.EmoteThreshold > 1 {
			return errors.New("spam.settings_emote.emote_threshold must be in [0.1,1.0]")
		}
		if channel.Spam.SettingsEmotes.MessageLimit < 2 || channel.Spam.SettingsEmotes.MessageLimit > 15 {
			return errors.New("spam.settings_emote.message_limit must be [2,15]")
		}
		if len(channel.Spam.SettingsEmotes.Punishments) == 0 {
			return errors.New("spam.settings_emote.punishments is required")
		}
		for _, punishment := range channel.Spam.SettingsEmotes.Punishments {
			if !validPunishments[punishment.Action] {
				return fmt.Errorf("spam.settings_emote.punishments must be on of delete, warn, timeout, ban; got %s", punishment.Action)
			}

			if punishment.Duration < 0 || punishment.Duration > 1209600 {
				return errors.New("spam.exceptions.duration must be [0,1209600]")
			}
		}
		if channel.Spam.SettingsEmotes.DurationResetPunishments < 0 || channel.Spam.SettingsEmotes.DurationResetPunishments > 86400 {
			return errors.New("spam.settings_emote.reset_timeout_seconds must be [0,86400]")
		}
		if channel.Spam.SettingsEmotes.MaxEmotesLength < 0 || channel.Spam.SettingsEmotes.MaxEmotesLength > 500 {
			return errors.New("spam.settings_emote.max_emote_length must be [0,500]")
		}
		if channel.Spam.SettingsEmotes.MaxEmotesLength != 0 && !validPunishments[channel.Spam.SettingsEmotes.MaxEmotesPunishment.Action] {
			return fmt.Errorf("spam.settings_emote.max_emotes_punishments must be on of delete, warn, timeout, ban; got %s", channel.Spam.SettingsEmotes.MaxEmotesPunishment.Action)
		}
		if channel.Spam.SettingsEmotes.Exceptions == nil {
			channel.Spam.SettingsEmotes.Exceptions = make(map[string]*ExceptionsSettings)
		}
		for _, except := range channel.Spam.SettingsEmotes.Exceptions {
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
		if channel.Automod.Delay < 0 || channel.Automod.Delay > 10 {
			return errors.New("automod.delay must be [0,10]")
		}

		// мворды
		for _, mw := range channel.Mword {
			if mw.Word == "" && mw.Regexp == nil {
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

		if channel.MwordGroup == nil {
			channel.MwordGroup = make(map[string]*MwordGroup)
		}
		for _, mwg := range channel.MwordGroup {
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
		}

		if channel.Markers == nil {
			channel.Markers = make(map[string]map[string][]*Markers)
		}

		if channel.Commands == nil {
			channel.Commands = make(map[string]*Commands)
		}

		if channel.Aliases == nil {
			channel.Aliases = make(map[string]string)
		}

		if channel.AliasGroups == nil {
			channel.AliasGroups = make(map[string]*AliasGroups)
		}
		for _, alg := range channel.AliasGroups {
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
	}

	if cfg.GlobalAliases == nil {
		cfg.GlobalAliases = make(map[string]string)
	}

	return nil
}
