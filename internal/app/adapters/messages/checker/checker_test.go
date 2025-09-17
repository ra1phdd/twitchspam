package checker

import (
	"github.com/dlclark/regexp2"
	"runtime"
	"strings"
	"testing"
	"time"
	"twitchspam/internal/app/domain/stream"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

func BenchmarkCheckMwords(b *testing.B) {
	cfg, err := config.New("../../../../../config.json")
	if err != nil {
		b.Fatal(err)
	}

	c := &Checker{
		cfg:      cfg.Get(),
		template: template.New(logger.New(), make(map[string]string), make([]string, 0), make([]*regexp2.Regexp, 0), cfg.Get().MwordGroup, cfg.Get().Mword, stream.NewStream("stintik")),
		timeouts: struct {
			spam             ports.StorePort[Empty]
			emote            ports.StorePort[Empty]
			exceptionsSpam   ports.StorePort[Empty]
			exceptionsEmotes ports.StorePort[Empty]
			mword            ports.StorePort[Empty]
			mwordGroup       ports.StorePort[Empty]
		}{
			spam:             storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int { return 15 }),
			emote:            storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int { return 15 }),
			exceptionsSpam:   storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int { return 15 }),
			exceptionsEmotes: storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int { return 15 }),
			mword:            storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int { return 15 }),
			mwordGroup:       storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int { return 15 }),
		}}
	text := "Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, хач and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum."
	words := strings.Fields(text)
	username := "benchmarkUser"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.CheckMwords(text, username, words)
	}
}
