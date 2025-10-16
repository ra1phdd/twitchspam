package config

import "time"

const (
	_ = iota
	AlwaysMode
	OnlineMode
	OfflineMode
)

func (m *Manager) GetDefault() *Config {
	return &Config{
		App: App{
			LogLevel: "info",
			GinMode:  "release",
		},
		WindowSecs: 180,
		Limiter: Limiter{
			Requests: 3,
			Per:      30 * time.Second,
		},
		Spam: Spam{
			Mode:           OnlineMode,
			WhitelistUsers: make(map[string]struct{}),
			Exceptions:     make(map[string]*ExceptionsSettings),
			SettingsDefault: SpamSettings{
				Enabled:             true,
				SimilarityThreshold: 0.7,
				MessageLimit:        3,
				Punishments: []Punishment{
					{Action: "timeout", Duration: 600},
					{Action: "timeout", Duration: 1800},
					{Action: "timeout", Duration: 3600},
				},
				DurationResetPunishments: 3600,
				MaxWordLength:            100,
				MaxWordPunishment: Punishment{
					Action:   "timeout",
					Duration: 30,
				},
				MinGapMessages: 3,
			},
			SettingsVIP: SpamSettings{
				Enabled:             false,
				SimilarityThreshold: 0.7,
				MessageLimit:        3,
				Punishments: []Punishment{
					{Action: "timeout", Duration: 600},
					{Action: "timeout", Duration: 1800},
					{Action: "timeout", Duration: 3600},
				},
				DurationResetPunishments: 3600,
				MaxWordLength:            100,
				MaxWordPunishment: Punishment{
					Action:   "timeout",
					Duration: 30,
				},
				MinGapMessages: 3,
			},
			SettingsEmotes: SpamSettingsEmote{
				Enabled:        true,
				EmoteThreshold: 0.9,
				MessageLimit:   7,
				Punishments: []Punishment{
					{Action: "timeout", Duration: 60},
					{Action: "timeout", Duration: 300},
					{Action: "timeout", Duration: 600},
				},
				DurationResetPunishments: 600,
				Exceptions:               make(map[string]*ExceptionsSettings),
				MaxEmotesLength:          15,
				MaxEmotesPunishment: Punishment{
					Action:   "timeout",
					Duration: 30,
				},
			},
		},
		Automod: Automod{
			Enabled: true,
			Delay:   0,
		},
		MwordGroup:    make(map[string]*MwordGroup),
		Aliases:       make(map[string]string),
		AliasGroups:   make(map[string]*AliasGroups),
		GlobalAliases: make(map[string]string),
		Markers:       make(map[string]map[string][]*Markers),
		Commands:      make(map[string]*Commands),
	}
}
