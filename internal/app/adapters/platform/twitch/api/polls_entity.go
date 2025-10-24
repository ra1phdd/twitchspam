package api

type PollChoice struct {
	Title string `json:"title"`
}

type CreatePollOptions struct {
	BroadcasterID              string       `json:"broadcaster_id"`
	Title                      string       `json:"title"`
	Choices                    []PollChoice `json:"choices"`
	Duration                   int          `json:"duration"`
	ChannelPointsVotingEnabled bool         `json:"channel_points_voting_enabled,omitempty"`
	ChannelPointsPerVote       int          `json:"channel_points_per_vote,omitempty"`
}

type PollChoiceResponse struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	Votes              int    `json:"votes"`
	ChannelPointsVotes int    `json:"channel_points_votes"`
	BitsVotes          int    `json:"bits_votes"`
}

type CreatePollResponse struct {
	Data []struct {
		ID                         string               `json:"id"`
		BroadcasterID              string               `json:"broadcaster_id"`
		BroadcasterName            string               `json:"broadcaster_name"`
		BroadcasterLogin           string               `json:"broadcaster_login"`
		Title                      string               `json:"title"`
		Choices                    []PollChoiceResponse `json:"choices"`
		ChannelPointsVotingEnabled bool                 `json:"channel_points_voting_enabled"`
		ChannelPointsPerVote       int                  `json:"channel_points_per_vote"`
		Status                     string               `json:"status"`
		Duration                   int                  `json:"duration"`
		StartedAt                  string               `json:"started_at"`
		EndedAt                    *string              `json:"ended_at"`
	} `json:"data"`
}

type EndPollOptions struct {
	BroadcasterID string `json:"broadcaster_id"`
	ID            string `json:"id"`
	Status        string `json:"status"`
}
