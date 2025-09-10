package admin

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"twitchspam/internal/app/infrastructure/config"
)

type mockAliasesPort struct {
	data map[string]string
}

func (m *mockAliasesPort) ReplaceOne(text string) string {
	return text
}

func (m *mockAliasesPort) ReplacePlaceholders(text string, parts []string) string {
	return text
}

func (m *mockAliasesPort) Update(aliases map[string]string) {
	m.data = make(map[string]string)
	for k, v := range aliases {
		m.data[k] = v
	}
}

func TestAdmin_handleAliasesAdd(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantText   string
		wantInCfg  map[string]string
		expectFail bool
	}{
		{
			name:     "valid_alias_with_from",
			args:     []string{"!hi", "from", "!hello"},
			wantText: "алиас `!hi` добавлен для команды `!hello`!",
			wantInCfg: map[string]string{
				"!hi": "!hello",
			},
			expectFail: false,
		},
		{
			name:     "without_exclamation_prefix",
			args:     []string{"bye", "from", "goodbye"},
			wantText: "алиас `!bye` добавлен для команды `!goodbye`!",
			wantInCfg: map[string]string{
				"!bye": "!goodbye",
			},
			expectFail: false,
		},
		{
			name:       "invalid_syntax",
			args:       []string{"only", "two", "three"},
			wantText:   "некорректный синтаксис!",
			wantInCfg:  map[string]string{},
			expectFail: true,
		},
		{
			name:     "multiple_words_before_and_after_from",
			args:     []string{"say", "hello", "from", "greet", "user"},
			wantText: "алиас `!say hello` добавлен для команды `!greet user`!",
			wantInCfg: map[string]string{
				"!say hello": "!greet user",
			},
			expectFail: false,
		},
		{
			name:       "no_from_keyword",
			args:       []string{"only", "two", "words"},
			wantText:   "некорректный синтаксис!",
			wantInCfg:  map[string]string{},
			expectFail: true,
		},
		{
			name:       "from_at_start",
			args:       []string{"from", "hello", "man"},
			wantText:   "некорректный синтаксис!",
			wantInCfg:  map[string]string{},
			expectFail: true,
		},
		{
			name:       "from_at_end",
			args:       []string{"hello", "man", "from"},
			wantText:   "некорректный синтаксис!",
			wantInCfg:  map[string]string{},
			expectFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Aliases: make(map[string]string),
			}

			mockPort := &mockAliasesPort{
				data: make(map[string]string),
			}
			a := &Admin{
				aliases: mockPort,
			}

			resp := a.handleAliasesAdd(cfg, "", tt.args)
			assert.Equal(t, tt.wantText, resp.Text[0], "Сообщение ответа некорректное")
			assert.Equal(t, tt.wantInCfg, cfg.Aliases, "cfg.Aliases не соответствует ожидаемому")
			assert.Equal(t, tt.wantInCfg, mockPort.data, "mockPort.data не соответствует ожидаемому")
		})
	}
}

func TestAdmin_handleAliasesDel(t *testing.T) {
	type args struct {
		initialAliases map[string]string
		args           []string
	}
	tests := []struct {
		name       string
		args       args
		wantText   string
		wantInCfg  map[string]string
		expectFail bool
	}{
		{
			name: "delete_existing_alias",
			args: args{
				initialAliases: map[string]string{"!hi": "!hello"},
				args:           []string{"!hi"},
			},
			wantText:   "алиас `!hi` удален!",
			wantInCfg:  map[string]string{},
			expectFail: false,
		},
		{
			name: "delete_existing_alias_without_exclamation",
			args: args{
				initialAliases: map[string]string{"!bye": "!goodbye"},
				args:           []string{"bye"},
			},
			wantText:   "алиас `!bye` удален!",
			wantInCfg:  map[string]string{},
			expectFail: false,
		},
		{
			name: "delete_non_existing_alias",
			args: args{
				initialAliases: map[string]string{"!hello": "!hi"},
				args:           []string{"!unknown"},
			},
			wantText:   "алиас не найден!",
			wantInCfg:  map[string]string{"!hello": "!hi"},
			expectFail: true,
		},
		{
			name: "no_arguments_provided",
			args: args{
				initialAliases: map[string]string{"!hello": "!hi"},
				args:           []string{},
			},
			wantText:   "некорректный синтаксис!",
			wantInCfg:  map[string]string{"!hello": "!hi"},
			expectFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Aliases: make(map[string]string),
			}
			for k, v := range tt.args.initialAliases {
				cfg.Aliases[k] = v
			}

			mockPort := &mockAliasesPort{
				data: make(map[string]string),
			}
			a := &Admin{
				aliases: mockPort,
			}

			resp := a.handleAliasesDel(cfg, "", tt.args.args)

			if tt.expectFail && len(tt.args.args) == 0 {
				assert.Equal(t, NonParametr.Text[0], resp.Text[0], "Сообщение ответа некорректное")
			} else {
				assert.Equal(t, tt.wantText, resp.Text[0], "Сообщение ответа некорректное")
			}

			assert.Equal(t, tt.wantInCfg, cfg.Aliases, "cfg.Aliases не соответствует ожидаемому")
			if !tt.expectFail || (len(tt.args.args) > 0 && cfg.Aliases[tt.args.args[0]] != "") {
				assert.Equal(t, tt.wantInCfg, mockPort.data, "mockPort.data не соответствует ожидаемому")
			}
		})
	}
}
