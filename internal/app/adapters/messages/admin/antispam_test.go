package admin

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"twitchspam/internal/app/infrastructure/config"
)

func TestAdmin_handleAntiSpamOnOff(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		typeSpam    string
		wantDefault bool
		wantVIP     bool
		wantEmotes  bool
	}{
		{
			name:        "enable_default",
			cmd:         "on",
			typeSpam:    "default",
			wantDefault: true,
			wantVIP:     false,
			wantEmotes:  false,
		},
		{
			name:        "disable_default",
			cmd:         "off",
			typeSpam:    "default",
			wantDefault: false,
			wantVIP:     false,
			wantEmotes:  false,
		},
		{
			name:        "enable_vip",
			cmd:         "on",
			typeSpam:    "vip",
			wantDefault: false,
			wantVIP:     true,
			wantEmotes:  false,
		},
		{
			name:        "disable_vip",
			cmd:         "off",
			typeSpam:    "vip",
			wantDefault: false,
			wantVIP:     false,
			wantEmotes:  false,
		},
		{
			name:        "enable_emote",
			cmd:         "on",
			typeSpam:    "emote",
			wantDefault: false,
			wantVIP:     false,
			wantEmotes:  true,
		},
		{
			name:        "disable_emote",
			cmd:         "off",
			typeSpam:    "emote",
			wantDefault: false,
			wantVIP:     false,
			wantEmotes:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Spam: config.Spam{
					SettingsDefault: config.SpamSettings{Enabled: false},
					SettingsVIP:     config.SpamSettings{Enabled: false},
					SettingsEmotes:  config.SpamSettingsEmote{Enabled: false},
				},
			}

			a := &Admin{}
			a.handleAntiSpamOnOff(cfg, tt.cmd, nil, tt.typeSpam)

			assert.Equal(t, tt.wantDefault, cfg.Spam.SettingsDefault.Enabled, "SettingsDefault.Enabled некорректно")
			assert.Equal(t, tt.wantVIP, cfg.Spam.SettingsVIP.Enabled, "SettingsVIP.Enabled некорректно")
			assert.Equal(t, tt.wantEmotes, cfg.Spam.SettingsEmotes.Enabled, "SettingsEmotes.Enabled некорректно")
		})
	}
}

func TestAdmin_handleMode(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		wantMode string
	}{
		{
			name:     "set_mode_always",
			cmd:      "always",
			wantMode: "always",
		},
		{
			name:     "set_mode_online",
			cmd:      "online",
			wantMode: "online",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Spam: config.Spam{Mode: ""},
			}

			a := &Admin{}
			a.handleMode(cfg, tt.cmd, nil)

			assert.Equal(t, tt.wantMode, cfg.Spam.Mode, "cfg.Spam.Mode некорректно")
		})
	}
}

func TestAdmin_handleSim(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		typeSpam    string
		wantValue   float64
		expectError bool
	}{
		{
			name:        "set_default_valid",
			args:        []string{"0.5"},
			typeSpam:    "default",
			wantValue:   0.5,
			expectError: false,
		},
		{
			name:        "set_vip_valid",
			args:        []string{"0.8"},
			typeSpam:    "vip",
			wantValue:   0.8,
			expectError: false,
		},
		{
			name:        "set_default_too_low",
			args:        []string{"0.05"},
			typeSpam:    "default",
			wantValue:   0,
			expectError: true,
		},
		{
			name:        "set_vip_too_high",
			args:        []string{"1.5"},
			typeSpam:    "vip",
			wantValue:   0,
			expectError: true,
		},
		{
			name:        "invalid_argument",
			args:        []string{"abc"},
			typeSpam:    "default",
			wantValue:   0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Spam: config.Spam{
					SettingsDefault: config.SpamSettings{SimilarityThreshold: 0},
					SettingsVIP:     config.SpamSettings{SimilarityThreshold: 0},
				},
			}

			a := &Admin{}
			resp := a.handleSim(cfg, "", tt.args, tt.typeSpam)

			if tt.typeSpam == "vip" {
				if !tt.expectError {
					assert.Equal(t, tt.wantValue, cfg.Spam.SettingsVIP.SimilarityThreshold, "VIP SimilarityThreshold некорректен")
				} else {
					assert.Equal(t, 0.0, cfg.Spam.SettingsVIP.SimilarityThreshold)
				}
			} else {
				if !tt.expectError {
					assert.Equal(t, tt.wantValue, cfg.Spam.SettingsDefault.SimilarityThreshold, "Default SimilarityThreshold некорректен")
				} else {
					assert.Equal(t, 0.0, cfg.Spam.SettingsDefault.SimilarityThreshold)
				}
			}

			if tt.expectError {
				assert.NotNil(t, resp)
				assert.Equal(t, "значение порога схожести сообщений должно быть от 0.1 до 1.0!", resp.Text[0])
			} else {
				assert.Nil(t, resp)
			}
		})
	}
}

