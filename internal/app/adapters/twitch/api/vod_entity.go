package api

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
