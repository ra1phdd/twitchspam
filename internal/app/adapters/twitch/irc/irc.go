package irc

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/pkg/logger"
)

type IRC struct {
	log logger.Logger
	cfg *config.Config

	mu    sync.Mutex
	chans map[string]chan bool
	ttl   time.Duration

	isListen sync.Once
	conn     net.Conn
	reader   *bufio.Reader
}

func New(log logger.Logger, cfg *config.Config, ttl time.Duration, modChannel string) (*IRC, error) {
	i := &IRC{
		log:   log,
		cfg:   cfg,
		chans: make(map[string]chan bool),
		ttl:   ttl,
	}
	go i.cleanupLoop()

	conn, err := tls.Dial("tcp", "irc.chat.twitch.tv:443", &tls.Config{MinVersion: tls.VersionTLS12})
	if err != nil {
		i.log.Error("Failed to connect to IRC chat Twitch", err)
		return nil, err
	}

	i.conn = conn
	i.reader = bufio.NewReader(conn)

	i.write(fmt.Sprintf("PASS oauth:%s", i.cfg.App.OAuth))
	i.write(fmt.Sprintf("NICK %s", i.cfg.App.Username))
	i.write("CAP REQ :twitch.tv/tags")
	i.write("CAP REQ :twitch.tv/membership")
	i.write("CAP REQ :twitch.tv/commands")

	i.join(modChannel)
	i.isListen.Do(func() {
		go i.listen()
	})

	return i, nil
}

func (i *IRC) listen() {
	i.log.Info("Listening on IRC chat Twitch")

	for {
		line, err := i.reader.ReadString('\n')
		if err != nil {
			i.log.Error("Failed to read line on Twitch", err)
			return
		}
		line = strings.TrimSpace(line)

		// keep-alive
		if strings.HasPrefix(line, "PING") {
			i.write("PONG :tmi.twitch.tv")
			continue
		}

		switch {
		case strings.Contains(line, "Login authentication failed"):
			i.log.Fatal("Login authentication to IRC failed", nil, slog.String("line", line))
		case strings.Contains(line, "Improperly formatted auth"):
			i.log.Fatal("Improperly formatted auth to IRC", nil, slog.String("line", line))
		case strings.Contains(line, "Your message was not sent because you are sending messages too quickly"):
			i.log.Error("Rate limit to IRC exceeded", nil, slog.String("line", line))
		case strings.Contains(line, "PRIVMSG"):
			irc := i.parseMessage(line)
			if irc != nil && irc.MessageID != "" {
				i.log.Debug("New IRC meta", slog.String("id", irc.MessageID), slog.Bool("isFirst", irc.IsFirst))
				i.NotifyIRC(irc.MessageID, irc.IsFirst)
			}
		case strings.Contains(line, "USERNOTICE"):
			i.log.Debug("New sub", slog.String("line", line))
		case strings.Contains(line, "JOIN"):
			i.log.Debug("New chatter", slog.String("line", line))
		case strings.Contains(line, "PART"):
			i.log.Debug("Exit chatter", slog.String("line", line))
		}
	}
}

func (i *IRC) join(channel string) {
	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel

	}
	i.write("JOIN " + channel)
}

func (i *IRC) part(channel string) {
	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel
	}
	i.write("PART " + channel)
}

func (i *IRC) write(msg string) {
	if i.conn != nil {
		_, _ = i.conn.Write([]byte(msg + "\r\n"))
	}
}
