package event_sub

import (
	"log/slog"
)

func (es *EventSub) subscribeEvents(payload SessionWelcomePayload) {
	if err := es.subscribeEvent("channel.chat.message", "1", map[string]string{
		"broadcaster_user_id": es.stream.ChannelID(),
		"user_id":             es.cfg.App.UserID,
	}, payload.Session.ID); err != nil {
		es.log.Error("Failed to subscribe to event channel.chat.message", err, slog.String("event", "channel.chat.message"))
	}

	if err := es.subscribeEvent("automod.message.hold", "1", map[string]string{
		"broadcaster_user_id": es.stream.ChannelID(),
		"moderator_user_id":   es.cfg.App.UserID,
	}, payload.Session.ID); err != nil {
		es.log.Error("Failed to subscribe to event automod", err, slog.String("event", "automod.message.hold"))
	}

	if err := es.subscribeEvent("stream.online", "1", map[string]string{
		"broadcaster_user_id": es.stream.ChannelID(),
	}, payload.Session.ID); err != nil {
		es.log.Error("Failed to subscribe to event", err, slog.String("event", "stream.online"))
	}

	if err := es.subscribeEvent("stream.offline", "1", map[string]string{
		"broadcaster_user_id": es.stream.ChannelID(),
	}, payload.Session.ID); err != nil {
		es.log.Error("Failed to subscribe to event", err, slog.String("event", "stream.offline"))
	}

	if err := es.subscribeEvent("channel.update", "2", map[string]string{
		"broadcaster_user_id": es.stream.ChannelID(),
	}, payload.Session.ID); err != nil {
		es.log.Error("Failed to subscribe to event", err, slog.String("event", "channel.update"))
	}

	if err := es.subscribeEvent("channel.moderate", "2", map[string]string{
		"broadcaster_user_id": es.stream.ChannelID(),
		"moderator_user_id":   es.cfg.App.UserID,
	}, payload.Session.ID); err != nil {
		es.log.Error("Failed to subscribe to event", err, slog.String("event", "channel.moderate"))
	}
}
