package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"twitchspam/internal/app/ports"
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

func (t *Twitch) SendChatMessages(msgs *ports.AnswerType) {
	for _, message := range msgs.Text {
		text := message
		if msgs.IsReply {
			text = fmt.Sprintf("@%s, %s", msgs.ReplyUsername, message)
		}

		if err := t.SendChatMessage(text); err != nil {
			t.log.Error("Failed to send message on chat", err)
		}
	}
}

func (t *Twitch) SendChatMessage(message string) error {
	reqBody := ChatMessageRequest{
		BroadcasterID: t.stream.ChannelID(),
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

	if len(chatResp.Data) == 0 || !chatResp.Data[0].IsSent {
		return fmt.Errorf("%s is not sent", message)
	}

	return nil
}

func (t *Twitch) SendChatAnnouncement(message, color string) error {
	reqBody := AnnouncementRequest{
		Message: message,
		Color:   color,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	queryParams := url.Values{}
	queryParams.Add("broadcaster_id", t.stream.ChannelID())
	queryParams.Add("moderator_id", t.cfg.App.UserID)

	err = t.doTwitchRequest("POST", "https://api.twitch.tv/helix/chat/announcements?"+queryParams.Encode(), bytes.NewReader(bodyBytes), nil)
	if err != nil {
		return err
	}

	return nil
}

// AnnouncementRequest represents the request body for sending an announcement
type AnnouncementRequest struct {
	Message string `json:"message"`
	Color   string `json:"color,omitempty"`
}

func (t *Twitch) DeleteChatMessage(messageID string) error {
	params := url.Values{}
	params.Set("broadcaster_id", t.stream.ChannelID())
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

func (t *Twitch) TimeoutUser(userID string, duration int, reason string) {
	reqBody := TimeoutRequest{
		Data: TimeoutData{
			UserID:   userID,
			Duration: duration,
			Reason:   reason,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.log.Error("Failed to marshel body", err)
	}

	params := url.Values{}
	params.Set("broadcaster_id", t.stream.ChannelID())
	params.Set("moderator_id", t.cfg.App.UserID)

	err = t.doTwitchRequest("POST", "https://api.twitch.tv/helix/moderation/bans?"+params.Encode(), bytes.NewReader(bodyBytes), nil)
	if err != nil {
		t.log.Error("Failed to send timeout request", err)
	}

	t.log.Info("Timeout applied successfully", slog.String("user_id", userID), slog.Int("duration", duration), slog.String("reason", reason))
}

func (t *Twitch) BanUser(userID string, reason string) {
	t.TimeoutUser(userID, 0, reason)
}
