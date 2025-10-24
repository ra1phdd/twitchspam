package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"twitchspam/internal/app/infrastructure/config"
)

func (t *Twitch) GetUrlVOD(channelID string, streams []*config.Markers) (map[string]string, error) {
	vods := make(map[string]string, len(streams))

	params := url.Values{}
	params.Set("user_id", channelID)
	params.Set("type", "archive")
	params.Set("first", "100")

	var videosResp VideoResponse
	if _, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodGet,
		URL:    "https://api.twitch.tv/helix/videos?" + params.Encode(),
		Token:  nil,
		Body:   nil,
	}, &videosResp); err != nil {
		return nil, err
	}

	if len(videosResp.Data) == 0 {
		return vods, fmt.Errorf("videos (user_id %s) not found", channelID)
	}

	for i := range streams {
		for _, v := range videosResp.Data {
			if v.StreamID == streams[i].StreamID {
				vods[v.StreamID] = v.URL
				break
			}
		}
	}

	return vods, nil
}
