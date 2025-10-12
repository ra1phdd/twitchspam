package app

import (
	"fmt"
	"net/http"
	"time"
	"twitchspam/internal/app/adapters/file_server"
	router "twitchspam/internal/app/adapters/http"
	"twitchspam/internal/app/adapters/message"
	"twitchspam/internal/app/adapters/platform/twitch"
	"twitchspam/internal/app/domain/stream"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/pkg/logger"
)

func New() error {
	log := logger.New()
	client := &http.Client{Timeout: 10 * time.Second}
	fs := file_server.New(client)

	manager, err := config.New("config.json")
	if err != nil {
		log.Fatal("Error loading config", err)
	}

	cfg := manager.Get()
	log.SetLogLevel(cfg.App.LogLevel)

	t := twitch.New(log, manager, client)

	for _, channel := range cfg.App.ModChannels {
		go func() {
			prefixedLog := logger.NewPrefixedLogger(log, channel)
			s := stream.NewStream(channel, fs)
			msg := message.New(prefixedLog, manager, s, t.API(), client)

			if err := t.AddChannel(channel, s, msg); err != nil {
				log.Info(fmt.Sprintf("[%s] Failed add channel", channel))
				return
			}

			log.Info(fmt.Sprintf("[%s] Chatbot started", channel))
		}()
	}

	r := router.NewRouter(log, manager)
	return r.Run()
}
