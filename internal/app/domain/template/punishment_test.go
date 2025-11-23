package template_test

import (
	"testing"
	"twitchspam/internal/app/domain/template"
)

func TestParsePunishment(t *testing.T) {
	t.Parallel()
	p := template.NewPunishment()

	tests := []struct {
		name       string
		input      string
		inherit    bool
		wantAction string
		wantDur    int
		wantErr    bool
	}{
		{"inherit enabled", "*", true, "inherit", 0, false},
		{"inherit disabled", "*", false, "", 0, true},
		{"none", "none", false, "none", 0, false},
		{"n", "n", false, "none", 0, false},
		{"delete", "delete", false, "delete", 0, false},
		{"d", "d", false, "delete", 0, false},
		{"warn", "warn", false, "warn", 0, false},
		{"w", "w", false, "warn", 0, false},
		{"ban", "ban", false, "ban", 0, false},
		{"b", "b", false, "ban", 0, false},
		{"zero-ban", "0", false, "ban", 0, false},

		{"numeric timeout", "60", false, "timeout", 60, false},
		{"invalid numeric", "99999999", false, "", 0, true},

		{"1s", "1s", false, "timeout", 1, false},
		{"3m", "3m", false, "timeout", 180, false},
		{"5h", "5h", false, "timeout", 5 * 3600, false},
		{"7d", "7d", false, "timeout", 7 * 86400, false},
		{"2w", "2w", false, "timeout", 2 * 604800, false},

		{"1с", "1с", false, "timeout", 1, false},
		{"3м", "3м", false, "timeout", 180, false},
		{"5ч", "5ч", false, "timeout", 5 * 3600, false},
		{"7д", "7д", false, "timeout", 7 * 86400, false},
		{"2н", "2н", false, "timeout", 2 * 604800, false},

		{"1w5d3m", "1w5d3m", false, "timeout", 604800 + 5*86400 + 180, false},
		{"1н5д3м", "1н5д3м", false, "timeout", 604800 + 5*86400 + 180, false},

		{"bad format", "5x", false, "", 0, true},
		{"empty", "", false, "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := p.Parse(tt.input, tt.inherit)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ожидалась ошибка, но её нет")
				}
				return
			}

			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}

			if got.Action != tt.wantAction {
				t.Fatalf("действие: получили %q, ожидалось %q", got.Action, tt.wantAction)
			}

			if got.Duration != tt.wantDur {
				t.Fatalf("длительность: получили %d, ожидалось %d", got.Duration, tt.wantDur)
			}
		})
	}
}
