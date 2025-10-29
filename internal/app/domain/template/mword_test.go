package template_test

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
)

func TestMatchMwordRule_CaseSensitiveAlwaysMode(t *testing.T) {
	t.Parallel()

	tmpl := template.New(
		template.WithMword([]config.Mword{}, make(map[string]*config.MwordGroup)),
	)

	msg := &message.ChatMessage{
		Message: message.Message{
			Text: message.Text{Original: "беZ пОлитики"},
		},
	}

	matched := tmpl.Mword().Check(msg, true)
	assert.Empty(t, matched, "issuing punishments without mwords")

	trueVal := true
	alwaysModeVal := config.AlwaysMode
	tmpl.Mword().Update([]config.Mword{
		{
			Punishments: []config.Punishment{
				{
					Action:   "timeout",
					Duration: 600,
				},
			},
			Options: &config.MwordOptions{
				CaseSensitive: &trueVal,
				Mode:          &alwaysModeVal,
			},
			Word: "беZ пОлитики",
		},
	}, make(map[string]*config.MwordGroup))

	matched = tmpl.Mword().Check(msg, true)
	assert.NotEmpty(t, matched, "the punishment was not issued under the current law")

	msg = &message.ChatMessage{
		Message: message.Message{
			Text: message.Text{Original: "беz политики"},
		},
	}

	matched = tmpl.Mword().Check(msg, true)
	assert.Empty(t, matched, "the punishment was given for a word with a mismatched case")
}
