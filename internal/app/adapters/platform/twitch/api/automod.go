package api

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func (t *Twitch) ManageHeldAutoModMessage(userID, msgID, action string) error {
	if userID == "" {
		return fmt.Errorf("userID is required")
	}
	if msgID == "" {
		return fmt.Errorf("msgID is required")
	}
	if action == "" {
		return fmt.Errorf("action is required")
	}

	if action != "ALLOW" && action != "DENY" {
		return fmt.Errorf("action must be either 'ALLOW' or 'DENY'")
	}

	requestBody := struct {
		UserID string `json:"user_id"`
		MsgID  string `json:"msg_id"`
		Action string `json:"action"`
	}{
		UserID: userID,
		MsgID:  msgID,
		Action: action,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	return t.doTwitchRequest("POST", "https://api.twitch.tv/helix/moderation/automod/message", bytes.NewReader(bodyBytes), nil)
}
