package template

import (
	"testing"
	"time"
	"twitchspam/internal/app/adapters/twitch/irc"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/pkg/logger"
)

func BenchmarkCheck(b *testing.B) {
	manager, err := config.New("../../../../config.json")
	if err != nil {
		b.Fatal(err)
	}

	cfg := manager.Get()
	tIRC, _ := irc.New(logger.New(), cfg, 250*time.Millisecond, "afsygga")

	t := New(WithMword(tIRC, cfg.Mword, cfg.MwordGroup))

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
		t.Mword().Check(msg)
	}
}
