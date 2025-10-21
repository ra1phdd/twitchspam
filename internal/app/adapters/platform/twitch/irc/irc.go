package irc

import (
	"bufio"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/pkg/logger"
)

type IRC struct {
	log logger.Logger
	cfg *config.Config

	mu       sync.Mutex
	channels map[string]bool
	chans    map[string]chan bool
	ttl      time.Duration

	client *http.Client
	conn   net.Conn
	reader *bufio.Reader
}

func New(log logger.Logger, cfg *config.Config, ttl time.Duration, client *http.Client) *IRC {
	i := &IRC{
		log:      log,
		cfg:      cfg,
		channels: make(map[string]bool),
		chans:    make(map[string]chan bool),
		ttl:      ttl,
		client:   client,
	}

	go i.runIRC()
	go i.cleanupLoop()

	return i
}

func (i *IRC) AddChannel(channel string) {
	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if i.channels[channel] {
		return
	}

	i.channels[channel] = true
	if i.conn != nil {
		i.join(channel)
	}
}

func (i *IRC) RemoveChannel(channel string) {
	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.channels[channel] {
		return
	}

	delete(i.channels, channel)
	if i.conn != nil {
		i.part(channel)
	}
}

func (i *IRC) runIRC() {
	for {
		err := i.connectAndListen()
		if err != nil {
			i.log.Warn("IRC connection lost, retrying...", slog.String("error", err.Error()))
			time.Sleep(5 * time.Second)
		}
	}
}

func (i *IRC) connectAndListen() error {
	conn, err := tls.Dial("tcp", "irc.chat.twitch.tv:443", &tls.Config{MinVersion: tls.VersionTLS12})
	if err != nil {
		i.log.Error("Failed to connect to IRC chat Twitch", err)
		return err
	}

	i.conn = conn
	i.reader = bufio.NewReader(i.conn)

	i.write("PASS oauth:" + i.cfg.App.OAuth)
	i.write("NICK " + i.cfg.App.Username)
	i.write("CAP REQ :twitch.tv/tags")
	i.write("CAP REQ :twitch.tv/membership")
	i.write("CAP REQ :twitch.tv/commands")

	i.mu.Lock()
	for ch := range i.channels {
		i.join(ch)
	}
	i.mu.Unlock()

	return i.listen()
}

func (i *IRC) listen() error {
	i.log.Info("Listening on IRC chat Twitch")

	for {
		line, err := i.reader.ReadString('\n')
		if err != nil {
			i.log.Error("Failed to read line on Twitch", err)
			return err
		}
		line = strings.TrimSpace(line)

		// keep-alive
		if strings.HasPrefix(line, "PING") {
			i.write("PONG :tmi.twitch.tv")
			continue
		}

		switch {
		case strings.Contains(line, "Login authentication failed"):
			i.log.Error("Login authentication to IRC failed", nil, slog.String("line", line))
		case strings.Contains(line, "Improperly formatted auth"):
			i.log.Error("Improperly formatted auth to IRC", nil, slog.String("line", line))
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
