package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
)

func (t *Twitch) UpdateChannelGameID(broadcasterID string, gameID string) error {
	if broadcasterID == "" {
		return fmt.Errorf("broadcasterID is required")
	}

	params := url.Values{}
	params.Set("broadcaster_id", broadcasterID)

	opts := ChannelUpdateOptions{GameID: gameID}

	bodyBytes, err := json.Marshal(opts)
	if err != nil {
		return err
	}

	return t.doTwitchRequest("PATCH", "https://api.twitch.tv/helix/channels?"+params.Encode(), bytes.NewReader(bodyBytes), nil)
}

func (t *Twitch) UpdateChannelTitle(broadcasterID string, title string) error {
	if broadcasterID == "" {
		return fmt.Errorf("broadcasterID is required")
	}

	params := url.Values{}
	params.Set("broadcaster_id", broadcasterID)

	opts := ChannelUpdateOptions{Title: title}

	bodyBytes, err := json.Marshal(opts)
	if err != nil {
		return err
	}

	return t.doTwitchRequest("PATCH", "https://api.twitch.tv/helix/channels?"+params.Encode(), bytes.NewReader(bodyBytes), nil)
}
