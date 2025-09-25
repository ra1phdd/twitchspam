package event_sub

import (
	"log/slog"
	"strings"
	"twitchspam/internal/app/adapters/messages/checker"
)

func (es *EventSub) checkMessage(msgEvent ChatMessageEvent) {
	msg := es.convertMap(msgEvent)
	if es.stream.IsLive() {
		es.stats.AddMessage(msg.Chatter.Username)
	}

	if !strings.HasPrefix(msg.Message.Text.Original, "!am alias") {
		text, ok := es.template.ReplaceAlias(msg.Message.Text.Words())
		if ok {
			msg.Message.Text.ReplaceOriginal(text)
		}
	}

	if adminAction := es.admin.FindMessages(msg); adminAction != nil {
		adminAction.ReplyUsername = msg.Chatter.Username
		es.api.SendChatMessages(adminAction)
		return
	}

	action := es.checker.Check(msg)
	switch action.Type {
	case checker.Ban:
		es.log.Warn("Banword in phrase", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text.Original))
		es.api.BanUser(msg.Chatter.UserID, action.Reason)
	case checker.Timeout:
		es.log.Warn("Spam is found", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text.Original), slog.Int("duration", int(action.Duration.Seconds())))
		if es.cfg.Spam.SettingsDefault.Enabled {
			es.api.TimeoutUser(msg.Chatter.UserID, int(action.Duration.Seconds()), action.Reason)
		}
	case checker.Delete:
		es.log.Warn("Muteword in phrase", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text.Original))
		if err := es.api.DeleteChatMessage(msg.Message.ID); err != nil {
			es.log.Error("Failed to delete message on chat", err)
		}
	}

	if userAction := es.user.FindMessages(msg); userAction != nil {
		if userAction.IsReply && userAction.ReplyUsername == "" {
			userAction.ReplyUsername = msg.Chatter.Username
		}

		es.api.SendChatMessages(userAction)
		return
	}
}
