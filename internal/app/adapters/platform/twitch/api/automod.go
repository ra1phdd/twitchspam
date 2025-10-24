package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

func (t *Twitch) ManageHeldAutoModMessage(userID, msgID, action string) error {
	if userID == "" {
		return errors.New("userID is required")
	}
	if msgID == "" {
		return errors.New("msgID is required")
	}
	if action == "" {
		return errors.New("action is required")
	}

	if action != "ALLOW" && action != "DENY" {
		return errors.New("action must be either 'ALLOW' or 'DENY'")
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

	if _, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodPost,
		URL:    "https://api.twitch.tv/helix/moderation/automod/message",
		Token:  nil,
		Body:   bytes.NewReader(bodyBytes),
	}, nil); err != nil {
		return err
	}

	return nil
}
