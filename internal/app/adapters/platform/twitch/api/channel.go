package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

func (t *Twitch) GetChannelID(username string) (string, error) {
	var userResp UserResponse
	if _, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodGet,
		URL:    "https://api.twitch.tv/helix/users?login=" + username,
		Token:  nil,
		Body:   nil,
	}, &userResp); err != nil {
		return "", err
	}

	if len(userResp.Data) == 0 {
		return "", fmt.Errorf("user %s not found", username)
	}
	return userResp.Data[0].ID, nil
}

func (t *Twitch) UpdateChannelCategoryID(broadcasterID string, categoryID string) error {
	if broadcasterID == "" {
		return errors.New("broadcasterID is required")
	}

	params := url.Values{}
	params.Set("broadcaster_id", broadcasterID)
	opts := ChannelUpdateOptions{GameID: categoryID}

	bodyBytes, err := json.Marshal(opts)
	if err != nil {
		return err
	}

	token, err := t.ensureUserToken(context.Background(), broadcasterID)
	if err != nil {
		return err
	}

	if statusCode, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodPatch,
		URL:    "https://api.twitch.tv/helix/channels?" + params.Encode(),
		Token:  token,
		Body:   bytes.NewReader(bodyBytes),
	}, nil); err != nil {
		if statusCode == http.StatusUnauthorized {
			return ErrUserAuthNotCompleted
		}
		if statusCode == http.StatusBadRequest {
			return ErrBadRequest
		}
		return err
	}

	return nil
}

func (t *Twitch) UpdateChannelTitle(broadcasterID string, title string) error {
	if broadcasterID == "" {
		return errors.New("broadcasterID is required")
	}

	params := url.Values{}
	params.Set("broadcaster_id", broadcasterID)

	opts := ChannelUpdateOptions{Title: title}

	bodyBytes, err := json.Marshal(opts)
	if err != nil {
		return err
	}

	token, err := t.ensureUserToken(context.Background(), broadcasterID)
	if err != nil {
		return err
	}

	if statusCode, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodPatch,
		URL:    "https://api.twitch.tv/helix/channels?" + params.Encode(),
		Token:  token,
		Body:   bytes.NewReader(bodyBytes),
	}, nil); err != nil {
		if statusCode == http.StatusUnauthorized {
			return ErrUserAuthNotCompleted
		}
		if statusCode == http.StatusBadRequest {
			return ErrBadRequest
		}
		return err
	}

	return nil
}