func TestAdmin_handleMsg(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		typeSpam    string
		wantValue   int
		expectError bool
	}{
		{
			name:        "set_default_valid",
			args:        []string{"5"},
			typeSpam:    "default",
			wantValue:   5,
			expectError: false,
		},
		{
			name:        "set_vip_valid",
			args:        []string{"10"},
			typeSpam:    "vip",
			wantValue:   10,
			expectError: false,
		},
		{
			name:        "set_emote_valid",
			args:        []string{"7"},
			typeSpam:    "emote",
			wantValue:   7,
			expectError: false,
		},
		{
			name:        "set_default_too_low",
			args:        []string{"1"},
			typeSpam:    "default",
			wantValue:   0,
			expectError: true,
		},
		{
			name:        "set_vip_too_high",
			args:        []string{"20"},
			typeSpam:    "vip",
			wantValue:   0,
			expectError: true,
		},
		{
			name:        "invalid_argument",
			args:        []string{"abc"},
			typeSpam:    "default",
			wantValue:   0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Spam: config.Spam{
					SettingsDefault: config.SpamSettings{MessageLimit: 0},
					SettingsVIP:     config.SpamSettings{MessageLimit: 0},
					SettingsEmotes:  config.SpamSettingsEmote{MessageLimit: 0},
				},
			}

			a := &Admin{}
			resp := a.handleMsg(cfg, "", tt.args, tt.typeSpam)

			switch tt.typeSpam {
			case "vip":
				if !tt.expectError {
					assert.Equal(t, tt.wantValue, cfg.Spam.SettingsVIP.MessageLimit, "VIP MessageLimit некорректен")
				} else {
					assert.Equal(t, 0, cfg.Spam.SettingsVIP.MessageLimit)
				}
			case "emote":
				if !tt.expectError {
					assert.Equal(t, tt.wantValue, cfg.Spam.SettingsEmotes.MessageLimit, "Emote MessageLimit некорректен")
				} else {
					assert.Equal(t, 0, cfg.Spam.SettingsEmotes.MessageLimit)
				}
			default:
				if !tt.expectError {
					assert.Equal(t, tt.wantValue, cfg.Spam.SettingsDefault.MessageLimit, "Default MessageLimit некорректен")
				} else {
					assert.Equal(t, 0, cfg.Spam.SettingsDefault.MessageLimit)
				}
			}

			if tt.expectError {
				assert.NotNil(t, resp)
				assert.Equal(t, "значение лимита сообщений должно быть от 2 до 15!", resp.Text[0])
			} else {
				assert.Nil(t, resp)
			}
		})
	}
}

func TestAdmin_handlePunishments(t *testing.T) {
	tests := []struct {
		name       string
		typeSpam   string
		args       []string
		wantErr    bool
		wantPunish []config.Punishment
	}{
		{
			name:       "no_arguments",
			typeSpam:   "default",
			args:       []string{},
			wantErr:    true,
			wantPunish: nil,
		},
		{
			name:       "valid_default",
			typeSpam:   "default",
			args:       []string{"-,600"},
			wantErr:    false,
			wantPunish: []config.Punishment{{Action: "delete"}, {Action: "timeout", Duration: 600}},
		},
		{
			name:     "invalid_punishment",
			typeSpam: "vip",
			args:     []string{"abc"},
			wantErr:  true,
		},
		{
			name:     "inherit_default_error",
			typeSpam: "default",
			args:     []string{"inherit"},
			wantErr:  true,
		},
		{
			name:       "inherit_vip_success",
			typeSpam:   "vip",
			args:       []string{"*"},
			wantErr:    false,
			wantPunish: []config.Punishment{{Action: "timeout", Duration: 30}, {Action: "delete"}},
		},
		{
			name:       "mixed_valid_and_inherit",
			typeSpam:   "vip",
			args:       []string{"-,600,*,1800"},
			wantErr:    false,
			wantPunish: []config.Punishment{{Action: "timeout", Duration: 30}, {Action: "delete"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Spam: config.Spam{
					SettingsDefault: config.SpamSettings{
						Punishments: []config.Punishment{{Action: "timeout", Duration: 30}, {Action: "delete"}},
					},
					SettingsVIP:    config.SpamSettings{},
					SettingsEmotes: config.SpamSettingsEmote{},
				},
			}

			a := &Admin{}
			resp := a.handlePunishments(cfg, "", tt.args, tt.typeSpam)

			if tt.wantErr {
				assert.NotNil(t, resp)
			} else {
				assert.Nil(t, resp)
				switch tt.typeSpam {
				case "default":
					assert.Equal(t, tt.wantPunish, cfg.Spam.SettingsDefault.Punishments)
				case "vip":
					assert.Equal(t, tt.wantPunish, cfg.Spam.SettingsVIP.Punishments)
				case "emote":
					assert.Equal(t, tt.wantPunish, cfg.Spam.SettingsEmotes.Punishments)
				}
			}
		})
	}
}

