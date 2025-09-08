package event_sub

import (
	"fmt"
	"log/slog"
	"twitchspam/internal/app/adapters/messages/checker"
)

func (es *EventSub) checkMessage(msgEvent ChatMessageEvent) {
	msg := es.convertMap(msgEvent)
	if es.stream.IsLive() {
		es.stats.AddMessage(msg.Chatter.Username)
	}

	sendMessages := func(messages []string, isReply bool, username string) {
		for _, message := range messages {
			text := message
			if isReply {
				text = fmt.Sprintf("@%s, %s", username, message)
			}

			if err := es.api.SendChatMessage(text); err != nil {
				es.log.Error("Failed to send message on chat", err)
			}
		}
	}

	msg.Message.Text = es.aliases.ReplaceOne(msg.Message.Text)
	if adminAction := es.admin.FindMessages(msg); adminAction != nil {
		sendMessages(adminAction.Text, adminAction.IsReply, msg.Chatter.Username)
		return
	}

	if userAction := es.user.FindMessages(msg); userAction != nil {
		sendMessages(userAction.Text, userAction.IsReply, msg.Chatter.Username)
		return
	}

	action := es.checker.Check(msg)
	switch action.Type {
	case checker.Ban:
		es.log.Warn("Banword in phrase", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text))
		es.api.BanUser(msg.Chatter.UserID, action.Reason)
	case checker.Timeout:
		es.log.Warn("Spam is found", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text), slog.Int("duration", int(action.Duration.Seconds())))
		if es.cfg.Spam.SettingsDefault.Enabled {
			es.api.TimeoutUser(msg.Chatter.UserID, int(action.Duration.Seconds()), action.Reason)
		}
	case checker.Delete:
		es.log.Warn("Muteword in phrase", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text))
		if err := es.api.DeleteChatMessage(msg.Message.ID); err != nil {
			es.log.Error("Failed to delete message on chat", err)
		}
	}
}
