package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ChatMessageRequest struct {
	BroadcasterID string `json:"broadcaster_id"`
	SenderID      string `json:"sender_id"`
	Message       string `json:"message"`
	ReplyParentID string `json:"reply_parent_message_id,omitempty"`
	ForSourceOnly *bool  `json:"for_source_only,omitempty"`
}

type ChatMessageResponse struct {
	Data []struct {
		MessageID  string `json:"message_id"`
		IsSent     bool   `json:"is_sent"`
		DropReason struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"drop_reason,omitempty"`
	} `json:"data"`
}

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

func SendChatMessage(broadcasterID, senderID, message, oauth, clientID string) error {
	url := "https://api.twitch.tv/helix/chat/messages"

	reqBody := ChatMessageRequest{
		BroadcasterID: broadcasterID,
		SenderID:      senderID,
		Message:       message,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+oauth)
	req.Header.Set("Client-Id", clientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("twitch returned %s: %s", resp.Status, string(raw))
	}

	var chatResp ChatMessageResponse
	if err := json.Unmarshal(raw, &chatResp); err != nil {
		return err
	}

	if !chatResp.Data[0].IsSent {
		return fmt.Errorf("%s is not sent", message)
	}

	return nil
}
