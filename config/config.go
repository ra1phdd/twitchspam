package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
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
	Enabled             bool    `json:"enabled"`               // !am - включить/выключить автомод
	PauseSeconds        int     `json:"pause_seconds"`         // !am <кол-во сек> - приостановка антифлуда
	Mode                string  `json:"mode"`                  // !am online/always - только на стриме/всегда
	SimilarityThreshold float64 `json:"similarity_threshold"`  // !am sim <0.0-1.0> - степень схожести сообщений
	MessageLimit        int     `json:"message_limit"`         // !am msg <кол-во сообщений (2-15) или off>
	CheckWindowSeconds  int     `json:"check_window_seconds"`  // !am time <секунды, макс 300> - за какой период проверять сообщения
	VIPEnabled          bool    `json:"vip_enabled"`           // !am vip - работать на випов
	Timeouts            []int   `json:"timeouts"`              // !am to <значения через пробел> - длительности таймаутов
	ResetTimeoutSeconds int     `json:"reset_timeout_seconds"` // !am rto <значение> - время сброса таймаутов
}
type Config struct {
	App      App      `json:"app"`
	Spam     Spam     `json:"spam"`
	Banwords []string `json:"banwords"`
}

var current atomic.Value // *Config
var loadedPath atomic.Pointer[string]

func Load(path string) (*Config, error) {
	cfg, err := readParseValidate(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg = &Config{}
			applyDefaults(cfg)

			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("marshal config: %w", err)
			}

			if err := writeAtomic(path, data, 0644); err != nil {
				return nil, fmt.Errorf("write config: %w", err)
			}
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	current.Store(cfg)
	loadedPath.Store(&path)
	return cfg, nil
}

func Get() *Config {
	cfg, _ := current.Load().(*Config)
	return cfg
}

func Update(modify func(cfg *Config)) error {
	old := Get()
	if old == nil {
		return errors.New("no config loaded")
	}

	newCfg := *old
	modify(&newCfg)

	applyDefaults(&newCfg)
	if err := validate(&newCfg); err != nil {
		return fmt.Errorf("invalid config update: %w", err)
	}

	current.Store(&newCfg)
	return Save()
}

func Save() error {
	path := loadedPath.Load()
	if path == nil {
		return errors.New("no config file loaded")
	}

	cfg := Get()
	if cfg == nil {
		return errors.New("no config to save")
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return writeAtomic(*path, data, 0644)
}

func readParseValidate(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("open/read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	applyDefaults(&cfg)
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.App.LogLevel == "" {
		cfg.App.LogLevel = "info"
	}

	if cfg.Spam.SimilarityThreshold <= 0 {
		cfg.Spam.SimilarityThreshold = 0.7
	}
	if cfg.Spam.PauseSeconds < 0 {
		cfg.Spam.PauseSeconds = 0
	}
	if cfg.Spam.Mode == "" {
		cfg.Spam.Mode = "online"
	}
	if cfg.Spam.MessageLimit < 2 || cfg.Spam.MessageLimit > 15 {
		cfg.Spam.MessageLimit = 3
	}
	if cfg.Spam.CheckWindowSeconds < 0 || cfg.Spam.CheckWindowSeconds > 300 {
		cfg.Spam.CheckWindowSeconds = 60
	}
	if cfg.Spam.Timeouts == nil || len(cfg.Spam.Timeouts) == 0 {
		cfg.Spam.Timeouts = []int{60, 600, 1800}
	}
	if cfg.Spam.ResetTimeoutSeconds <= 0 {
		cfg.Spam.ResetTimeoutSeconds = 60
	}
}

func validate(cfg *Config) error {
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

	return nil
}

func writeAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp := filepath.Join(dir, fmt.Sprintf(".%s.tmp-%d", base, time.Now().UnixNano()))

	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
