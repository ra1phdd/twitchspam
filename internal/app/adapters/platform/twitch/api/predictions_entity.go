package api

type Outcome struct {
	Title string `json:"title"`
}

type CreatePredictionOptions struct {
	BroadcasterID    string    `json:"broadcaster_id"`
	Title            string    `json:"title"`
	Outcomes         []Outcome `json:"outcomes"`
	PredictionWindow int       `json:"prediction_window"`
}

type CreatePredictionResponse struct {
	Data []struct {
		ID               string  `json:"id"`
		BroadcasterID    string  `json:"broadcaster_id"`
		BroadcasterName  string  `json:"broadcaster_name"`
		BroadcasterLogin string  `json:"broadcaster_login"`
		Title            string  `json:"title"`
		WinningOutcomeID *string `json:"winning_outcome_id,omitempty"`
		Outcomes         []struct {
			ID            string `json:"id"`
			Title         string `json:"title"`
			Users         int    `json:"users"`
			ChannelPoints int    `json:"channel_points"`
			TopPredictors []struct {
				UserID            string `json:"user_id"`
				UserName          string `json:"user_name"`
				UserLogin         string `json:"user_login"`
				ChannelPointsUsed int    `json:"channel_points_used"`
				ChannelPointsWon  int    `json:"channel_points_won"`
			} `json:"top_predictors"`
			Color string `json:"color"`
		} `json:"outcomes"`
		PredictionWindow int     `json:"prediction_window"`
		Status           string  `json:"status"`
		CreatedAt        string  `json:"created_at"`
		EndedAt          *string `json:"ended_at,omitempty"`
		LockedAt         *string `json:"locked_at,omitempty"`
	} `json:"data"`
}

type EndPredictionOptions struct {
	BroadcasterID    string `json:"broadcaster_id"`
	ID               string `json:"id"`
	Status           string `json:"status"`
	WinningOutcomeID string `json:"winning_outcome_id,omitempty"`
}
