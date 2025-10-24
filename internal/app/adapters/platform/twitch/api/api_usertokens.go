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

func (t *Twitch) ensureUserToken(ctx context.Context, broadcasterID string) (*config.UserTokens, error) {
	token, ok := t.cfg.UsersTokens[broadcasterID]
	if !ok {
		return nil, ErrUserAuthNotCompleted
	}

	if time.Now().After(token.ObtainedAt.Add(time.Duration(token.ExpiresIn-300) * time.Second)) {
		resp, err := t.refreshUserToken(ctx, token)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh user token: %w", err)
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

func (t *Twitch) refreshUserToken(ctx context.Context, token *config.UserTokens) (*TokenResponse, error) {
	if token == nil {
		return nil, errors.New("token is nil")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", token.RefreshToken)
	data.Set("client_id", t.cfg.UserAccess.ClientID)
	data.Set("client_secret", t.cfg.UserAccess.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://id.twitch.tv/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to refresh token: %s", string(raw))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}
