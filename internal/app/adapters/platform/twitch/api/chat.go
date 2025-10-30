package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"net/http"
	"net/url"
	"time"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/ports"
)

func (t *Twitch) SendChatMessages(channelID string, msgs *ports.AnswerType) {
	for _, message := range msgs.Text {
		text := message
		if msgs.IsReply {
			text = fmt.Sprintf("@%s, %s", msgs.ReplyUsername, message)
		}

		if err := t.SendChatMessage(channelID, text); err != nil {
			t.log.Error("Failed to send message on chat", err)
		}
	}
}

func (t *Twitch) SendChatMessage(channelID, message string) error {
	reqBody := ChatMessageRequest{
		BroadcasterID: channelID,
		SenderID:      t.cfg.App.UserID,
		Message:       message,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	var chatResp ChatMessageResponse
	if _, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodPost,
		URL:    "https://api.twitch.tv/helix/chat/messages",
		Token:  nil,
		Body:   bytes.NewReader(bodyBytes),
	}, &chatResp); err != nil {
		return err
	}

	if len(chatResp.Data) == 0 || !chatResp.Data[0].IsSent {
		return fmt.Errorf("%s is not sent", message)
	}
	return nil
}

func (t *Twitch) SendChatAnnouncements(channelID string, msgs *ports.AnswerType, color string) {
	for i, message := range msgs.Text {
		if err := t.SendChatAnnouncement(channelID, message, color); err != nil {
			t.log.Error("Failed to send message on chat", err)
		}

		if i >= 2 {
			break
		}
		time.Sleep(2 * time.Second)
	}
}

func (t *Twitch) SendChatAnnouncement(channelID, message, color string) error {
	reqBody := AnnouncementRequest{
		Message: message,
		Color:   color,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	params := url.Values{}
	params.Add("broadcaster_id", channelID)
	params.Add("moderator_id", t.cfg.App.UserID)

	if _, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodPost,
		URL:    "https://api.twitch.tv/helix/chat/announcements?" + params.Encode(),
		Token:  nil,
		Body:   bytes.NewReader(bodyBytes),
	}, nil); err != nil {
		return err
	}

	return nil
}

func (t *Twitch) DeleteChatMessage(channelName, channelID, messageID string) error {
	params := url.Values{}
	params.Set("broadcaster_id", channelID)
	params.Set("moderator_id", t.cfg.App.UserID)
	if messageID != "" {
		params.Set("message_id", messageID)
	}

	if _, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodDelete,
		URL:    "https://api.twitch.tv/helix/moderation/chat?" + params.Encode(),
		Token:  nil,
		Body:   nil,
	}, nil); err != nil {
		return err
	}

	metrics.ModerationActions.With(prometheus.Labels{"channel": channelName, "action": "delete"}).Inc()
	return nil
}

func (t *Twitch) TimeoutUser(channelName, channelID, userID string, duration int, reason string) {
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
		return
	}

	params := url.Values{}
	params.Set("broadcaster_id", channelID)
	params.Set("moderator_id", t.cfg.App.UserID)

	if _, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodPost,
		URL:    "https://api.twitch.tv/helix/moderation/bans?" + params.Encode(),
		Token:  nil,
		Body:   bytes.NewReader(bodyBytes),
	}, nil); err != nil {
		t.log.Error("Failed to timeout user", err, slog.Any("data", reqBody.Data))
		return
	}

	t.log.Info("Timeout applied successfully", slog.String("user_id", userID), slog.Int("duration", duration), slog.String("reason", reason))
	metrics.ModerationActions.With(prometheus.Labels{"channel": channelName, "action": "timeout"}).Inc()
}

func (t *Twitch) WarnUser(channelName, broadcasterID, userID, reason string) error {
	reqBody := WarnRequest{
		Data: WarnData{
			UserID: userID,
			Reason: reason,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.log.Error("Failed to marshal request body", err)
		return err
	}

	params := url.Values{}
	params.Set("broadcaster_id", broadcasterID)
	params.Set("moderator_id", broadcasterID)

	token, err := t.ensureUserToken(context.Background(), broadcasterID)
	if err != nil {
		t.log.Error("Failed to get user token", err)
		return err
	}

	if statusCode, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodPost,
		URL:    "https://api.twitch.tv/helix/moderation/warnings?" + params.Encode(),
		Token:  token,
		Body:   bytes.NewReader(bodyBytes),
	}, nil); err != nil {
		switch statusCode {
		case http.StatusUnauthorized:
			return ErrUserAuthNotCompleted
		case http.StatusBadRequest:
			return ErrBadRequest
		case http.StatusConflict:
			return errors.New("warning update conflict — another process is updating the user's warning state")
		}
		return err
	}

	t.log.Info("User warning issued successfully",
		slog.String("channel", channelName),
		slog.String("user_id", userID),
		slog.String("reason", reason))

	metrics.ModerationActions.With(prometheus.Labels{"channel": channelName, "action": "warn"}).Inc()
	return nil
}

func (t *Twitch) BanUser(channelName, channelID, userID string, reason string) {
	t.TimeoutUser(channelName, channelID, userID, 0, reason)

	metrics.ModerationActions.With(prometheus.Labels{"channel": channelName, "action": "ban"}).Inc()
	metrics.ModerationActions.With(prometheus.Labels{"channel": channelName, "action": "timeout"}).Dec()
}
