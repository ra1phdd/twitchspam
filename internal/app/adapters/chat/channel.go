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
		ID        string `json:"id"`
		UserID    string `json:"user_id"`
		UserLogin string `json:"user_login"`
		UserName  string `json:"user_name"`
		Title     string `json:"title"`
		GameName  string `json:"game_name"`
		StartedAt string `json:"started_at"`
		Type      string `json:"type"`
	} `json:"data"`
}

func GetChannelID(username, oauth, clientID string) (string, error) {
	url := fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+oauth)
	req.Header.Set("Client-Id", clientID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("twitch returned non-OK status %s: %s", resp.Status, string(raw))
	}

	var userResp UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(userResp.Data) == 0 {
		return "", fmt.Errorf("user %s not found", username)
	}

	return userResp.Data[0].ID, nil
}

func IsLive(username, oauth, clientID string) (bool, error) {
	url := fmt.Sprintf("https://api.twitch.tv/helix/streams?user_login=%s", username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+oauth)
	req.Header.Set("Client-Id", clientID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("twitch returned non-OK status %s: %s", resp.Status, string(raw))
	}

	var streamResp StreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&streamResp); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return len(streamResp.Data) > 0, nil
}
