package twitch

import (
	"net/http"
	"time"
	"twitchspam/internal/app/adapters/messages/admin"
	"twitchspam/internal/app/adapters/messages/checker"
	"twitchspam/internal/app/adapters/messages/user"
	"twitchspam/internal/app/adapters/twitch/api"
	"twitchspam/internal/app/adapters/twitch/event_sub"
	"twitchspam/internal/app/adapters/twitch/irc"
	"twitchspam/internal/app/domain/aliases"
	"twitchspam/internal/app/domain/banwords"
	"twitchspam/internal/app/domain/regex"
	"twitchspam/internal/app/domain/stats"
	"twitchspam/internal/app/domain/stream"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Twitch struct {
	log     logger.Logger
	cfg     *config.Config
	stream  ports.StreamPort
	api     ports.APIPort
	checker ports.CheckerPort
	admin   ports.AdminPort
	user    ports.UserPort
	aliases ports.AliasesPort
	bwords  ports.BanwordsPort
	stats   ports.StatsPort
	irc     ports.IRCPort

	client *http.Client
}

func New(log logger.Logger, manager *config.Manager, client *http.Client, modChannel string) (*Twitch, error) {
	t := &Twitch{
		log:    log,
		cfg:    manager.Get(),
		stats:  stats.New(),
		client: client,
	}

	t.stream = stream.NewStream(modChannel)
	t.api = api.NewTwitch(t.log, t.cfg, t.stream, t.client)

	channelID, err := t.api.GetChannelID(modChannel)
	if err != nil {
		return nil, err
	}
	t.stream.SetChannelID(channelID)

	live, err := t.api.GetLiveStream()
	if err != nil {
		return nil, err
	}
	t.stream.SetIslive(live.IsOnline)

	if live.IsOnline {
		t.log.Info("Stream started")
		t.stream.SetIslive(true)
		t.stream.SetStreamID(live.ID)

		t.stats.SetStartTime(live.StartedAt)
		t.stats.SetOnline(live.ViewerCount)
	}

	t.irc, err = irc.New(t.log, t.cfg, 1*time.Second, modChannel)
	if err != nil {
		return nil, err
	}

	r := regex.New()
	t.aliases = aliases.New(t.cfg.Aliases, t.stream)
	t.bwords = banwords.New(t.cfg.Banwords.Words, t.cfg.Banwords.Regexp)
	t.checker = checker.NewCheck(log, t.cfg, t.stream, t.stats, t.bwords, r, t.irc)
	t.admin = admin.New(log, manager, t.stream, r, t.api, t.aliases)
	t.user = user.New(log, t.cfg, t.stream, t.stats, r, t.aliases)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			live, err := t.api.GetLiveStream()
			if err != nil {
				log.Error("Error getting viewer count", err)
				return
			}

			if live.IsOnline {
				t.stream.SetIslive(true)
				t.stream.SetStreamID(live.ID)
				t.stats.SetOnline(live.ViewerCount)
			}
		}
	}()

	es := event_sub.New(t.log, t.cfg, t.stream, t.api, t.checker, t.admin, t.user, t.aliases, t.bwords, t.stats, t.client)
	go es.RunEventLoop()

	return t, nil
}