func TestAdmin_handleDurationResetPunishments(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		typeSpam    string
		wantValue   int
		expectError bool
	}{
		{
			name:        "set_default_valid",
			args:        []string{"100"},
			typeSpam:    "default",
			wantValue:   100,
			expectError: false,
		},
		{
			name:        "set_vip_valid",
			args:        []string{"3600"},
			typeSpam:    "vip",
			wantValue:   3600,
			expectError: false,
		},
		{
			name:        "set_emote_valid",
			args:        []string{"86400"},
			typeSpam:    "emote",
			wantValue:   86400,
			expectError: false,
		},
		{
			name:        "set_default_too_low",
			args:        []string{"0"},
			typeSpam:    "default",
			wantValue:   0,
			expectError: true,
		},
		{
			name:        "set_vip_too_high",
			args:        []string{"90000"},
			typeSpam:    "vip",
			wantValue:   0,
			expectError: true,
		},
		{
			name:        "invalid_argument",
			args:        []string{"abc"},
			typeSpam:    "emote",
			wantValue:   0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Spam: config.Spam{
					SettingsDefault: config.SpamSettings{DurationResetPunishments: 0},
					SettingsVIP:     config.SpamSettings{DurationResetPunishments: 0},
					SettingsEmotes:  config.SpamSettingsEmote{DurationResetPunishments: 0},
				},
			}

			a := &Admin{}
			resp := a.handleDurationResetPunishments(cfg, "", tt.args, tt.typeSpam)

			var gotValue int
			switch tt.typeSpam {
			case "vip":
				gotValue = cfg.Spam.SettingsVIP.DurationResetPunishments
			case "emote":
				gotValue = cfg.Spam.SettingsEmotes.DurationResetPunishments
			default:
				gotValue = cfg.Spam.SettingsDefault.DurationResetPunishments
			}
			assert.Equal(t, tt.wantValue, gotValue, "DurationResetPunishments некорректен")

			if tt.expectError {
				assert.NotNil(t, resp)
				assert.Contains(t, resp.Text, "значение времени сброса наказаний должно быть от 1 до 86400!")
			} else {
				assert.Nil(t, resp)
			}
		})
	}
}

func TestAdmin_handleMaxLen(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		typeSpam    string
		wantValue   int
		expectError bool
	}{
		// Default
		{
			name:        "default_valid",
			args:        []string{"100"},
			typeSpam:    "default",
			wantValue:   100,
			expectError: false,
		},
		{
			name:        "default_too_high",
			args:        []string{"600"},
			typeSpam:    "default",
			wantValue:   0,
			expectError: true,
		},
		{
			name:        "default_invalid",
			args:        []string{"abc"},
			typeSpam:    "default",
			wantValue:   0,
			expectError: true,
		},
		// VIP
		{
			name:        "vip_valid",
			args:        []string{"250"},
			typeSpam:    "vip",
			wantValue:   250,
			expectError: false,
		},
		{
			name:        "vip_too_high",
			args:        []string{"600"},
			typeSpam:    "vip",
			wantValue:   0,
			expectError: true,
		},
		// Emote
		{
			name:        "emote_valid",
			args:        []string{"20"},
			typeSpam:    "emote",
			wantValue:   20,
			expectError: false,
		},
		{
			name:        "emote_too_high",
			args:        []string{"50"},
			typeSpam:    "emote",
			wantValue:   0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Spam: config.Spam{
					SettingsDefault: config.SpamSettings{MaxWordLength: 0},
					SettingsVIP:     config.SpamSettings{MaxWordLength: 0},
					SettingsEmotes:  config.SpamSettingsEmote{MaxEmotesLength: 0},
				},
			}

			a := &Admin{}
			resp := a.handleMaxLen(cfg, "", tt.args, tt.typeSpam)

			var gotValue int
			switch tt.typeSpam {
			case "vip":
				gotValue = cfg.Spam.SettingsVIP.MaxWordLength
			case "emote":
				gotValue = cfg.Spam.SettingsEmotes.MaxEmotesLength
			default:
				gotValue = cfg.Spam.SettingsDefault.MaxWordLength
			}
			assert.Equal(t, tt.wantValue, gotValue, "MaxLen некорректен")

			if tt.expectError {
				assert.NotNil(t, resp)
				if tt.typeSpam == "emote" {
					assert.Contains(t, resp.Text, "значение максимального количества эмоутов должно быть от 0 до 30!")
				} else {
					assert.Contains(t, resp.Text, "значение максимальной длины слова должно быть от 0 до 500!")
				}
			} else {
				assert.Nil(t, resp)
			}
		})
	}
}

