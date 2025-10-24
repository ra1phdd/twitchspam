package app

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
	"twitchspam/internal/app/adapters/file_server"
	router "twitchspam/internal/app/adapters/http"
	"twitchspam/internal/app/adapters/message"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/adapters/platform/twitch"
	"twitchspam/internal/app/domain/stream"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

const configPath = "config.json"

func New() error {
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: http.DefaultTransport,
	}
	log := logger.New()

	manager, err := config.New(configPath)
	if err != nil {
		log.Fatal("Error loading config", err)
	}

	cfg := manager.Get()
	if cfg.Proxy != nil && cfg.Proxy.Address != "" && cfg.Proxy.Port != 0 {
		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", cfg.Proxy.Address, cfg.Proxy.Port), nil, proxy.Direct)
		if err != nil {
			return err
		}

		client.Transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		}
	}

	fs := file_server.New(log, client)
	log.SetLogLevel(cfg.App.LogLevel)
	gin.SetMode(cfg.App.GinMode)

	prometheus.MustRegister(metrics.MessageProcessingTime)

	metrics.BotEnabled.Set(map[bool]float64{true: 1, false: 0}[cfg.Enabled])
	metrics.AntiSpamEnabled.With(prometheus.Labels{"type": "default"}).Set(map[bool]float64{true: 1, false: 0}[cfg.Spam.SettingsDefault.Enabled])
	metrics.AntiSpamEnabled.With(prometheus.Labels{"type": "vip"}).Set(map[bool]float64{true: 1, false: 0}[cfg.Spam.SettingsVIP.Enabled])
	metrics.AntiSpamEnabled.With(prometheus.Labels{"type": "emote"}).Set(map[bool]float64{true: 1, false: 0}[cfg.Spam.SettingsEmotes.Enabled])

	if _, err := os.Stat("cache"); os.IsNotExist(err) {
		if err := os.Mkdir("cache", 0700); err != nil {
			log.Error("Error creating cache directory", err)
			return err
		}
	} else if err != nil {
		log.Error("Error stat cache directory", err)
		return err
	}

	t := twitch.New(log, manager, client)
	cacheStats := storage.NewCache[stream.SessionStats](0, 0, true, true, "cache/stats.json", 0)

	streams := make(map[string]ports.StreamPort, len(cfg.App.ModChannels))
	channelIDs := make([]string, 0, len(cfg.App.ModChannels))

	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, channel := range cfg.App.ModChannels {
		wg.Add(1)
		go func() {
			defer wg.Done()

			prefixedLog := logger.NewPrefixedLogger(log, channel)
			st := stream.NewStream(channel, fs, cacheStats)

			id, err := t.API().GetChannelID(channel)
			if err != nil {
				log.Error("Error getting live stream", err)
				return
			}
			st.SetChannelID(id)
			channelIDs = append(channelIDs, id)

			msg := message.New(prefixedLog, manager, st, t.API(), client)
			if err := t.AddChannel(channel, st, msg); err != nil {
				log.Info(fmt.Sprintf("[%s] Failed add channel", channel))
				return
			}

			mu.Lock()
			streams[channel] = st
			mu.Unlock()

			metrics.MessagesPerStream.With(prometheus.Labels{"channel": channel}).Add(0)
			metrics.ModerationActions.With(prometheus.Labels{"channel": channel, "action": "delete"}).Set(0)
			metrics.ModerationActions.With(prometheus.Labels{"channel": channel, "action": "timeout"}).Set(0)
			metrics.ModerationActions.With(prometheus.Labels{"channel": channel, "action": "ban"}).Set(0)
			log.Info(fmt.Sprintf("[%s] Chatbot started", channel))
		}()
	}

	wg.Wait()

	go func() {
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
					log.Error("Error getting live stream", nil)
					continue
				}

				livedStreams[d.UserLogin] = struct{}{}

				s.SetIslive(true)
				s.Stats().SetOnline(d.ViewerCount)
				s.OnceStart().Do(func() {
					s.Stats().SetStartTime(d.StartedAt.In(time.Local))
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

	r, err := router.NewRouter(log, manager, client)
	if err != nil {
		return err
	}
	return r.Run()
}
