package api

type ChannelUpdateOptions struct {
	GameID                   string   `json:"game_id,omitempty"`
	Title                    string   `json:"title,omitempty"`
	BroadcasterLanguage      string   `json:"broadcaster_language,omitempty"`
	Delay                    *int     `json:"delay,omitempty"` // nil если не хотим менять
	Tags                     []string `json:"tags,omitempty"`
	IsBrandedContent         *bool    `json:"is_branded_content,omitempty"`
	ContentClassificationIDs []struct {
		ID        string `json:"id"`
		IsEnabled bool   `json:"is_enabled"`
	} `json:"content_classification_labels,omitempty"`
}
