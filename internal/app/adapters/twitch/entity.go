package twitch

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

type ChatMessageEvent struct {
	BroadcasterUserID    string `json:"broadcaster_user_id"`
	BroadcasterUserLogin string `json:"broadcaster_user_login"`
	BroadcasterUserName  string `json:"broadcaster_user_name"`

	//SourceBroadcasterUserID *string `json:"source_broadcaster_user_id"`
	//SourceBroadcasterUserLogin *string `json:"source_broadcaster_user_login"`
	//SourceBroadcasterUserName  *string `json:"source_broadcaster_user_name"`

	ChatterUserID    string `json:"chatter_user_id"`
	ChatterUserLogin string `json:"chatter_user_login"`
	ChatterUserName  string `json:"chatter_user_name"`

	MessageID string `json:"message_id"`
	//SourceMessageID *string `json:"source_message_id"`
	//IsSourceOnly    *bool   `json:"is_source_only"`

	Message struct {
		Text      string `json:"text"`
		Fragments []struct {
			Type string `json:"type"`
			Text string `json:"text"`
			//Cheermote *string `json:"cheermote"`
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

	//SourceBadges *[]struct {
	//		SetID string `json:"set_id"`
	//		ID    string `json:"id"`
	//		Info  string `json:"info"`
	//	} `json:"badges"`

	MessageType string `json:"message_type"`
	//Cheer                       *string `json:"cheer"`
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
	//ChannelPointsCustomRewardID *string `json:"channel_points_custom_reward_id"`
	//ChannelPointsAnimationID    *string `json:"channel_points_animation_id"`
}

type AutomodHoldEvent struct {
	BroadcasterUserID string `json:"broadcaster_user_id"`
	UserID            string `json:"user_id"`
	UserName          string `json:"user_name"`
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

type ChatMessageRequest struct {
	BroadcasterID string `json:"broadcaster_id"`
	SenderID      string `json:"sender_id"`
	Message       string `json:"message"`
	ReplyParentID string `json:"reply_parent_message_id,omitempty"`
	ForSourceOnly *bool  `json:"for_source_only,omitempty"`
}

type ChatMessageResponse struct {
	Data []struct {
		MessageID  string `json:"message_id"`
		IsSent     bool   `json:"is_sent"`
		DropReason struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"drop_reason,omitempty"`
	} `json:"data"`
}

type UserResponse struct {
	Data []struct {
		ID    string `json:"id"`
		Login string `json:"login"`
	} `json:"data"`
}

type StreamResponse struct {
	Data []struct {
		ID           string   `json:"id"`
		UserID       string   `json:"user_id"`
		UserLogin    string   `json:"user_login"`
		UserName     string   `json:"user_name"`
		GameID       string   `json:"game_id"`
		GameName     string   `json:"game_name"`
		Type         string   `json:"type"`
		Title        string   `json:"title"`
		Tags         []string `json:"tags"`
		ViewerCount  int      `json:"viewer_count"`
		StartedAt    string   `json:"started_at"`
		Language     string   `json:"language"`
		ThumbnailURL string   `json:"thumbnail_url"`
		IsMature     bool     `json:"is_mature"`
	} `json:"data"`
}

type VideoResponse struct {
	Data []struct {
		ID            string `json:"id"`
		StreamID      string `json:"stream_id"`
		UserID        string `json:"user_id"`
		UserLogin     string `json:"user_login"`
		UserName      string `json:"user_name"`
		Title         string `json:"title"`
		Description   string `json:"description"`
		CreatedAt     string `json:"created_at"`
		PublishedAt   string `json:"published_at"`
		URL           string `json:"url"`
		ThumbnailURL  string `json:"thumbnail_url"`
		Viewable      string `json:"viewable"`
		ViewCount     int    `json:"view_count"`
		Language      string `json:"language"`
		Type          string `json:"type"`
		Duration      string `json:"duration"`
		MutedSegments []struct {
			Duration int `json:"duration"`
			Offset   int `json:"offset"`
		} `json:"muted_segments"`
	} `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}
