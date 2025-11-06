package template_test

import (
	"github.com/stretchr/testify/assert"
	"regexp"
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
				NoRepeat: &trueVal,
				Mode:     &alwaysModeVal,
			},
			NameRegexp: "phone",
			Regexp:     regexp.MustCompile(`\b(?:4(?:\d{15}|\d{3}(?: \d{4}){3})|5[1-5](?:\d{14}|\d{2}(?: \d{4}){3})|(?:222[1-9]|22[3-9]\d|2[3-6]\d{2}|27[0-1]\d|2720)(?:\d{12}|(?: \d{4}){3})|(?:2200|2201|2202|2203|2204)(?:\d{12}|(?: \d{4}){3}))\b`),
		},
	}, make(map[string]*config.MwordGroup))

	msg := &message.ChatMessage{
		Message: message.Message{
			Text: message.Text{Original: "у кого есть возможность киньте на сигареты.курить хочу жесть 2200154518300289"},
		},
	}

	_, matched := tmpl.Mword().Check(msg, true)
	assert.NotEmpty(t, matched, "the punishment was given for a word with a mismatched case")
}
