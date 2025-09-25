package checker

import (
	"testing"
	"time"
	"twitchspam/internal/app/adapters/twitch/irc"
	"twitchspam/internal/app/domain/stream"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
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

	msg := &ports.ChatMessage{
		Broadcaster: ports.Broadcaster{
			UserID:   "3455435354453",
			Login:    "afsygga",
			Username: "aFsYGGA",
		},
		Chatter: ports.Chatter{
			UserID:   "3457879534798",
			Login:    "stintik",
			Username: "Stintik",
		},
		Message: ports.Message{
			ID:        "4539459834589543",
			Text:      ports.MessageText{Original: "Эта настройка определяет, сколько уникальных сообщений должно пройти между похожими сообщениями одного пользователя, прежде чем они будут считаться спамом. Например, если min_gap установлено в 2, пользователь может отправить 2 разных сообщения между повторяющимися сообщениями, чтобы избежать срабатывания антиспама. Если последовательность похожих сообщений идет подряд без достаточного количества разных сообщений, антиспам применит наказание."},
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
