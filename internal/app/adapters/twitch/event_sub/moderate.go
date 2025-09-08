package event_sub

import (
	"log/slog"
)

func (es *EventSub) checkModerate(modEvent ChannelModerateEvent) {
	switch modEvent.Action {
	case "delete":
		es.log.Info("The moderator deleted the user's message", slog.String("mod_username", modEvent.ModeratorUserName))
		es.stats.AddDeleted(modEvent.ModeratorUserName)
	case "timeout":
		es.log.Info("The moderator muted the user", slog.String("mod_username", modEvent.ModeratorUserName), slog.String("username", modEvent.Timeout.Username), slog.Time("expires_at", modEvent.Timeout.ExpiresAt), slog.String("reason", modEvent.Timeout.Reason))
		es.stats.AddTimeout(modEvent.ModeratorUserName)
	case "ban":
		es.log.Info("The moderator banned the user", slog.String("mod_username", modEvent.ModeratorUserName), slog.String("username", modEvent.Ban.Username), slog.String("reason", modEvent.Ban.Reason))
		es.stats.AddBan(modEvent.ModeratorUserName)
	}
}
