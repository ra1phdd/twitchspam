package chat

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"golang.org/x/time/rate"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"twitchspam/config"
	"twitchspam/internal/app/adapters/event_sub"
	"twitchspam/internal/app/adapters/moderation"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/domain/admin"
	"twitchspam/internal/app/domain/antispam"
	"twitchspam/internal/app/domain/stream"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Chat struct {
	log        logger.Logger
	cfg        *config.Config
	automod    ports.AutomodPort
	moderation ports.ModerationPort
	checker    ports.CheckerPort
	admin      ports.AdminPort
	stream     ports.StreamPort

	isListen sync.Once
	conn     net.Conn
	reader   *bufio.Reader
}

func New(log logger.Logger, manager *config.Manager, client *http.Client, modChannel string) (*Chat, error) {
	cfg := manager.Get()

	channelID, err := GetChannelID(modChannel, cfg.App.OAuth, cfg.App.ClientID)
	if err != nil {
		return nil, err
	}

	isLive, err := IsLive(modChannel, cfg.App.OAuth, cfg.App.ClientID)
	if err != nil {
		return nil, err
	}

	st := stream.NewStream(channelID, modChannel)
	st.SetIslive(isLive)

	c := &Chat{
		log:        log,
		cfg:        cfg,
		stream:     st,
		automod:    event_sub.NewEventSub(log, cfg, st, client),
		moderation: moderation.New(log, cfg, st, client),
		checker:    antispam.NewCheck(log, cfg, st),
		admin:      admin.New(log, manager),
	}

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

	limiter := rate.NewLimiter(rate.Every(30*time.Second), 1)
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

			adminAction := c.admin.FindMessages(irc)
			if adminAction != admin.None {
				c.Say(fmt.Sprintf("@%s, %s!", irc.Username, adminAction), c.stream.ChannelName())
				continue
			}

			text := strings.ToLower(domain.NormalizeText(irc.Text))
			if (strings.Contains(text, "че за игра") || strings.Contains(text, "чё за игра") || strings.Contains(text, "что за игра") ||
				strings.Contains(text, "как игра называется") || strings.Contains(text, "как игра называеться")) && limiter.Allow() {
				c.Say(fmt.Sprintf("@%s, !g", irc.Username), c.stream.ChannelName())
				continue
			}

			if strings.Contains(text, "афсигга плохой бот") || strings.Contains(text, "афсига плохой бот") ||
				strings.Contains(text, "афсугга плохой бот") || strings.Contains(text, "афсуга плохой бот") {
				c.Say(fmt.Sprintf("@%s, ((", irc.Username), c.stream.ChannelName())
			}

			if strings.Contains(text, "афсигга хороший бот") || strings.Contains(text, "афсига хороший бот") ||
				strings.Contains(text, "афсугга хороший бот") || strings.Contains(text, "афсуга хороший бот") {
				c.Say(fmt.Sprintf("@%s, nya", irc.Username), c.stream.ChannelName())
			}

			action := c.checker.Check(irc)
			switch action.Type {
			case antispam.Ban:
				c.log.Warn("Banword in phrase", slog.String("username", action.Username), slog.String("text", action.Text))
				c.moderation.Ban(action.UserID, action.Reason)
			case antispam.Timeout:
				c.log.Warn("Spam is found", slog.String("username", action.Username), slog.String("text", action.Text), slog.Int("duration", int(action.Duration.Seconds())))
				if c.cfg.Spam.Enabled {
					c.moderation.Timeout(action.UserID, int(action.Duration.Seconds()), action.Reason)
				}
			}
		case strings.Contains(line, "CLEARCHAT"):
			c.log.Debug("User muted", slog.String("line", line))
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