func TestAdmin_handleMaxPunishment(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		typeSpam    string
		wantPunish  config.Punishment
		expectError bool
	}{
		{
			name:       "default_delete",
			args:       []string{"-"},
			typeSpam:   "default",
			wantPunish: config.Punishment{Action: "delete"},
		},
		{
			name:       "vip_timeout",
			args:       []string{"600"},
			typeSpam:   "vip",
			wantPunish: config.Punishment{Action: "timeout", Duration: 600},
		},
		{
			name:       "emote_delete",
			args:       []string{"-"},
			typeSpam:   "emote",
			wantPunish: config.Punishment{Action: "delete"},
		},
		{
			name:       "default_inherit",
			args:       []string{"*"},
			typeSpam:   "default",
			wantPunish: config.Punishment{Action: "delete"},
		},
		{
			name:        "invalid_argument",
			args:        []string{"abc"},
			typeSpam:    "default",
			expectError: true,
		},
		{
			name:        "empty_args",
			args:        []string{},
			typeSpam:    "vip",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Spam: config.Spam{
					SettingsDefault: config.SpamSettings{
						MaxWordPunishment: config.Punishment{},
						Punishments:       []config.Punishment{{Action: "delete"}},
					},
					SettingsVIP: config.SpamSettings{
						MaxWordPunishment: config.Punishment{},
						Punishments:       []config.Punishment{{Action: "timeout", Duration: 300}},
					},
					SettingsEmotes: config.SpamSettingsEmote{
						MaxEmotesPunishment: config.Punishment{},
						Punishments:         []config.Punishment{{Action: "delete"}},
					},
				},
			}

			a := &Admin{}
			resp := a.handleMaxPunishment(cfg, "", tt.args, tt.typeSpam)

			var got config.Punishment
			switch tt.typeSpam {
			case "vip":
				got = cfg.Spam.SettingsVIP.MaxWordPunishment
			case "emote":
				got = cfg.Spam.SettingsEmotes.MaxEmotesPunishment
			default:
				got = cfg.Spam.SettingsDefault.MaxWordPunishment
			}

			if tt.expectError {
				assert.NotNil(t, resp)
			} else {
				assert.Nil(t, resp)
				assert.Equal(t, tt.wantPunish, got, "MaxPunishment некорректен")
			}
		})
	}
}

func TestAdmin_handleMinGap(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		typeSpam    string
		wantValue   int
		expectError bool
	}{
		// Default
		{
			name:        "default_valid",
			args:        []string{"5"},
			typeSpam:    "default",
			wantValue:   5,
			expectError: false,
		},
		{
			name:        "default_too_high",
			args:        []string{"20"},
			typeSpam:    "default",
			wantValue:   0,
			expectError: true,
		},
		{
			name:        "default_invalid",
			args:        []string{"abc"},
			typeSpam:    "default",
			wantValue:   0,
			expectError: true,
		},
		// VIP
		{
			name:        "vip_valid",
			args:        []string{"10"},
			typeSpam:    "vip",
			wantValue:   10,
			expectError: false,
		},
		{
			name:        "vip_too_high",
			args:        []string{"30"},
			typeSpam:    "vip",
			wantValue:   0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Spam: config.Spam{
					SettingsDefault: config.SpamSettings{MinGapMessages: 0},
					SettingsVIP:     config.SpamSettings{MinGapMessages: 0},
				},
			}

			a := &Admin{}
			resp := a.handleMinGap(cfg, "", tt.args, tt.typeSpam)

			// Проверяем значение в конфиге
			var got int
			if tt.typeSpam == "vip" {
				got = cfg.Spam.SettingsVIP.MinGapMessages
			} else {
				got = cfg.Spam.SettingsDefault.MinGapMessages
			}
			assert.Equal(t, tt.wantValue, got, "MinGapMessages некорректен")

			// Проверяем ответ функции
			if tt.expectError {
				assert.NotNil(t, resp)
				assert.Contains(t, resp.Text, "значение должно быть от 0 до 15!")
			} else {
				assert.Nil(t, resp)
			}
		})
	}
}

