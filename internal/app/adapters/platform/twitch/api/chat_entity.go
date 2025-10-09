package api

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

type TimeoutRequest struct {
	Data TimeoutData `json:"data"`
}

type TimeoutData struct {
	UserID   string `json:"user_id"`
	Duration int    `json:"duration,omitempty"`
	Reason   string `json:"reason,omitempty"`
}
