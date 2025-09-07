package event_sub

import (
	"log/slog"
	"twitchspam/internal/app/adapters/twitch"
)

func (t *Twitch) checkModerate(modEvent twitch.ChannelModerateEvent) {
	switch modEvent.Action {
	case "delete":
		t.log.Info("The moderator deleted the user's message", slog.String("mod_username", modEvent.ModeratorUserName))
		t.stats.AddDeleted(modEvent.ModeratorUserName)
	case "timeout":
		t.log.Info("The moderator muted the user", slog.String("mod_username", modEvent.ModeratorUserName), slog.String("username", modEvent.Timeout.Username), slog.Time("expires_at", modEvent.Timeout.ExpiresAt), slog.String("reason", modEvent.Timeout.Reason))
		t.stats.AddTimeout(modEvent.ModeratorUserName)
	case "ban":
		t.log.Info("The moderator banned the user", slog.String("mod_username", modEvent.ModeratorUserName), slog.String("username", modEvent.Ban.Username), slog.String("reason", modEvent.Ban.Reason))
		t.stats.AddBan(modEvent.ModeratorUserName)
	}
}
