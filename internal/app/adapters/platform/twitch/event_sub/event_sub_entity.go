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
	Event json.RawMessage `json:"event"`
}

type StreamMessageEvent struct {
	BroadcasterUserID    string `json:"broadcaster_user_id"`
	BroadcasterUserLogin string `json:"broadcaster_user_login"`
}

type ChatMessageEvent struct {
	BroadcasterUserID    string `json:"broadcaster_user_id"`
	BroadcasterUserLogin string `json:"broadcaster_user_login"`
	BroadcasterUserName  string `json:"broadcaster_user_name"`

	// SourceBroadcasterUserID *string `json:"source_broadcaster_user_id"`
	// SourceBroadcasterUserLogin *string `json:"source_broadcaster_user_login"`
	// SourceBroadcasterUserName  *string `json:"source_broadcaster_user_name"`

	ChatterUserID    string `json:"chatter_user_id"`
	ChatterUserLogin string `json:"chatter_user_login"`
	ChatterUserName  string `json:"chatter_user_name"`

	MessageID string `json:"message_id"`
	// SourceMessageID *string `json:"source_message_id"`
	// IsSourceOnly    *bool   `json:"is_source_only"`

	Message struct {
		Text      string `json:"text"`
		Fragments []struct {
			Type string `json:"type"`
			Text string `json:"text"`
			// Cheermote *string `json:"cheermote"`
			Emote *struct {
				ID         string   `json:"id"`
				EmoteSetID string   `json:"emote_set_id"`
				OwnerID    string   `json:"owner_id"`
				Format     []string `json:"format"`
			} `json:"emote"`
			Mention *struct {
				UserID    string `json:"user_id"`
				UserLogin string `json:"user_login"`
				UserName  string `json:"user_name"`
			} `json:"mention"`
		} `json:"fragments"`
	} `json:"message"`

	Color string `json:"color"`

	Badges []struct {
		SetID string `json:"set_id"`
		ID    string `json:"id"`
		Info  string `json:"info"`
	} `json:"badges"`

	// SourceBadges *[]struct {
	//		SetID string `json:"set_id"`
	//		ID    string `json:"id"`
	//		Info  string `json:"info"`
	// } `json:"badges"`

	MessageType string `json:"message_type"`
	// Cheer                       *string `json:"cheer"`
	Reply *struct {
		ParentMessageID   string `json:"parent_message_id"`
		ParentMessageBody string `json:"parent_message_body"`
		ParentUserID      string `json:"parent_user_id"`
		ParentUserName    string `json:"parent_user_name"`
		ParentUserLogin   string `json:"parent_user_login"`
		ThreadMessageID   string `json:"thread_message_id"`
		ThreadUserID      string `json:"thread_user_id"`
		ThreadUserName    string `json:"thread_user_name"`
		ThreadUserLogin   string `json:"thread_user_login"`
	} `json:"reply"`
	// ChannelPointsCustomRewardID *string `json:"channel_points_custom_reward_id"`
	// ChannelPointsAnimationID    *string `json:"channel_points_animation_id"`
}

type AutomodHoldEvent struct {
	BroadcasterUserID    string `json:"broadcaster_user_id"`
	BroadcasterUserLogin string `json:"broadcaster_user_login"`
	BroadcasterUserName  string `json:"broadcaster_user_name"`
	UserID               string `json:"user_id"`
	UserLogin            string `json:"user_login"`
	UserName             string `json:"user_name"`
	MessageID            string `json:"message_id"`
	Message              struct {
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
