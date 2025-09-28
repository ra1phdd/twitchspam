package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/time/rate"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

type Config struct {
	App         App                              `json:"app"`
	Enabled     bool                             `json:"enabled"`
	Limiter     *Limiter                         `json:"limiter"`
	Spam        Spam                             `json:"spam"`
	Automod     Automod                          `json:"automod"`
	Mword       map[string]*Mword                `json:"mword"`
	MwordGroup  map[string]*MwordGroup           `json:"mword_group"`
	Aliases     map[string]string                `json:"aliases"`       // ключ - алиас, значение - оригинальная команда
	AliasGroups map[string]*AliasGroups          `json:"aliases_group"` // первый ключ - название группы, второй ключ - алиас, значение - оригинальная команда
	Markers     map[string]map[string][]*Markers `json:"markers"`       // первый ключ - юзернейм, второй ключ - название маркера
	Commands    map[string]*Commands             `json:"commands"`
	Banwords    Banwords                         `json:"banwords"`
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
	Mode               string                         `json:"mode"`                 // !am online/always - только на стриме/всегда
	CheckWindowSeconds int                            `json:"check_window_seconds"` // !am time <секунды, макс 300>
	WhitelistUsers     map[string]struct{}            `json:"whitelist_users"`      // !am add/del <список>
	SettingsDefault    SpamSettings                   `json:"settings_default"`
	SettingsVIP        SpamSettings                   `json:"settings_vip"`
	SettingsEmotes     SpamSettingsEmote              `json:"settings_emotes"`
	Exceptions         map[string]*ExceptionsSettings `json:"exceptions"`
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
	Enabled                  bool                           `json:"enabled"`
	EmoteThreshold           float64                        `json:"emote_threshold"`
	MessageLimit             int                            `json:"message_limit"`
	Punishments              []Punishment                   `json:"punishments"`
	DurationResetPunishments int                            `json:"duration_reset_punishments"`
	Exceptions               map[string]*ExceptionsSettings `json:"exceptions"`
	MaxEmotesLength          int                            `json:"max_emotes_length"`
	MaxEmotesPunishment      Punishment                     `json:"max_emotes_punishment"`
}

type Automod struct {
	Enabled bool `json:"enabled"`
	Delay   int  `json:"delay"`
}

type ExceptionsSettings struct {
	Enabled      bool           `json:"enabled"`
	MessageLimit int            `json:"message_limit"`
	Punishments  []Punishment   `json:"punishments"`
	Options      ExceptOptions  `json:"options"`
	Regexp       *regexp.Regexp `json:"regexp"`
}

type AliasGroups struct {
	Aliases  map[string]struct{} `json:"aliases"`
	Original string              `json:"original"`
}

type Mword struct {
	Punishments []Punishment   `json:"punishments"`
	Options     MwordOptions   `json:"options"`
	Regexp      *regexp.Regexp `json:"regexp"`
}

type MwordGroup struct {
	Enabled     bool             `json:"enabled"`
	Punishments []Punishment     `json:"punishments"`
	Options     MwordOptions     `json:"options"`
	Words       []string         `json:"words"`
	Regexp      []*regexp.Regexp `json:"regexp"`
}

type Markers struct {
	StreamID  string        `json:"stream_id"`
	CreatedAt time.Time     `json:"date"`
	Timecode  time.Duration `json:"time_code"`
}

type Commands struct {
	Text    string   `json:"text"`
	Timer   *Timers  `json:"timer"`
	Limiter *Limiter `json:"limiter"`
}

type Timers struct {
	Enabled  bool          `json:"enabled"`
	Interval time.Duration `json:"interval"`
	Count    int           `json:"count"`
	Options  TimerOptions  `json:"options"`
}

type Banwords struct {
	Words  []string         `json:"words"`
	Regexp []*regexp.Regexp `json:"regexp"`
}

