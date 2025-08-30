package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type UserResponse struct {
	Data []struct {
		ID    string `json:"id"`
		Login string `json:"login"`
	} `json:"data"`
}

type StreamResponse struct {
	Data []struct {
		ID          string `json:"id"`
		UserID      string `json:"user_id"`
		UserLogin   string `json:"user_login"`
		UserName    string `json:"user_name"`
		Title       string `json:"title"`
		GameName    string `json:"game_name"`
		StartedAt   string `json:"started_at"`
		Type        string `json:"type"`
		ViewerCount int    `json:"viewer_count"`
	} `json:"data"`
}

func doTwitchGet(url, oauth, clientID string, target interface{}) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+oauth)
	req.Header.Set("Client-Id", clientID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitch returned %s: %s", resp.Status, string(raw))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func GetChannelID(username, oauth, clientID string) (string, error) {
	var userResp UserResponse
	if err := doTwitchGet("https://api.twitch.tv/helix/users?login="+username, oauth, clientID, &userResp); err != nil {
		return "", err
	}
	if len(userResp.Data) == 0 {
		return "", fmt.Errorf("user %s not found", username)
	}
	return userResp.Data[0].ID, nil
}

func GetOnline(username, oauth, clientID string) (int, bool, error) {
	var streamResp StreamResponse
	err := doTwitchGet("https://api.twitch.tv/helix/streams?user_login="+username, oauth, clientID, &streamResp)
	if err != nil {
		return 0, false, err
	}
	if len(streamResp.Data) == 0 {
		return 0, false, nil
	}
	return streamResp.Data[0].ViewerCount, true, nil
}
