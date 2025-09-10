package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dlclark/regexp2"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Config struct {
	App        App                             `json:"app"`
	Enabled    bool                            `json:"enabled"`
	Spam       Spam                            `json:"spam"`
	Mword      map[string]Mword                `json:"mword"`
	MwordGroup map[string]MwordGroup           `json:"mword_group"`
	Aliases    map[string]string               `json:"aliases"` // ключ - алиас, значение - оригинальная команда
	Markers    map[string]map[string][]Markers `json:"markers"` // первый ключ - юзернейм, второй ключ - название маркера
	Links      map[string]Links                `json:"links"`
	Answers    map[string]Answers              `json:"answers"` // ключ
	Banwords   Banwords                        `json:"banwords"`
}

type App struct {
	LogLevel    string   `json:"log_level"`
	OAuth       string   `json:"oauth,required"`
	ClientID    string   `json:"client_id,required"`
	Username    string   `json:"username,required"`
	UserID      string   `json:"user_id,required"`
	ModChannels []string `json:"mod_channels,required"`
}

type Spam struct {
	Mode               string                            `json:"mode"`                 // !am online/always - только на стриме/всегда
	CheckWindowSeconds int                               `json:"check_window_seconds"` // !am time <секунды, макс 300>
	DelayAutomod       int                               `json:"delay_automod"`        // !am da <0-10> - задержка срабатывания
	WhitelistUsers     []string                          `json:"whitelist_users"`      // !am add/del <список>
	SettingsDefault    SpamSettings                      `json:"settings_default"`
	SettingsVIP        SpamSettings                      `json:"settings_vip"`
	SettingsEmotes     SpamSettingsEmote                 `json:"settings_emotes"`
	Exceptions         map[string]SpamExceptionsSettings `json:"exceptions"`
}

type SpamSettings struct {
	Enabled                  bool         `json:"enabled"`
	SimilarityThreshold      float64      `json:"similarity_threshold"`       // !am sim <0.1-1.0>
	MessageLimit             int          `json:"message_limit"`              // !am msg <2-15 или off>
	Punishments              []Punishment `json:"punishments"`                // !am p <значения через запятую>
	DurationResetPunishments int          `json:"duration_reset_punishments"` // !am rto <значение>
	MaxWordLength            int          `json:"max_word_length"`            // !am mwlen <значение или 0 для оффа>
	MaxWordPunishment        Punishment   `json:"max_word_punishment"`
	MinGapMessages           int          `json:"min_gap_messages"` // !am min_gap <0-15>
}

type SpamSettingsEmote struct {
	Enabled                  bool                        `json:"enabled"`
	MessageLimit             int                         `json:"message_limit"`
	Punishments              []Punishment                `json:"punishments"`
	DurationResetPunishments int                         `json:"duration_reset_punishments"`
	Exceptions               map[string]*EmoteExceptions `json:"exceptions"`
	MaxEmotesLength          int                         `json:"max_emotes_length"`
	MaxEmotesPunishment      Punishment                  `json:"max_emotes_punishment"`
}

type EmoteExceptions struct {
	Enabled      bool            `json:"enabled"`
	MessageLimit int             `json:"message_limit"`
	Punishments  []Punishment    `json:"punishments"`
	Regexp       *regexp2.Regexp `json:"regexp"`
}

type SpamExceptionsSettings struct {
	MessageLimit int             `json:"message_limit"`
	Punishments  []Punishment    `json:"punishments"`
	Regexp       *regexp2.Regexp `json:"regexp"`
}

type Mword struct {
	Punishments []Punishment    `json:"punishments"`
	Regexp      *regexp2.Regexp `json:"regexp"`
}

type MwordGroup struct {
	Enabled     bool              `json:"enabled"`
	Punishments []Punishment      `json:"punishments"`
	Words       []string          `json:"words"`
	Regexp      []*regexp2.Regexp `json:"regexp"`
}

type Markers struct {
	StreamID  string        `json:"stream_id"`
	CreatedAt time.Time     `json:"date"`
	Timecode  time.Duration `json:"time_code"`
}

type Links struct {
	Text string `json:"text"`
}

type Answers struct {
	Enabled bool              `json:"enabled"`
	Words   []string          `json:"words"`
	Regexp  []*regexp2.Regexp `json:"regexp"`
	Text    string            `json:"text"`
}

type Banwords struct {
	Words  []string          `json:"words"`
	Regexp []*regexp2.Regexp `json:"regexp"`
}

type Punishment struct {
	Action   string `json:"action"`   // "delete", "ban", "timeout"
	Duration int    `json:"duration"` // только для таймаута
}

type Manager struct {
	mu   sync.RWMutex
	cfg  *Config
	path string
}

