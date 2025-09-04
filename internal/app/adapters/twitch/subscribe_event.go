package twitch

import "log/slog"

func (t *Twitch) subscribeEvents(payload SessionWelcomePayload) {
	if err := t.subscribeEvent("channel.chat.message", "1", map[string]string{
		"broadcaster_user_id": t.stream.ChannelID(),
		"user_id":             t.cfg.App.UserID,
	}, payload.Session.ID); err != nil {
		t.log.Error("Failed to subscribe to event channel.chat.message", err, slog.String("event", "channel.chat.message"))
	}

	if err := t.subscribeEvent("automod.message.hold", "1", map[string]string{
		"broadcaster_user_id": t.stream.ChannelID(),
		"moderator_user_id":   t.cfg.App.UserID,
	}, payload.Session.ID); err != nil {
		t.log.Error("Failed to subscribe to event automod", err, slog.String("event", "automod.message.hold"))
	}

	if err := t.subscribeEvent("stream.online", "1", map[string]string{
		"broadcaster_user_id": t.stream.ChannelID(),
	}, payload.Session.ID); err != nil {
		t.log.Error("Failed to subscribe to event", err, slog.String("event", "stream.online"))
	}

	if err := t.subscribeEvent("stream.offline", "1", map[string]string{
		"broadcaster_user_id": t.stream.ChannelID(),
	}, payload.Session.ID); err != nil {
		t.log.Error("Failed to subscribe to event", err, slog.String("event", "stream.offline"))
	}

	if err := t.subscribeEvent("channel.update", "2", map[string]string{
		"broadcaster_user_id": t.stream.ChannelID(),
	}, payload.Session.ID); err != nil {
		t.log.Error("Failed to subscribe to event", err, slog.String("event", "channel.update"))
	}

	if err := t.subscribeEvent("channel.moderate", "2", map[string]string{
		"broadcaster_user_id": t.stream.ChannelID(),
		"moderator_user_id":   t.cfg.App.UserID,
	}, payload.Session.ID); err != nil {
		t.log.Error("Failed to subscribe to event", err, slog.String("event", "channel.moderate"))
	}
}
