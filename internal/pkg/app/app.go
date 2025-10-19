package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"time"
	"twitchspam/internal/app/adapters/file_server"
	router "twitchspam/internal/app/adapters/http"
	"twitchspam/internal/app/adapters/message"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/adapters/platform/twitch"
	"twitchspam/internal/app/domain/stream"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

func New() error {
	log := logger.New()
	client := &http.Client{Timeout: 10 * time.Second}
	fs := file_server.New(log, client)

	manager, err := config.New("config.json")
	if err != nil {
		log.Fatal("Error loading config", err)
	}

	cfg := manager.Get()
	log.SetLogLevel(cfg.App.LogLevel)
	gin.SetMode(cfg.App.GinMode)

	metrics.BotEnabled.Set(map[bool]float64{true: 1, false: 0}[cfg.Enabled])
	metrics.AntiSpamEnabled.With(prometheus.Labels{"type": "default"}).Set(map[bool]float64{true: 1, false: 0}[cfg.Spam.SettingsDefault.Enabled])
	metrics.AntiSpamEnabled.With(prometheus.Labels{"type": "vip"}).Set(map[bool]float64{true: 1, false: 0}[cfg.Spam.SettingsVIP.Enabled])
	metrics.AntiSpamEnabled.With(prometheus.Labels{"type": "emote"}).Set(map[bool]float64{true: 1, false: 0}[cfg.Spam.SettingsEmotes.Enabled])

	streams := make(map[string]ports.StreamPort, len(cfg.App.ModChannels))
	t := twitch.New(log, manager, client)

	for _, channel := range cfg.App.ModChannels {
		prefixedLog := logger.NewPrefixedLogger(log, channel)
		streams[channel] = stream.NewStream(channel, fs)
		msg := message.New(prefixedLog, manager, streams[channel], t.API(), client)

		if err := t.AddChannel(channel, streams[channel], msg); err != nil {
			log.Info(fmt.Sprintf("[%s] Failed add channel", channel))
			continue
		}

		log.Info(fmt.Sprintf("[%s] Chatbot started", channel))
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
				s.Stats().SetOnline(d.ViewerCount)
				s.OnceStart().Do(func() {
					s.Stats().SetStartTime(d.StartedAt)
				})
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

	r, err := router.NewRouter(log, manager)
	if err != nil {
		return err
	}
	return r.Run()
}
