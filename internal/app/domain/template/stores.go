package template

import (
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/ports"
)

type StoresTemplate struct {
	messages ports.StorePort[storage.Message]
	timeouts ports.StorePort[int]
}

func NewStores(cfg *config.Config) *StoresTemplate {
	st := &StoresTemplate{
		messages: storage.New[storage.Message](50, time.Duration(cfg.WindowSecs)*time.Second),
		timeouts: storage.New[int](15, 0),
	}
	st.SetMessageCapacity(cfg)

	return st
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

func (s *StoresTemplate) Timeouts() ports.StorePort[int] {
	return s.timeouts
}
