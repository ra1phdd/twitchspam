package automod

import "encoding/json"

type EventSubMessage struct {
	Metadata struct {
		MessageType string `json:"message_type"`
	} `json:"metadata"`
	Payload json.RawMessage `json:"payload"`
}

type SessionWelcomePayload struct {
	Session struct {
		ID string `json:"id"`
	} `json:"session"`
}

type Message struct {
	Subscription struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Version   string `json:"version"`
		Status    string `json:"status"`
		Cost      int    `json:"cost"`
		Condition struct {
			BroadcasterUserID string `json:"broadcaster_user_id"`
			ModeratorUserID   string `json:"moderator_user_id"`
		} `json:"condition"`
		Transport struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
		} `json:"transport"`
		CreatedAt string `json:"created_at"`
	} `json:"subscription"`
	Event struct {
		BroadcasterUserID string `json:"broadcaster_user_id"`
		UserID            string `json:"user_id"`
		MessageID         string `json:"message_id"`
		Message           struct {
			Text      string `json:"text"`
			Fragments []struct {
				Type      string  `json:"type"`
				Text      string  `json:"text"`
				Cheermote *string `json:"cheermote"`
				Emote     *string `json:"emote"`
			} `json:"fragments"`
		} `json:"message"`
		Category string `json:"category"`
		Level    int    `json:"level"`
		HeldAt   string `json:"held_at"`
	} `json:"event"`
}
