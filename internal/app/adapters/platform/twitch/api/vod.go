package api

import (
	"fmt"
	"net/url"
	"twitchspam/internal/app/infrastructure/config"
)

func (t *Twitch) GetUrlVOD(channelID string, streams []*config.Markers) (map[string]string, error) {
	remaining := len(streams)
	vods := make(map[string]string, remaining)

	params := url.Values{}
	params.Set("user_id", channelID)
	params.Set("type", "archive")
	params.Set("first", "100")

	var videosResp VideoResponse
	err := t.doTwitchRequest("GET", "https://api.twitch.tv/helix/videos?"+params.Encode(), nil, nil, &videosResp)
	if err != nil {
		return vods, err
	}

	if len(videosResp.Data) == 0 {
		return vods, fmt.Errorf("videos (user_id %s) not found", channelID)
	}

	for i := range streams {
		for _, v := range videosResp.Data {
			if v.StreamID == streams[i].StreamID {
				vods[v.StreamID] = v.URL
				remaining--
			}
		}
	}

	return vods, nil
}
