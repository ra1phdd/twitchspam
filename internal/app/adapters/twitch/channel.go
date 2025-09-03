package twitch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (i *Twitch) GetChannelID(username string) (string, error) {
	var userResp UserResponse
	if err := i.doTwitchRequest("GET", "https://api.twitch.tv/helix/users?login="+username, nil, &userResp); err != nil {
		return "", err
	}
	if len(userResp.Data) == 0 {
		return "", fmt.Errorf("user %s not found", username)
	}
	return userResp.Data[0].ID, nil
}

func (i *Twitch) GetOnline(username string) (int, bool, error) {
	var streamResp StreamResponse
	err := i.doTwitchRequest("GET", "https://api.twitch.tv/helix/streams?user_login="+username, nil, &streamResp)
	if err != nil {
		return 0, false, err
	}
	if len(streamResp.Data) == 0 {
		return 0, false, nil
	}
	return streamResp.Data[0].ViewerCount, true, nil
}

func (i *Twitch) SendChatMessage(broadcasterID, message string) error {
	reqBody := ChatMessageRequest{
		BroadcasterID: broadcasterID,
		SenderID:      i.cfg.App.UserID,
		Message:       message,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	var chatResp ChatMessageResponse
	err = i.doTwitchRequest("POST", "https://api.twitch.tv/helix/chat/messages", bytes.NewReader(bodyBytes), &chatResp)
	if err != nil {
		return err
	}

	if !chatResp.Data[0].IsSent {
		return fmt.Errorf("%s is not sent", message)
	}

	return nil
}

func (i *Twitch) DeleteChatMessage(broadcasterID, messageID string) error {
	params := url.Values{}
	params.Set("broadcaster_id", broadcasterID)
	params.Set("moderator_id", i.cfg.App.UserID)
	if messageID != "" {
		params.Set("message_id", messageID)
	}

	err := i.doTwitchRequest("DELETE", "https://api.twitch.tv/helix/moderation/chat?"+params.Encode(), nil, nil)
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