type Punishment struct {
	Action   string `json:"action"`   // "delete", "ban", "timeout"
	Duration int    `json:"duration"` // только для таймаута
}

type ExceptOptions struct {
	NoSub         bool `json:"no_sub"`
	NoVip         bool `json:"no_vip"`
	NoRepeat      bool `json:"norepeat"`
	OneWord       bool `json:"one_word"`
	Contains      bool `json:"contains"`
	CaseSensitive bool `json:"case_sensitive"`
}

type MwordOptions struct {
	IsFirst       bool `json:"is_first"`
	NoSub         bool `json:"no_sub"`
	NoVip         bool `json:"no_vip"`
	NoRepeat      bool `json:"norepeat"`
	OneWord       bool `json:"one_word"`
	Contains      bool `json:"contains"`
	CaseSensitive bool `json:"case_sensitive"`
}

type TimerOptions struct {
	IsAnnounce bool `json:"is_announce"`
	IsAlways   bool `json:"is_always"`
}

type Limiter struct {
	Requests int           `json:"requests"` // сколько запросов
	Per      time.Duration `json:"per"`      // за какое время
	Rate     *rate.Limiter `json:"-"`
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
		Limiter: &Limiter{
			Requests: 3,
			Per:      30 * time.Second,
		},
		Spam: Spam{
			Mode:               "online",
			CheckWindowSeconds: 60,
			WhitelistUsers:     make(map[string]struct{}),
			Exceptions:         make(map[string]*ExceptionsSettings),
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
		Mword:      make(map[string]*Mword),
		MwordGroup: make(map[string]*MwordGroup),
		Aliases:    make(map[string]string),
		Markers:    make(map[string]map[string][]*Markers),
		Commands:   make(map[string]*Commands),
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
	if cfg.Automod.Delay < 0 || cfg.Automod.Delay > 10 {
		return errors.New("automod.delay must be between 0 and 10")
	}

	spam := map[string]SpamSettings{
		"default": cfg.Spam.SettingsDefault,
		"vip":     cfg.Spam.SettingsVIP,
	}

	for _, s := range spam {
		if s.SimilarityThreshold < 0.1 || cfg.Spam.SettingsDefault.SimilarityThreshold > 1 {
			return errors.New("spam.similarity_threshold must be in [0.1,1.0]")
		}
		if s.MessageLimit < 2 || s.MessageLimit > 15 {
			return errors.New("spam.message_limit must be 2..15")
		}
		if s.DurationResetPunishments < 0 {
			return errors.New("spam.reset_timeout_seconds must be >= 0")
		}
		if s.MaxWordLength < 0 {
			return errors.New("spam.max_word must be >= 0")
		}
		if s.MinGapMessages < 0 || s.MinGapMessages > 15 {
			return errors.New("spam.min_gap_messages must be in 0..15")
		}
	}

	if cfg.Spam.WhitelistUsers == nil {
		cfg.Spam.WhitelistUsers = make(map[string]struct{})
	}

	if cfg.Spam.Exceptions == nil {
		cfg.Spam.Exceptions = make(map[string]*ExceptionsSettings)
	}

	if cfg.Mword == nil {
		cfg.Mword = make(map[string]*Mword)
	}

	if cfg.MwordGroup == nil {
		cfg.MwordGroup = make(map[string]*MwordGroup)
	}

	if cfg.Aliases == nil {
		cfg.Aliases = make(map[string]string)
	}

	if cfg.Markers == nil {
		cfg.Markers = make(map[string]map[string][]*Markers)
	}

	if cfg.Commands == nil {
		cfg.Commands = make(map[string]*Commands)
	}

	if cfg.Spam.SettingsEmotes.Exceptions == nil {
		cfg.Spam.SettingsEmotes.Exceptions = make(map[string]*ExceptionsSettings)
	}

	if cfg.AliasGroups == nil {
		cfg.AliasGroups = make(map[string]*AliasGroups)
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
