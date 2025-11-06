package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
)

type ValidateResponse struct {
	ClientID  string   `json:"client_id"`
	Login     string   `json:"login"`
	Scopes    []string `json:"scopes"`
	UserID    string   `json:"user_id"`
	ExpiresIn int      `json:"expires_in"`
}

func (t *Twitch) ValidateToken(ctx context.Context, accessToken string) error {
	if accessToken == "" {
		return errors.New("empty access token")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "OAuth "+accessToken)

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var v ValidateResponse
		if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
			return err
		}
		return nil
	case http.StatusUnauthorized:
		return ErrUserAuthNotCompleted
	default:
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("validate request failed: %s", string(raw))
	}
}

func (t *Twitch) ensureUserToken(ctx context.Context, broadcasterID string) (*config.UserTokens, error) {
	token, ok := t.cfg.UsersTokens[broadcasterID]
	if !ok {
		return nil, ErrUserAuthNotCompleted
	}

	if time.Now().After(token.ObtainedAt.Add(time.Duration(token.ExpiresIn-300) * time.Second)) {
		resp, err := t.RefreshToken(ctx, broadcasterID, token)
		if err != nil {
			return nil, err
		}

		newToken := &config.UserTokens{
			AccessToken:  resp.AccessToken,
			RefreshToken: resp.RefreshToken,
			ExpiresIn:    resp.ExpiresIn,
			ObtainedAt:   time.Now(),
		}
		t.cfg.UsersTokens[broadcasterID] = newToken

		return newToken, nil
	}

	return token, nil
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (t *Twitch) RefreshToken(ctx context.Context, broadcasterID string, token *config.UserTokens) (*config.UserTokens, error) {
	if token == nil {
		return nil, ErrUserAuthNotCompleted
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", token.RefreshToken)
	data.Set("client_id", t.cfg.UserAccess.ClientID)
	data.Set("client_secret", t.cfg.UserAccess.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://id.twitch.tv/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		delete(t.cfg.UsersTokens, broadcasterID)
		return nil, ErrUserAuthNotCompleted
	}

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to refresh token: %s", string(raw))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &config.UserTokens{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
	}, nil
}
