package twitch

import (
	"fmt"
	"log/slog"
	"twitchspam/internal/app/adapters/messages/admin"
	"twitchspam/internal/app/adapters/messages/checker"
	"twitchspam/internal/app/adapters/messages/user"
)

func (t *Twitch) checkMessage(msgEvent ChatMessageEvent) {
	msg := t.convertMap(msgEvent)
	if t.stream.IsLive() {
		t.stats.AddMessage(msg.Chatter.Username)
	}

	if adminAction := t.admin.FindMessages(msg); adminAction != admin.None {
		if err := t.SendChatMessage(msg.Broadcaster.UserID, fmt.Sprintf("@%s, %s", msg.Chatter.Username, adminAction)); err != nil {
			t.log.Error("Failed to send message on chat", err)
		}
		return
	}

	if userAction := t.user.FindMessages(msg); userAction != user.None {
		if err := t.SendChatMessage(msg.Broadcaster.UserID, fmt.Sprintf("@%s, %s", msg.Chatter.Username, userAction)); err != nil {
			t.log.Error("Failed to send message on chat", err)
		}
		return
	}

	//if strings.HasPrefix(msg.Message.Text, "!vn") {
	//	id, err := t.GetUrlVOD(t.stream.StreamID())
	//	if err != nil {
	//		t.log.Error("Failed to get live message", err)
	//		return
	//	}
	//
	//	if err := t.SendChatMessage(msg.Broadcaster.UserID, fmt.Sprintf("@%s, %s - https://www.twitch.tv/videos/%s?t=%s",
	//		msg.Chatter.Username, time.Now().Format("02-01"), id, t.stats.GetStartTime().Round(time.Second).String())); err != nil {
	//		t.log.Error("Failed to send message on chat", err)
	//	}
	//}

	action := t.checker.Check(msg)
	switch action.Type {
	case checker.Ban:
		t.log.Warn("Banword in phrase", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text))
		t.moderation.Ban(msg.Chatter.UserID, action.Reason)
	case checker.Timeout:
		t.log.Warn("Spam is found", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text), slog.Int("duration", int(action.Duration.Seconds())))
		if t.cfg.Spam.SettingsDefault.Enabled {
			t.moderation.Timeout(msg.Chatter.UserID, int(action.Duration.Seconds()), action.Reason)
		}
	case checker.Delete:
		t.log.Warn("Muteword in phrase", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text))
		if err := t.DeleteChatMessage(msg.Broadcaster.UserID, msg.Message.ID); err != nil {
			t.log.Error("Failed to delete message on chat", err)
		}
	}
}
