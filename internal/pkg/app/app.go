package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
	"twitchspam/internal/app/adapters/file_server"
	router "twitchspam/internal/app/adapters/http"
	"twitchspam/internal/app/adapters/message"
	"twitchspam/internal/app/adapters/platform/twitch"
	"twitchspam/internal/app/domain/stream"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
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
	gin.SetMode(cfg.App.GinMode)

	streams := make(map[string]ports.StreamPort, len(cfg.App.ModChannels))
	t := twitch.New(log, manager, client)

	for _, channel := range cfg.App.ModChannels {
		go func() {
			prefixedLog := logger.NewPrefixedLogger(log, channel)
			streams[channel] = stream.NewStream(channel, fs)
			msg := message.New(prefixedLog, manager, streams[channel], t.API(), client)

			if err := t.AddChannel(channel, streams[channel], msg); err != nil {
				log.Info(fmt.Sprintf("[%s] Failed add channel", channel))
				return
			}

			log.Info(fmt.Sprintf("[%s] Chatbot started", channel))
		}()
	}

	go func() {
		var channelIDs []string
		for _, channel := range cfg.App.ModChannels {
			id, err := t.API().GetChannelID(channel)
			if err != nil {
				log.Error("Error getting live stream", err)
				return
			}

			channelIDs = append(channelIDs, id)
		}

		syncLiveStreams := func() {
			data, err := t.API().GetLiveStreams(channelIDs)
			if err != nil {
				log.Error("Error getting live stream", err)
				return
			}

			livedStreams := make(map[string]struct{}, len(data))
			for _, d := range data {
				s, ok := streams[d.UserLogin]
				if !ok {
					fmt.Println("Error getting live stream")
					continue
				}

				livedStreams[d.UserLogin] = struct{}{}

				s.SetIslive(true)
				s.SetChannelID(d.UserID)
				s.Stats().SetStartTime(d.StartedAt)
				s.Stats().SetOnline(d.ViewerCount)
			}

			for _, channel := range cfg.App.ModChannels {
				if _, ok := livedStreams[channel]; ok {
					continue
				}

				s, ok := streams[channel]
				if !ok {
					continue
				}
				s.SetIslive(false)
			}
		}

		syncLiveStreams()
		for range time.Tick(30 * time.Second) {
			syncLiveStreams()
		}
	}()

	r := router.NewRouter(log, manager)
	return r.Run()
}
