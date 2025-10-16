package api

import (
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
	err := t.doTwitchRequest("GET", "https://api.twitch.tv/helix/streams?"+params.Encode(), nil, &streamResp)
	if err != nil {
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
