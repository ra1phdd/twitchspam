package api

import (
	"context"
	"net/http"
	"net/url"
	"time"
	"twitchspam/internal/app/ports"
)

func (t *Twitch) GetLiveStreams(channelIDs []string) ([]*ports.Stream, error) {
	params := url.Values{}
	for _, channelID := range channelIDs {
		params.Add("user_id", channelID)
	}
	params.Set("type", "live")

	var streamResp StreamResponse
	if _, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodGet,
		URL:    "https://api.twitch.tv/helix/streams?" + params.Encode(),
		Token:  nil,
		Body:   nil,
	}, &streamResp); err != nil {
		return nil, err
	}

	streams := make([]*ports.Stream, 0, len(streamResp.Data))
	loc := time.Now().Location()

	for _, stream := range streamResp.Data {
		startTime, _ := time.Parse(time.RFC3339, stream.StartedAt)

		streams = append(streams, &ports.Stream{
			ID:          stream.ID,
			UserID:      stream.UserID,
			UserLogin:   stream.UserLogin,
			Username:    stream.UserName,
			ViewerCount: stream.ViewerCount,
			StartedAt:   startTime.In(loc),
		})
	}

	return streams, nil
}
