package event_sub

import (
	"log/slog"
	"time"
	"twitchspam/internal/app/adapters/messages/checker"
	"twitchspam/internal/app/ports"
)

func (es *EventSub) checkAutomod(am AutomodHoldEvent) {
	if !es.cfg.Enabled || !es.cfg.Automod.Enabled {
		return
	}

	if es.cfg.Automod.Delay > 0 {
		time.Sleep(time.Duration(es.cfg.Automod.Delay) * time.Second)
	}

	msg := &ports.ChatMessage{
		Message: ports.Message{
			ID: am.MessageID,
			Text: ports.MessageText{
				Original: am.Message.Text,
			},
		},
	}

	if action := es.checker.CheckBanwords(msg.Message.Text.LowerNorm(), msg.Message.Text.Words()); action != nil {
		es.api.BanUser(am.UserID, action.Reason)
	}

	if action := es.checker.CheckAds(msg.Message.Text.Lower(), am.UserName); action != nil {
		es.api.BanUser(am.UserID, action.Reason)
	}

	action := es.checker.CheckMwords(msg)
	if action == nil {
		return
	}

	switch action.Type {
	case checker.Ban:
		es.log.Warn("Banword in phrase", slog.String("username", am.UserName), slog.String("text", am.Message.Text))
		es.api.BanUser(am.UserID, action.Reason)
	case checker.Timeout:
		es.log.Warn("Spam is found", slog.String("username", am.UserName), slog.String("text", am.Message.Text), slog.Int("duration", int(action.Duration.Seconds())))
		if es.cfg.Spam.SettingsDefault.Enabled {
			es.api.TimeoutUser(am.UserID, int(action.Duration.Seconds()), action.Reason)
		}
	case checker.Delete:
		es.log.Warn("Muteword in phrase", slog.String("username", am.UserName), slog.String("text", am.Message.Text))
		if err := es.api.DeleteChatMessage(am.MessageID); err != nil {
			es.log.Error("Failed to delete message on chat", err)
		}
	}
}
