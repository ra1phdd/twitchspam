package template

import (
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/ports"
)

type StoresTemplate struct {
	messages ports.StorePort[storage.Message]
	timeouts *ports.StoreTimeouts
}

func NewStores(cfg *config.Config) *StoresTemplate {
	st := &StoresTemplate{
		messages: storage.New[storage.Message](50, time.Duration(cfg.WindowSecs)*time.Second),
		timeouts: &ports.StoreTimeouts{
			SpamDefault:      storage.New[storage.Empty](15, time.Duration(cfg.Spam.SettingsDefault.DurationResetPunishments)*time.Second),
			SpamVIP:          storage.New[storage.Empty](15, time.Duration(cfg.Spam.SettingsVIP.DurationResetPunishments)*time.Second),
			SpamEmote:        storage.New[storage.Empty](15, time.Duration(cfg.Spam.SettingsEmotes.DurationResetPunishments)*time.Second),
			ExceptionsSpam:   storage.New[storage.Empty](15, time.Duration(cfg.WindowSecs)*time.Second),
			ExceptionsEmotes: storage.New[storage.Empty](15, time.Duration(cfg.WindowSecs)*time.Second),
			Mword:            storage.New[storage.Empty](15, time.Duration(cfg.WindowSecs)*time.Second),
		},
	}
	st.SetMessageCapacity(cfg)

	return st
}

func (s *StoresTemplate) SetAllTimeoutsTTL(cfg *config.Config) {
	s.timeouts.SpamDefault.SetTTL(time.Duration(cfg.Spam.SettingsDefault.DurationResetPunishments) * time.Second)
	s.timeouts.SpamVIP.SetTTL(time.Duration(cfg.Spam.SettingsVIP.DurationResetPunishments) * time.Second)
	s.timeouts.SpamEmote.SetTTL(time.Duration(cfg.Spam.SettingsEmotes.DurationResetPunishments) * time.Second)
	s.timeouts.ExceptionsSpam.SetTTL(time.Duration(cfg.Spam.SettingsDefault.DurationResetPunishments) * time.Second)
	s.timeouts.ExceptionsEmotes.SetTTL(time.Duration(cfg.Spam.SettingsEmotes.DurationResetPunishments) * time.Second)
	s.timeouts.Mword.SetTTL(time.Duration(cfg.Spam.SettingsDefault.DurationResetPunishments) * time.Second)
}

func (s *StoresTemplate) SetMessageCapacity(cfg *config.Config) {
	capacity := func() int {
		defLimit := float64(cfg.Spam.SettingsDefault.MessageLimit*cfg.Spam.SettingsDefault.MinGapMessages) / cfg.Spam.SettingsDefault.SimilarityThreshold
		vipLimit := float64(cfg.Spam.SettingsVIP.MessageLimit*cfg.Spam.SettingsVIP.MinGapMessages) / cfg.Spam.SettingsVIP.SimilarityThreshold
		emoteLimit := float64(cfg.Spam.SettingsEmotes.MessageLimit) / cfg.Spam.SettingsEmotes.EmoteThreshold

		return int(max(defLimit, vipLimit, emoteLimit))
	}()

	if capacity <= 50 {
		return
	}
	s.messages.SetCapacity(capacity)
}

func (s *StoresTemplate) Messages() ports.StorePort[storage.Message] {
	return s.messages
}

func (s *StoresTemplate) Timeouts() *ports.StoreTimeouts {
	return s.timeouts
}
