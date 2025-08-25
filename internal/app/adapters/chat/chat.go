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
	"twitchspam/config"
	"twitchspam/internal/app/adapters/automod"
	"twitchspam/internal/app/adapters/moderation"
	"twitchspam/internal/app/domain/antispam"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Chat struct {
	log        logger.Logger
	automod    ports.AutomodPort
	moderation ports.ModerationPort
	checker    ports.CheckerPort

	isListen sync.Once
	conn     net.Conn
	reader   *bufio.Reader
}

func New(log logger.Logger, client *http.Client, modChannel string) (*Chat, error) {
	cfg := config.Get()
	channelID, err := GetChannelID(modChannel, cfg.App.OAuth, cfg.App.ClientID)
	if err != nil {
		return nil, err
	}

	c := &Chat{
		log:        log,
		automod:    automod.New(log, cfg.App.OAuth, cfg.App.ClientID, cfg.App.UserID, channelID, client),
		moderation: moderation.New(log, cfg.App.UserID, channelID, client),
		checker:    antispam.NewCheck(log, cfg),
	}

	go c.automod.RunEventLoop()
	return c, nil
}

func (c *Chat) Connect(modChannel string) error {
	conn, err := tls.Dial("tcp", "irc.chat.twitch.tv:443", &tls.Config{})
	if err != nil {
		c.log.Error("Failed to connect to IRC chat Twitch", err)
		return err
	}
	cfg := config.Get()

	c.conn = conn
	c.reader = bufio.NewReader(conn)

	c.write(fmt.Sprintf("PASS oauth:%s", cfg.App.OAuth))
	c.write(fmt.Sprintf("NICK %s", cfg.App.Username))
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
			c.log.Info("New message", slog.String("username", irc.Username), slog.String("text", irc.Text))

			action := c.checker.Check(irc, config.Get())
			switch action.Type {
			case antispam.Ban:
				c.log.Warn("Banword in phrase", slog.String("username", action.Username), slog.String("text", action.Text))
				//c.moderation.Ban(action.UserID, action.Reason)
			case antispam.Timeout:
				c.log.Warn("Spam is found", slog.String("username", action.Username), slog.String("text", action.Text), slog.Int("duration", int(action.Duration.Seconds())))
				//c.moderation.Timeout(action.UserID, int(action.Duration.Seconds()), action.Reason)
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
