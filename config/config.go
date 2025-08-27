package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type App struct {
	LogLevel    string   `json:"log_level"`
	OAuth       string   `json:"oauth,required"`
	ClientID    string   `json:"client_id,required"`
	Username    string   `json:"username,required"`
	UserID      string   `json:"user_id,required"`
	ModChannels []string `json:"mod_channels,required"`
}

type Spam struct {
	Enabled             bool           `json:"enabled"`               // !am - включить/выключить автомод
	PauseSeconds        int            `json:"pause_seconds"`         // !am <кол-во сек> - приостановка антифлуда
	Mode                string         `json:"mode"`                  // !am online/always - только на стриме/всегда
	SimilarityThreshold float64        `json:"similarity_threshold"`  // !am sim <0.0-1.0> - степень схожести сообщений
	MessageLimit        int            `json:"message_limit"`         // !am msg <кол-во сообщений (2-15) или off>
	CheckWindowSeconds  int            `json:"check_window_seconds"`  // !am time <секунды, макс 300> - за какой период проверять сообщения
	VIPEnabled          bool           `json:"vip_enabled"`           // !am vip - работать на випов
	Timeouts            []int          `json:"timeouts"`              // !am to <значения через пробел> - длительности таймаутов
	ResetTimeoutSeconds int            `json:"reset_timeout_seconds"` // !am rto <значение> - время сброса таймаутов
	MaxWordLength       int            `json:"max_word_length"`       // !am mw <значение или 0 для оффа> - максимальная длина слова
	MaxWordTimeoutTime  int            `json:"max_word_timeout_time"` // !am mwt <кол-во сек>
	MinGapMessages      int            `json:"min_gap_messages"`      // !am min_gap <значение от 0 до 15> - сколько сообщений между спамом должно быть чтобы это не считалось спамом
	WhitelistUsers      []string       `json:"whitelist_users"`       // !am add/del <пользователи через запятую>
	SpamExceptions      map[string]int `json:"spam_exceptions"`       // !am except <слова/фразы через запятую> <таймаут в секундах> - исключения из спама (таймаут за ку 30 сек пример)
	DelayAutomod        int            `json:"delay_automod"`         // !am da <кол-во сек от 0 до 10> - задержка срабатывания на автомод (чтобы модеры не обленились)
}

type Config struct {
	App      App      `json:"app"`
	Spam     Spam     `json:"spam"`
	Banwords []string `json:"banwords"`
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
			m.cfg = &Config{}
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

	if cfg.Spam.SimilarityThreshold < 0 || cfg.Spam.SimilarityThreshold > 1 {
		return errors.New("spam.similarity_threshold must be in [0,1]")
	}
	if cfg.Spam.PauseSeconds < 0 {
		return errors.New("spam.pause_seconds must be >= 0")
	}
	if cfg.Spam.Mode != "online" && cfg.Spam.Mode != "always" {
		return errors.New("spam.mode must be 'online' or 'always'")
	}
	if cfg.Spam.MessageLimit < 2 || cfg.Spam.MessageLimit > 15 {
		return errors.New("spam.message_limit must be 2..15")
	}
	if cfg.Spam.CheckWindowSeconds < 1 || cfg.Spam.CheckWindowSeconds > 300 {
		return errors.New("spam.check_window_seconds must be 1..300")
	}
	if cfg.Spam.ResetTimeoutSeconds < 0 {
		return errors.New("spam.reset_timeout_seconds must be >= 0")
	}
	if cfg.Spam.MaxWordLength < 0 {
		return errors.New("spam.max_word must be >= 0")
	}
	if cfg.Spam.MinGapMessages < 0 || cfg.Spam.MinGapMessages > 15 {
		return errors.New("spam.min_gap_messages must be in 0..15")
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