func TestAdmin_handleTime(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantValue   int
		expectError bool
	}{
		{
			name:        "valid_value",
			args:        []string{"100"},
			wantValue:   100,
			expectError: false,
		},
		{
			name:        "too_low",
			args:        []string{"0"},
			wantValue:   0,
			expectError: true,
		},
		{
			name:        "too_high",
			args:        []string{"500"},
			wantValue:   0,
			expectError: true,
		},
		{
			name:        "invalid_argument",
			args:        []string{"abc"},
			wantValue:   0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Spam: config.Spam{
					CheckWindowSeconds: 0,
				},
			}

			a := &Admin{}
			resp := a.handleTime(cfg, "", tt.args)

			assert.Equal(t, tt.wantValue, cfg.Spam.CheckWindowSeconds, "CheckWindowSeconds некорректен")

			if tt.expectError {
				assert.NotNil(t, resp)
				assert.Contains(t, resp.Text, "значение окна проверки сообщений должно быть от 1 до 300!")
			} else {
				assert.Nil(t, resp)
			}
		})
	}
}

func TestAdmin_handleAdd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		initialList []string
		wantList    []string
		wantText    string
	}{
		{
			name:        "add_new_users",
			args:        []string{"user1", "user2"},
			initialList: []string{},
			wantList:    []string{"user1", "user2"},
			wantText:    "добавлены в список: user1, user2!",
		},
		{
			name:        "add_existing_user",
			args:        []string{"user1", "user2"},
			initialList: []string{"user1"},
			wantList:    []string{"user1", "user2"},
			wantText:    "добавлены в список: user2 • уже в списке: user1!",
		},
		{
			name:        "add_all_existing",
			args:        []string{"user1", "user2"},
			initialList: []string{"user1", "user2"},
			wantList:    []string{"user1", "user2"},
			wantText:    "уже в списке: user1, user2!",
		},
		{
			name:        "add_empty_args",
			args:        []string{},
			initialList: []string{},
			wantList:    []string{},
			wantText:    "", // возвращает NonParametr, можно проверить отдельно
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Spam: config.Spam{
					WhitelistUsers: append([]string{}, tt.initialList...),
				},
			}

			a := &Admin{}
			resp := a.handleAdd(cfg, "", tt.args)

			assert.Equal(t, tt.wantList, cfg.Spam.WhitelistUsers, "WhitelistUsers некорректен")
			if tt.wantText == "" {
				assert.Equal(t, NonParametr, resp)
			} else {
				assert.NotNil(t, resp)
				assert.Equal(t, tt.wantText, resp.Text[0])
			}
		})
	}
}

func TestAdmin_handleDel(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		initialList []string
		wantList    []string
		wantText    string
	}{
		{
			name:        "remove_existing_users",
			args:        []string{"user1", "user2"},
			initialList: []string{"user1", "user2", "user3"},
			wantList:    []string{"user3"},
			wantText:    "удалены из списка: user1, user2!",
		},
		{
			name:        "remove_some_not_found",
			args:        []string{"user1", "user4"},
			initialList: []string{"user1", "user2"},
			wantList:    []string{"user2"},
			wantText:    "удалены из списка: user1 • нет в списке: user4!",
		},
		{
			name:        "remove_all_not_found",
			args:        []string{"user4", "user5"},
			initialList: []string{"user1", "user2"},
			wantList:    []string{"user1", "user2"},
			wantText:    "нет в списке: user4, user5!",
		},
		{
			name:        "remove_empty_args",
			args:        []string{},
			initialList: []string{"user1"},
			wantList:    []string{"user1"},
			wantText:    "", // возвращает NonParametr
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Spam: config.Spam{
					WhitelistUsers: append([]string{}, tt.initialList...),
				},
			}

			a := &Admin{}
			resp := a.handleDel(cfg, "", tt.args)

			assert.Equal(t, tt.wantList, cfg.Spam.WhitelistUsers, "WhitelistUsers некорректен")
			if tt.wantText == "" {
				assert.Equal(t, NonParametr, resp)
			} else {
				assert.NotNil(t, resp)
				assert.Equal(t, tt.wantText, resp.Text[0])
			}
		})
	}
}
