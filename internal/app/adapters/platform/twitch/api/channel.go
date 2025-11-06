package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

func (t *Twitch) GetChannelIDs(usernames []string) (map[string]string, error) {
	if len(usernames) == 0 {
		return nil, errors.New("no usernames provided")
	}

	params := url.Values{}
	for _, u := range usernames {
		params.Add("login", strings.TrimPrefix(u, "@"))
	}

	var userResp UserResponse
	if _, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodGet,
		URL:    "https://api.twitch.tv/helix/users?" + params.Encode(),
		Token:  nil,
		Body:   nil,
	}, &userResp); err != nil {
		return nil, err
	}

	result := make(map[string]string, len(userResp.Data))
	for _, u := range userResp.Data {
		result[u.Login] = u.ID
	}

	return result, nil
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
