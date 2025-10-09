package api

import (
	"net/url"
	"time"
	"twitchspam/internal/app/ports"
)

func (t *Twitch) GetLiveStream(channelID string) (*ports.Stream, error) {
	params := url.Values{}
	params.Set("user_id", channelID)
	params.Set("type", "live")

	var streamResp StreamResponse
	err := t.doTwitchRequest("GET", "https://api.twitch.tv/helix/streams?"+params.Encode(), nil, &streamResp)
	if err != nil {
		return nil, err
	}

	if len(streamResp.Data) == 0 {
		return &ports.Stream{ID: "", IsOnline: false, ViewerCount: 0}, nil
	}

	startTime, _ := time.Parse(time.RFC3339, streamResp.Data[0].StartedAt)
	loc := time.Now().Location()
	return &ports.Stream{
		ID:          streamResp.Data[0].ID,
		IsOnline:    true,
		ViewerCount: streamResp.Data[0].ViewerCount,
		StartedAt:   startTime.In(loc),
	}, nil
}
