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

type Manager struct {
	mu   sync.RWMutex
	cfg  *Config
	path string
}

func New(path string) (*Manager, error) {
	m := &Manager{path: path}

	var err error
	m.cfg, err = m.readParseValidate(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read config: %w", err)
	}

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
	if path == "" {
		return nil, errors.New("no config path provided")
	}

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
