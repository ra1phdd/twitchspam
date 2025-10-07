package event_sub

import (
	"log/slog"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/messages/checker"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/storage"
)

func (es *EventSub) checkMessage(msgEvent ChatMessageEvent) {
	msg := es.convertMap(msgEvent)
	if es.stream.IsLive() {
		es.stream.Stats().AddMessage(msg.Chatter.Username)
	}

	es.template.Store().Messages().Push(msg.Chatter.Username, msg.Message.ID, storage.Message{
		Data:               msg,
		Time:               time.Now(),
		HashWordsLowerNorm: domain.WordsToHashes(msg.Message.Text.Words(domain.Lower, domain.RemovePunctuation, domain.RemoveDuplicateLetters)),
		IgnoreAntispam:     !es.cfg.Enabled || !es.template.SpamPause().CanProcess() || !es.cfg.Spam.SettingsDefault.Enabled,
	})

	if !strings.HasPrefix(msg.Message.Text.Text(), "!am al ") && !strings.HasPrefix(msg.Message.Text.Text(), "!am alg ") {
		text, ok := es.template.Aliases().Replace(msg.Message.Text.Words(domain.RemovePunctuation))
		if ok {
			msg.Message.Text.ReplaceOriginal(text)
		}
	}

	if adminAction := es.admin.FindMessages(msg); adminAction != nil {
		adminAction.ReplyUsername = msg.Chatter.Username
		es.api.SendChatMessages(adminAction)
		return
	}

	if userAction := es.user.FindMessages(msg); userAction != nil {
		if userAction.IsReply && userAction.ReplyUsername == "" {
			userAction.ReplyUsername = msg.Chatter.Username
		}
		es.api.SendChatMessages(userAction)
		return
	}

	action := es.checker.Check(msg)
	switch action.Type {
	case checker.Ban:
		es.log.Warn("Ban user", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text.Text()))
		es.api.BanUser(msg.Chatter.UserID, action.Reason)
	case checker.Timeout:
		es.log.Warn("Timeout user", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text.Text()), slog.Int("duration", int(action.Duration.Seconds())))
		if es.cfg.Spam.SettingsDefault.Enabled {
			es.api.TimeoutUser(msg.Chatter.UserID, int(action.Duration.Seconds()), action.Reason)
		}
	case checker.Delete:
		es.log.Warn("Delete message", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text.Text()))
		if err := es.api.DeleteChatMessage(msg.Message.ID); err != nil {
			es.log.Error("Failed to delete message on chat", err)
		}
	}
}
