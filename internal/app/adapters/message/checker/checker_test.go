package checker

import (
	"testing"
	"time"
	"twitchspam/internal/app/adapters/platform/twitch/irc"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/domain/stream"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/pkg/logger"
)

func BenchmarkCheck(b *testing.B) {
	manager, err := config.New("../../../../../config.json")
	if err != nil {
		b.Fatal(err)
	}

	cfg := manager.Get()

	s := stream.NewStream("afsygga")
	t := template.New(logger.New(), cfg.Aliases, cfg.Banwords.Words, cfg.Banwords.Regexp, s)
	i, _ := irc.New(logger.New(), cfg, 100*time.Millisecond, "afsygga")
	c := NewCheck(logger.New(), cfg, s, stream.New(), t, i)

	msg := &domain.ChatMessage{
		Broadcaster: domain.Broadcaster{
			UserID:   "3455435354453",
			Login:    "afsygga",
			Username: "aFsYGGA",
		},
		Chatter: domain.Chatter{
			UserID:   "3457879534798",
			Login:    "stintik",
			Username: "Stintik",
		},
		Message: domain.Message{
			ID:        "4539459834589543",
			Text:      domain.MessageText{Original: "че с онлайном"},
			EmoteOnly: false,
			Emotes:    nil,
		},
		Reply: nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Check(msg)
	}
}
