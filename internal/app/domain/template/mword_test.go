package template

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
)

func TestMatchMwordRule_CaseSensitiveAlwaysMode(t *testing.T) {
	tmpl := &MwordTemplate{}

	msg := &domain.ChatMessage{
		Message: domain.Message{
			Text: domain.MessageText{Original: "беZ пОлитики"},
		},
	}

	opts := config.MwordOptions{
		CaseSensitive: true,
		Mode:          config.AlwaysMode,
	}

	word := "беZ пОлитики"

	matched := tmpl.matchMwordRule(msg, word, nil, opts, true)
	assert.True(t, matched, "ожидалось, что слово 'беZ' будет найдено при CaseSensitive=true и AlwaysMode")

	// Проверяем, что при изменённом регистре не совпадает
	word = "беz политики"
	matched = tmpl.matchMwordRule(msg, word, nil, opts, true)
	assert.False(t, matched, "ожидалось, что слово 'Без' не совпадёт при CaseSensitive=true")
}
