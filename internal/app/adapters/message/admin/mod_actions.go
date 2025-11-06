package admin

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"twitchspam/internal/app/adapters/platform/twitch/api"
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type BanUser struct {
	re     *regexp.Regexp
	log    logger.Logger
	stream ports.StreamPort
	api    ports.APIPort
}

func (b *BanUser) Execute(_ *config.Config, _ string, msg *message.ChatMessage) *ports.AnswerType {
	matches := b.re.FindStringSubmatch(msg.Message.Text.Text()) // !am ban <username> <причина?>
	if len(matches) != 3 {
		return nonParametr
	}

	username := strings.TrimSpace(strings.TrimPrefix(matches[1], "@"))
	reason := strings.TrimSpace(matches[2])

	ids, err := b.api.GetChannelIDs([]string{username})
	if err != nil || len(ids) == 0 {
		return unknownUser
	}
	b.log.Info("Ban command received", slog.String("username", username), slog.String("reason", reason))

	b.api.BanUser(b.stream.ChannelName(), b.stream.ChannelID(), ids[strings.ToLower(username)], reason)
	return &ports.AnswerType{Text: []string{fmt.Sprintf("пользователь %s забанен!", username)}, IsReply: true}
}

type UnbanUser struct {
	re     *regexp.Regexp
	log    logger.Logger
	stream ports.StreamPort
	api    ports.APIPort
}

func (u *UnbanUser) Execute(_ *config.Config, _ string, msg *message.ChatMessage) *ports.AnswerType {
	matches := u.re.FindStringSubmatch(msg.Message.Text.Text()) // !am unban/untimeout <username>
	if len(matches) != 2 {
		return nonParametr
	}

	username := strings.TrimSpace(strings.TrimPrefix(matches[1], "@"))

	ids, err := u.api.GetChannelIDs([]string{username})
	if err != nil || len(ids) == 0 {
		return unknownUser
	}
	u.log.Info("Unban command received", slog.String("username", username))

	u.api.UnbanUser(u.stream.ChannelID(), ids[strings.ToLower(username)])
	return &ports.AnswerType{Text: []string{fmt.Sprintf("ограничения с пользователя %s сняты!", username)}, IsReply: true}
}

type WarnUser struct {
	re     *regexp.Regexp
	log    logger.Logger
	stream ports.StreamPort
	api    ports.APIPort
}

func (w *WarnUser) Execute(_ *config.Config, _ string, msg *message.ChatMessage) *ports.AnswerType {
	matches := w.re.FindStringSubmatch(msg.Message.Text.Text()) // !am warn <username> <причина>
	if len(matches) != 3 {
		return nonParametr
	}

	username := strings.TrimSpace(strings.TrimPrefix(matches[1], "@"))
	reason := strings.TrimSpace(matches[2])

	ids, err := w.api.GetChannelIDs([]string{username})
	if err != nil || len(ids) == 0 {
		return unknownUser
	}
	w.log.Info("Warn command received", slog.String("username", username), slog.String("reason", reason))

	if err := w.api.WarnUser(w.stream.ChannelName(), w.stream.ChannelID(), ids[strings.ToLower(username)], reason); err != nil {
		w.log.Error("Failed to warn user", err)
		if errors.Is(err, api.ErrUserAuthNotCompleted) {
			return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
		}
		return unknownError
	}

	return &ports.AnswerType{Text: []string{fmt.Sprintf("пользователь %s предупреждён!", username)}, IsReply: true}
}

type TimeoutUser struct {
	re     *regexp.Regexp
	log    logger.Logger
	stream ports.StreamPort
	api    ports.APIPort
}

func (t *TimeoutUser) Execute(_ *config.Config, _ string, msg *message.ChatMessage) *ports.AnswerType {
	matches := t.re.FindStringSubmatch(msg.Message.Text.Text()) // !am timeout <username> <длительность?> <причина?>
	if len(matches) != 4 {
		return nonParametr
	}

	username := strings.TrimSpace(strings.TrimPrefix(matches[1], "@"))
	duration := 600
	if matches[2] != "" {
		if d, err := strconv.Atoi(matches[2]); err == nil {
			duration = d
		}
	}
	reason := strings.TrimSpace(matches[3])

	ids, err := t.api.GetChannelIDs([]string{username})
	if err != nil || len(ids) == 0 {
		return unknownUser
	}
	t.log.Info("Timeout command received", slog.String("username", username), slog.Int("duration", duration), slog.String("reason", reason))

	t.api.TimeoutUser(t.stream.ChannelName(), t.stream.ChannelID(), ids[strings.ToLower(username)], duration, reason)
	return &ports.AnswerType{Text: []string{fmt.Sprintf("пользователь %s отправлен в таймаут на %d сек!", username, duration)}, IsReply: true}
}
