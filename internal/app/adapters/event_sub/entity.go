package event_sub

import (
	"encoding/json"
	"time"
)

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

type EventSubEnvelope struct {
	Subscription struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Version string `json:"version"`
	} `json:"subscription"`
	// тут Event — это `json.RawMessage`, чтобы потом парсить по типу
	Event json.RawMessage `json:"event"`
}

type AutomodHoldEvent struct {
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
}

type ChannelUpdateEvent struct {
	BroadcasterUserID           string   `json:"broadcaster_user_id"`
	BroadcasterUserLogin        string   `json:"broadcaster_user_login"`
	BroadcasterUserName         string   `json:"broadcaster_user_name"`
	Title                       string   `json:"title"`
	Language                    string   `json:"language"`
	CategoryID                  string   `json:"category_id"`
	CategoryName                string   `json:"category_name"`
	ContentClassificationLabels []string `json:"content_classification_labels"`
}

type ChannelUpdateMessage struct {
	Subscription struct {
		ID        string                 `json:"id"`
		Type      string                 `json:"type"`
		Version   string                 `json:"version"`
		Status    string                 `json:"status"`
		Cost      int                    `json:"cost"`
		Condition map[string]interface{} `json:"condition"`
		Transport map[string]interface{} `json:"transport"`
		CreatedAt time.Time              `json:"created_at"`
	} `json:"subscription"`
	Event ChannelUpdateEvent `json:"event"`
}

type ChannelModerateEvent struct {
	BroadcasterUserID string `json:"broadcaster_user_id"`
	ModeratorUserID   string `json:"moderator_user_id"`
	ModeratorUserName string `json:"moderator_user_name"`
	Action            string `json:"action"`
	Timeout           *struct {
		UserID    string    `json:"user_id"`
		Username  string    `json:"user_name"`
		ExpiresAt time.Time `json:"expires_at"`
		Reason    string    `json:"reason"`
	} `json:"timeout,omitempty"`
	Ban *struct {
		UserID   string `json:"user_id"`
		Username string `json:"user_name"`
		Reason   string `json:"reason"`
	} `json:"ban,omitempty"`
	Unban *struct {
		UserID string `json:"user_id"`
	} `json:"unban,omitempty"`
	Warn *struct {
		UserID string `json:"user_id"`
		Reason string `json:"reason"`
	} `json:"warn,omitempty"`
}
