package template

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
)

func TestMatchMwordRule_CaseSensitiveAlwaysMode(t *testing.T) {
	tmpl := New(
		WithMword([]config.Mword{}, make(map[string]*config.MwordGroup)),
	)

	msg := &domain.ChatMessage{
		Message: domain.Message{
			Text: domain.MessageText{Original: "беZ пОлитики"},
		},
	}

	matched := tmpl.Mword().Check(msg, true)
	assert.False(t, len(matched) > 0, "issuing punishments without mwords")

	tmpl.Mword().Update([]config.Mword{
		{
			Punishments: []config.Punishment{
				{
					Action:   "timeout",
					Duration: 600,
				},
			},
			Options: config.MwordOptions{
				CaseSensitive: true,
				Mode:          config.AlwaysMode,
			},
			Word: "беZ пОлитики",
		},
	}, make(map[string]*config.MwordGroup))

	matched = tmpl.Mword().Check(msg, true)
	assert.True(t, len(matched) > 0, "the punishment was not issued under the current law")

	msg = &domain.ChatMessage{
		Message: domain.Message{
			Text: domain.MessageText{Original: "беz политики"},
		},
	}

	matched = tmpl.Mword().Check(msg, true)
	assert.False(t, len(matched) > 0, "the punishment was given for a word with a mismatched case")
}
