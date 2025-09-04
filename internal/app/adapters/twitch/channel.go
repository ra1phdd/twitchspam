package twitch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (t *Twitch) GetChannelID(username string) (string, error) {
	var userResp UserResponse
	if err := t.doTwitchRequest("GET", "https://api.twitch.tv/helix/users?login="+username, nil, &userResp); err != nil {
		return "", err
	}
	if len(userResp.Data) == 0 {
		return "", fmt.Errorf("user %s not found", username)
	}
	return userResp.Data[0].ID, nil
}

type Stream struct {
	ID          string
	IsOnline    bool
	ViewerCount int
}

func (t *Twitch) GetLiveStream(broadcasterID string) (*Stream, error) {
	params := url.Values{}
	params.Set("user_id", broadcasterID)
	params.Set("type", "live")

	var streamResp StreamResponse
	err := t.doTwitchRequest("GET", "https://api.twitch.tv/helix/streams?"+params.Encode(), nil, &streamResp)
	if err != nil {
		return nil, err
	}

	if len(streamResp.Data) == 0 {
		return &Stream{ID: "", IsOnline: false, ViewerCount: 0}, nil
	}

	return &Stream{
		ID:          streamResp.Data[0].ID,
		IsOnline:    true,
		ViewerCount: streamResp.Data[0].ViewerCount,
	}, nil
}

func (t *Twitch) GetUrlVOD(id string) (string, error) {
	params := url.Values{}
	params.Set("id", id)
	params.Set("type", "archive")

	var videoResp VideoResponse
	err := t.doTwitchRequest("GET", "https://api.twitch.tv/helix/videos?"+params.Encode(), nil, &videoResp)
	if err != nil {
		return "", err
	}

	if len(videoResp.Data) == 0 {
		return "", fmt.Errorf("video %s not found", id)
	}
	return videoResp.Data[0].URL, nil
}

func (t *Twitch) SendChatMessage(broadcasterID, message string) error {
	reqBody := ChatMessageRequest{
		BroadcasterID: broadcasterID,
		SenderID:      t.cfg.App.UserID,
		Message:       message,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	var chatResp ChatMessageResponse
	err = t.doTwitchRequest("POST", "https://api.twitch.tv/helix/chat/messages", bytes.NewReader(bodyBytes), &chatResp)
	if err != nil {
		return err
	}

	if !chatResp.Data[0].IsSent {
		return fmt.Errorf("%s is not sent", message)
	}

	return nil
}

func (t *Twitch) DeleteChatMessage(broadcasterID, messageID string) error {
	params := url.Values{}
	params.Set("broadcaster_id", broadcasterID)
	params.Set("moderator_id", t.cfg.App.UserID)
	if messageID != "" {
		params.Set("message_id", messageID)
	}

	err := t.doTwitchRequest("DELETE", "https://api.twitch.tv/helix/moderation/chat?"+params.Encode(), nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (t *Twitch) doTwitchRequest(method, url string, body io.Reader, target interface{}) error {
	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("Authorization", "Bearer "+t.cfg.App.OAuth)
	req.Header.Set("Client-Id", t.cfg.App.ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitch returned %s: %s", resp.Status, string(raw))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
