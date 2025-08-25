package app

import (
	"fmt"
	"net/http"
	"time"
	"twitchspam/config"
	"twitchspam/internal/app/adapters/chat"
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

	var err error
	a.cfg, err = config.Load("config.json")
	if err != nil {
		a.log.Fatal("Error loading config", err)
	}
	a.log.SetLogLevel(a.cfg.App.LogLevel)

	for _, channel := range a.cfg.App.ModChannels {
		log := logger.NewPrefixedLogger(a.log, channel)

		c, err := chat.New(log, a.client, channel)
		if err != nil {
			return err
		}

		if err := c.Connect(channel); err != nil {
			return err
		}
		a.log.Info(fmt.Sprintf("[%s] Chatbot started", channel))
	}

	return nil
}
