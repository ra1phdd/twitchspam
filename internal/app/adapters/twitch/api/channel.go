package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
	"twitchspam/internal/app/adapters/twitch"
	"twitchspam/internal/app/infrastructure/config"
)

type Twitch struct {
	cfg    *config.Config
	client *http.Client
}

func NewTwitch(cfg *config.Config, client *http.Client) *Twitch {
	return &Twitch{cfg: cfg, client: client}
}

func (t *Twitch) GetChannelID(username string) (string, error) {
	var userResp twitch.UserResponse
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
	StartedAt   time.Time
}

func (t *Twitch) GetLiveStream(broadcasterID string) (*Stream, error) {
	params := url.Values{}
	params.Set("user_id", broadcasterID)
	params.Set("type", "live")

	var streamResp twitch.StreamResponse
	err := t.doTwitchRequest("GET", "https://api.twitch.tv/helix/streams?"+params.Encode(), nil, &streamResp)
	if err != nil {
		return nil, err
	}

	if len(streamResp.Data) == 0 {
		return &Stream{ID: "", IsOnline: false, ViewerCount: 0}, nil
	}

	startTime, _ := time.Parse(time.RFC3339, streamResp.Data[0].StartedAt)
	loc := time.Now().Location()
	return &Stream{
		ID:          streamResp.Data[0].ID,
		IsOnline:    true,
		ViewerCount: streamResp.Data[0].ViewerCount,
		StartedAt:   startTime.In(loc),
	}, nil
}

func (t *Twitch) GetUrlVOD(broadcasterID string, streams []*config.Markers) (map[string]string, error) {
	remaining := len(streams)
	vods := make(map[string]string, remaining)

	params := url.Values{}
	params.Set("user_id", broadcasterID)
	params.Set("type", "archive")
	params.Set("first", "100")

	var videosResp twitch.VideoResponse
	err := t.doTwitchRequest("GET", "https://api.twitch.tv/helix/videos?"+params.Encode(), nil, &videosResp)
	if err != nil {
		return vods, err
	}

	if len(videosResp.Data) == 0 {
		return vods, fmt.Errorf("videos (user_id %s) not found", broadcasterID)
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

func (t *Twitch) SendChatMessage(broadcasterID, message string) error {
	reqBody := twitch.ChatMessageRequest{
		BroadcasterID: broadcasterID,
		SenderID:      t.cfg.App.UserID,
		Message:       message,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	var chatResp twitch.ChatMessageResponse
	err = t.doTwitchRequest("POST", "https://api.twitch.tv/helix/chat/messages", bytes.NewReader(bodyBytes), &chatResp)
	if err != nil {
		return err
	}

	if len(chatResp.Data) == 0 || !chatResp.Data[0].IsSent {
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
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+t.cfg.App.OAuth)
	req.Header.Set("Client-Id", t.cfg.App.ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitch returned %s: %s", resp.Status, string(raw))
	}

	if target == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
