package config

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const configPath = "configs/config.json"
const channelsDir = "configs/channels"

type Manager struct {
	mu  sync.RWMutex
	cfg *Config
}

func New() (*Manager, error) {
	m := &Manager{}

	err := os.MkdirAll(channelsDir, 0750)
	if err != nil {
		return nil, err
	}

	m.cfg, err = m.readParseValidate()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if errors.Is(err, os.ErrNotExist) {
		m.cfg = m.GetDefault()
		data, err := json.Marshal(m.cfg, json.OmitZeroStructFields(true))
		if err != nil {
			return nil, fmt.Errorf("marshal config: %w", err)
		}

		if err := (*jsontext.Value)(&data).Indent(); err != nil {
			return nil, fmt.Errorf("indent: %w", err)
		}

		if err := m.writeAtomic(configPath, data, 0644); err != nil {
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

func (m *Manager) readParseValidate() (*Config, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("open/read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	channels, err := m.loadChannels()
	if err != nil {
		return nil, fmt.Errorf("load channels: %w", err)
	}
	cfg.Channels = channels

	if err := m.validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return &cfg, nil
}

func (m *Manager) loadChannels() (map[string]*Channel, error) {
	files, err := os.ReadDir(channelsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]*Channel), nil
		}
		return nil, err
	}

	channels := make(map[string]*Channel)
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(channelsDir, f.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f.Name(), err)
		}

		var ch Channel
		if err := json.Unmarshal(data, &ch); err != nil {
			return nil, fmt.Errorf("parse %s: %w", f.Name(), err)
		}

		channels[strings.ToLower(ch.Name)] = &ch
	}
	return channels, nil
}

func (m *Manager) saveLocked() error {
	if m.cfg == nil {
		return errors.New("no config to save")
	}

	if err := m.saveChannels(m.cfg.Channels); err != nil {
		return fmt.Errorf("save channels: %w", err)
	}

	cfgCopy := *m.cfg
	cfgCopy.Channels = nil

	data, err := json.Marshal(cfgCopy, json.OmitZeroStructFields(true))
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := (*jsontext.Value)(&data).Indent(); err != nil {
		return fmt.Errorf("indent: %w", err)
	}

	return m.writeAtomic(configPath, data, 0644)
}

func (m *Manager) saveChannels(channels map[string]*Channel) error {
	if err := os.MkdirAll(channelsDir, 0750); err != nil {
		return err
	}

	for name, ch := range channels {
		filename := name + ".json"
		path := filepath.Join(channelsDir, filename)

		data, err := json.Marshal(ch, json.OmitZeroStructFields(true))
		if err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}

		if err := (*jsontext.Value)(&data).Indent(); err != nil {
			return fmt.Errorf("indent: %w", err)
		}

		if err := m.writeAtomic(path, data, 0644); err != nil {
			return fmt.Errorf("write channel %s: %w", name, err)
		}
	}

	return nil
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
