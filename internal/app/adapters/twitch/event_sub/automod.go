package event_sub

import (
	"log/slog"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/messages/checker"
	"twitchspam/internal/app/domain"
)

func (es *EventSub) checkAutomod(am AutomodHoldEvent) {
	if !es.cfg.Enabled {
		return
	}
	text := strings.ToLower(domain.NormalizeText(am.Message.Text))

	if action := es.checker.CheckBanwords(text, am.Message.Text); action != nil {
		time.Sleep(time.Duration(es.cfg.Spam.DelayAutomod) * time.Second)
		es.api.BanUser(am.UserID, action.Reason)
	}

	if action := es.checker.CheckAds(text, am.UserName); action != nil {
		es.api.BanUser(am.UserID, action.Reason)
	}

	action := es.checker.CheckMwords(text)
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
