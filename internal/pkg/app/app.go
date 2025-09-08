package app

import (
	"fmt"
	"net/http"
	"time"
	"twitchspam/internal/app/adapters/twitch"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/pkg/logger"
)

type App struct {
	log    logger.Logger
	cfg    *config.Config
	client *http.Client
}

func New() error {
	a := &App{
		log:    logger.New(),
		client: &http.Client{Timeout: 10 * time.Second},
	}

	manager, err := config.New("config.json")
	if err != nil {
		a.log.Fatal("Error loading config", err)
	}

	a.cfg = manager.Get()
	a.log.SetLogLevel(a.cfg.App.LogLevel)

	for _, channel := range a.cfg.App.ModChannels {
		log := logger.NewPrefixedLogger(a.log, channel)

		_, err := twitch.New(log, manager, a.client, channel)
		if err != nil {
			return err
		}

		a.log.Info(fmt.Sprintf("[%s] Chatbot started", channel))
	}

	return nil
}
