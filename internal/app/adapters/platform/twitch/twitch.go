package twitch

import (
	"net/http"
	"time"
	"twitchspam/internal/app/adapters/platform/twitch/api"
	"twitchspam/internal/app/adapters/platform/twitch/event_sub"
	"twitchspam/internal/app/adapters/platform/twitch/irc"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Twitch struct {
	log    logger.Logger
	cfg    *config.Config
	client *http.Client

	api      ports.APIPort
	irc      ports.IRCPort
	eventSub *event_sub.EventSub
}

func New(log logger.Logger, manager *config.Manager, client *http.Client) *Twitch {
	cfg := manager.Get()

	t := &Twitch{
		log:    log,
		cfg:    cfg,
		client: client,
	}
	t.api = api.NewTwitch(log, cfg, client)
	t.irc = irc.New(log, cfg, 1*time.Second)
	t.eventSub = event_sub.NewTwitch(t.log, t.cfg, t.api, t.irc, t.client)

	return t
}

func (t *Twitch) API() ports.APIPort {
	return t.api
}

func (t *Twitch) IRC() ports.IRCPort {
	return t.irc
}

func (t *Twitch) AddChannel(channel string, stream ports.StreamPort, message ports.MessagePort) error {
	t.irc.AddChannel(channel)

	channelID, err := t.api.GetChannelID(channel)
	if err != nil {
		return err
	}
	stream.SetChannelID(channelID)

	go func() {
		live, err := t.api.GetLiveStream(channelID)
		if err != nil {
			t.log.Error("Error getting live stream", err)
			return
		}
		stream.SetIslive(live.IsOnline)

		if live.IsOnline {
			t.log.Info("Stream started")
			stream.SetIslive(true)
			stream.SetStreamID(live.ID)

			stream.Stats().SetStartTime(live.StartedAt)
			stream.Stats().SetOnline(live.ViewerCount)
		}

		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			live, err := t.api.GetLiveStream(channelID)
			if err != nil {
				t.log.Error("Error getting viewer count", err)
				return
			}

			if live.IsOnline {
				stream.SetIslive(true)
				stream.SetStreamID(live.ID)
				stream.Stats().SetOnline(live.ViewerCount)
				stream.Stats().SetEndTime(time.Now())
			}
		}
	}()

	t.eventSub.AddChannel(channelID, channel, stream, message)
	return nil
}

func (t *Twitch) RemoveChannel(channel string) {
	t.irc.RemoveChannel(channel)
}