func New(path string) (*Manager, error) {
	m := &Manager{path: path}

	var err error
	m.cfg, err = m.readParseValidate(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			m.cfg = m.GetDefault()
			data, err := json.MarshalIndent(m.cfg, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("marshal config: %w", err)
			}
			if err := m.writeAtomic(path, data, 0644); err != nil {
				return nil, fmt.Errorf("write config: %w", err)
			}
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	return m, nil
}

func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.cfg
}

func (m *Manager) Update(modify func(cfg *Config)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cfg == nil {
		return errors.New("no config loaded")
	}

	modify(m.cfg)

	if err := m.validate(m.cfg); err != nil {
		return fmt.Errorf("invalid config update: %w", err)
	}

	return m.saveLocked()
}

func (m *Manager) GetDefault() *Config {
	return &Config{
		App: App{},
		Spam: Spam{
			Mode:               "online",
			CheckWindowSeconds: 60,
			Exceptions:         make(map[string]SpamExceptionsSettings),
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
				Enabled:      true,
				MessageLimit: 7,
				Punishments: []Punishment{
					{Action: "timeout", Duration: 60},
					{Action: "timeout", Duration: 300},
					{Action: "timeout", Duration: 600},
				},
				DurationResetPunishments: 600,
				MaxEmotesLength:          15,
				MaxEmotesPunishment: Punishment{
					Action:   "timeout",
					Duration: 30,
				},
			},
			DelayAutomod: 0,
		},
		Mword:      make(map[string]Mword),
		MwordGroup: make(map[string]MwordGroup),
		Aliases:    make(map[string]string),
		Markers:    make(map[string]map[string][]Markers),
		Links:      make(map[string]Links),
		Answers:    make(map[string]Answers),
	}
}

func (m *Manager) readParseValidate(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("open/read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	if err := m.validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return &cfg, nil
}

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

	// spam
	if cfg.Spam.Mode != "online" && cfg.Spam.Mode != "always" {
		return errors.New("spam.mode must be 'online' or 'always'")
	}
	if cfg.Spam.CheckWindowSeconds < 1 || cfg.Spam.CheckWindowSeconds > 300 {
		return errors.New("spam.check_window_seconds must be 1..300")
	}
	if cfg.Spam.DelayAutomod < 0 || cfg.Spam.DelayAutomod > 10 {
		return errors.New("spam.delay_automod must be between 0 and 10")
	}

	spam := map[string]SpamSettings{
		"default": cfg.Spam.SettingsDefault,
		"vip":     cfg.Spam.SettingsVIP,
	}

	for _, s := range spam {
		if s.SimilarityThreshold < 0.1 || cfg.Spam.SettingsDefault.SimilarityThreshold > 1 {
			return errors.New("spam.similarity_threshold must be in [0.1,1.0]")
		}
		if cfg.Spam.SettingsDefault.MessageLimit < 2 || cfg.Spam.SettingsDefault.MessageLimit > 15 {
			return errors.New("spam.message_limit must be 2..15")
		}
		if cfg.Spam.SettingsDefault.DurationResetPunishments < 0 {
			return errors.New("spam.reset_timeout_seconds must be >= 0")
		}
		if cfg.Spam.SettingsDefault.MaxWordLength < 0 {
			return errors.New("spam.max_word must be >= 0")
		}
		if cfg.Spam.SettingsDefault.MinGapMessages < 0 || cfg.Spam.SettingsDefault.MinGapMessages > 15 {
			return errors.New("spam.min_gap_messages must be in 0..15")
		}
	}

	if cfg.Spam.Exceptions == nil {
		cfg.Spam.Exceptions = make(map[string]SpamExceptionsSettings)
	}

	if cfg.Mword == nil {
		cfg.Mword = make(map[string]Mword)
	}

	if cfg.MwordGroup == nil {
		cfg.MwordGroup = make(map[string]MwordGroup)
	}

	if cfg.Aliases == nil {
		cfg.Aliases = make(map[string]string)
	}

	if cfg.Markers == nil {
		cfg.Markers = make(map[string]map[string][]Markers)
	}

	if cfg.Links == nil {
		cfg.Links = make(map[string]Links)
	}

	if cfg.Answers == nil {
		cfg.Answers = make(map[string]Answers)
	}

	return nil
}

func (m *Manager) saveLocked() error {
	if m.path == "" {
		return errors.New("no config file loaded")
	}
	if m.cfg == nil {
		return errors.New("no config to save")
	}

	data, err := json.MarshalIndent(m.cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return m.writeAtomic(m.path, data, 0644)
}

func (m *Manager) writeAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp := filepath.Join(dir, fmt.Sprintf(".%s.tmp-%d", base, time.Now().UnixNano()))

	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
