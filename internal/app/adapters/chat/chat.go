package chat

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"twitchspam/config"
	"twitchspam/internal/app/adapters/event_sub"
	"twitchspam/internal/app/adapters/moderation"
	"twitchspam/internal/app/adapters/stats"
	"twitchspam/internal/app/domain/antispam"
	"twitchspam/internal/app/domain/messages/admin"
	"twitchspam/internal/app/domain/messages/user"
	"twitchspam/internal/app/domain/stream"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Chat struct {
	log        logger.Logger
	cfg        *config.Config
	stream     ports.StreamPort
	automod    ports.AutomodPort
	moderation ports.ModerationPort
	checker    ports.CheckerPort
	admin      ports.AdminPort
	user       ports.UserPort
	stats      ports.StatsPort

	isListen sync.Once
	conn     net.Conn
	reader   *bufio.Reader
}

func New(log logger.Logger, manager *config.Manager, client *http.Client, modChannel string) (*Chat, error) {
	c := &Chat{
		log: log,
		cfg: manager.Get(),
	}

	channelID, err := GetChannelID(modChannel, c.cfg.App.OAuth, c.cfg.App.ClientID)
	if err != nil {
		return nil, err
	}

	viewerCount, isLive, err := GetOnline(modChannel, c.cfg.App.OAuth, c.cfg.App.ClientID)
	if err != nil {
		return nil, err
	}

	c.stream = stream.NewStream(channelID, modChannel)
	c.stream.SetIslive(isLive)

	c.stats = stats.New(log)
	if isLive {
		c.log.Info("Stream started")
		c.stream.SetIslive(true)
		c.stats.SetStartTime(time.Now())
		c.stats.SetOnline(viewerCount)
	}

	c.automod = event_sub.NewEventSub(log, c.cfg, c.stream, c.stats, client)
	c.moderation = moderation.New(log, c.cfg, c.stream, client)
	c.checker = antispam.NewCheck(log, c.cfg, c.stream, c.stats)
	c.admin = admin.New(log, manager, c.stream)
	c.user = user.New(log, manager, c.stream, c.stats)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			viewerCount, _, err := GetOnline(modChannel, c.cfg.App.OAuth, c.cfg.App.ClientID)
			if err != nil {
				log.Error("Error getting viewer count", err)
				return
			}
			c.stats.SetOnline(viewerCount)
		}
	}()
	go c.automod.RunEventLoop()

	return c, nil
}

func (c *Chat) Connect(modChannel string) error {
	conn, err := tls.Dial("tcp", "irc.chat.twitch.tv:443", &tls.Config{MinVersion: tls.VersionTLS12})
	if err != nil {
		c.log.Error("Failed to connect to IRC chat Twitch", err)
		return err
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)

	c.write(fmt.Sprintf("PASS oauth:%s", c.cfg.App.OAuth))
	c.write(fmt.Sprintf("NICK %s", c.cfg.App.Username))
	c.write("CAP REQ :twitch.tv/tags")
	c.write("CAP REQ :twitch.tv/membership")
	c.write("CAP REQ :twitch.tv/commands")

	c.Join(modChannel)
	c.isListen.Do(func() {
		go c.listen()
	})

	return nil
}

func (c *Chat) listen() {
	c.log.Info("Listening on IRC chat Twitch")

	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			c.log.Error("Failed to read line on Twitch", err)
			return
		}
		line = strings.TrimSpace(line)

		// keep-alive
		if strings.HasPrefix(line, "PING") {
			c.write("PONG :tmi.twitch.tv")
			continue
		}

		switch {
		case strings.Contains(line, "Login authentication failed"):
			c.log.Fatal("Login authentication failed", nil, slog.String("line", line))
		case strings.Contains(line, "Improperly formatted auth"):
			c.log.Fatal("Improperly formatted auth", nil, slog.String("line", line))
		case strings.Contains(line, "Your message was not sent because you are sending messages too quickly"):
			c.log.Error("Rate limit exceeded", nil, slog.String("line", line))
		case strings.Contains(line, "PRIVMSG"):
			irc := ParseIRC(line)
			c.log.Debug("New message", slog.String("username", irc.Username), slog.String("text", irc.Text))

			if adminAction := c.admin.FindMessages(irc); adminAction != admin.None {
				c.Say(fmt.Sprintf("@%s, %s!", irc.Username, adminAction), c.stream.ChannelName())
				continue
			}

			if userAction := c.user.FindMessages(irc); userAction != user.None {
				c.Say(fmt.Sprintf("@%s, %s", irc.Username, userAction), c.stream.ChannelName())
				continue
			}

			action := c.checker.Check(irc)
			switch action.Type {
			case antispam.Ban:
				c.log.Warn("Banword in phrase", slog.String("username", action.Username), slog.String("text", action.Text))
				c.moderation.Ban(action.UserID, action.Reason)
			case antispam.Timeout:
				c.log.Warn("Spam is found", slog.String("username", action.Username), slog.String("text", action.Text), slog.Int("duration", int(action.Duration.Seconds())))
				if c.cfg.Spam.SettingsDefault.Enabled {
					c.moderation.Timeout(action.UserID, int(action.Duration.Seconds()), action.Reason)
				}
			}
		case strings.Contains(line, "CLEARCHAT"):
			irc := ParseIRC(line)
			c.log.Debug("User muted", slog.String("moderator", irc.Channel))
		case strings.Contains(line, "USERNOTICE"):
			c.log.Debug("New sub", slog.String("line", line))
		case strings.Contains(line, "JOIN"):
			c.log.Debug("New chatter", slog.String("line", line))
		case strings.Contains(line, "PART"):
			c.log.Debug("Exit chatter", slog.String("line", line))
		}
	}
}

func (c *Chat) Join(channel string) {
	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel
	}
	c.write("JOIN " + channel)
}

func (c *Chat) Part(channel string) {
	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel
	}
	c.write("PART " + channel)
}

func (c *Chat) Say(message, channel string) {
	if len(message) >= 500 {
		c.log.Error("Message too long", nil, slog.String("message", message))
		return
	}
	c.write(fmt.Sprintf("PRIVMSG #%s :%s", channel, message))
}

func (c *Chat) write(msg string) {
	if c.conn != nil {
		_, _ = c.conn.Write([]byte(msg + "\r\n"))
	}
}
