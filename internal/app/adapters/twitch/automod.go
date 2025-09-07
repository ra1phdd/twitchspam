package twitch

import (
	"log/slog"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/messages/checker"
	"twitchspam/internal/app/domain"
)

func (t *Twitch) checkAutomod(am AutomodHoldEvent) {
	if !t.cfg.Enabled {
		return
	}
	text := strings.ToLower(domain.NormalizeText(am.Message.Text))

	if action := t.checker.CheckBanwords(text, am.Message.Text); action != nil {
		time.Sleep(time.Duration(t.cfg.Spam.DelayAutomod) * time.Second)
		t.moderation.Ban(am.UserID, action.Reason)
	}

	if action := t.checker.CheckAds(text, am.UserName); action != nil {
		t.moderation.Ban(am.UserID, action.Reason)
	}

	action := t.checker.CheckMwords(text)
	if action == nil {
		return
	}

	switch action.Type {
	case checker.Ban:
		t.log.Warn("Banword in phrase", slog.String("username", am.UserName), slog.String("text", am.Message.Text))
		t.moderation.Ban(am.UserID, action.Reason)
	case checker.Timeout:
		t.log.Warn("Spam is found", slog.String("username", am.UserName), slog.String("text", am.Message.Text), slog.Int("duration", int(action.Duration.Seconds())))
		if t.cfg.Spam.SettingsDefault.Enabled {
			t.moderation.Timeout(am.UserID, int(action.Duration.Seconds()), action.Reason)
		}
	case checker.Delete:
		t.log.Warn("Muteword in phrase", slog.String("username", am.UserName), slog.String("text", am.Message.Text))
		if err := t.DeleteChatMessage(am.BroadcasterUserID, am.MessageID); err != nil {
			t.log.Error("Failed to delete message on chat", err)
		}
	}
}
