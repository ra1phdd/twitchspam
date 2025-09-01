package twitch

type TimeoutRequest struct {
	Data TimeoutData `json:"data"`
}

type TimeoutData struct {
	UserID   string `json:"user_id"`
	Duration int    `json:"duration,omitempty"`
	Reason   string `json:"reason,omitempty"`
}
