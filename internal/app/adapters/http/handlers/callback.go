package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
	"twitchspam/internal/app/infrastructure/config"
)

func (h *Handlers) CallbackHandler(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		h.log.Warn("Missing authorization code in callback", slog.String("path", c.FullPath()), slog.String("client_ip", c.ClientIP()))
		c.String(http.StatusBadRequest, "Missing authorization code")
		return
	}

	h.log.Debug("Received authorization code from Twitch OAuth callback",
		slog.String("code_prefix", code[:min(len(code), 6)]),
		slog.String("client_ip", c.ClientIP()),
	)

	token, err := h.exchangeCodeForToken(code)
	if err != nil {
		h.log.Error("Failed to exchange authorization code for tokens", err, slog.String("code_prefix", code[:min(len(code), 6)]))
		c.String(http.StatusInternalServerError, fmt.Sprintf("Token exchange error: %v", err))
		return
	}

	expiry := token.ObtainedAt.Add(time.Duration(token.ExpiresIn) * time.Second)
	h.log.Info("OAuth tokens successfully obtained",
		slog.Time("obtained_at", token.ObtainedAt),
		slog.Int("expires_in_sec", token.ExpiresIn),
		slog.Time("expires_at", expiry),
		slog.Bool("has_refresh_token", token.RefreshToken != ""),
	)

	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/users", nil)
	if err != nil {
		h.log.Error("Failed to create Twitch user info request", err)
		c.String(http.StatusInternalServerError, "Internal error creating request")
		return
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Client-Id", h.manager.Get().UserAccess.ClientID)

	h.log.Trace("Prepared Twitch user info request",
		slog.String("url", req.URL.String()),
		slog.String("client_id", h.manager.Get().UserAccess.ClientID),
	)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.log.Error("Failed to execute Twitch user info request", err)
		c.String(http.StatusInternalServerError, "Error sending request to Twitch")
		return
	}
	defer resp.Body.Close()

	h.log.Debug("Received response from Twitch user info endpoint", slog.Int("status_code", resp.StatusCode))

	var userResp struct {
		Data []struct {
			ID          string `json:"id"`
			Login       string `json:"login"`
			DisplayName string `json:"display_name"`
			Email       string `json:"email"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		h.log.Error("Failed to decode Twitch user info response", err)
		c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to parse Twitch response: %v", err))
		return
	}

	if len(userResp.Data) == 0 {
		h.log.Warn("Received empty user data from Twitch", slog.Int("status_code", resp.StatusCode))
		c.String(http.StatusInternalServerError, "No user data received")
		return
	}

	user := userResp.Data[0]
	h.log.Info("Fetched Twitch user info",
		slog.String("user_id", user.ID),
		slog.String("login", user.Login),
		slog.String("display_name", user.DisplayName),
		slog.String("email", user.Email),
	)

	if err := h.manager.Update(func(cfg *config.Config) {
		cfg.UsersTokens[user.ID] = token
	}); err != nil {
		h.log.Error("Failed to update user tokens in configuration", err,
			slog.String("user_id", user.ID),
			slog.Time("token_obtained_at", token.ObtainedAt),
			slog.Int("expires_in_sec", token.ExpiresIn),
		)
		c.String(http.StatusInternalServerError, "Internal error updating configuration")
		return
	}

	h.log.Debug("User tokens stored in configuration",
		slog.String("user_id", user.ID),
		slog.Time("token_obtained_at", token.ObtainedAt),
		slog.Time("token_expires_at", expiry),
	)

	c.String(http.StatusOK, "Success")
}

func (h *Handlers) exchangeCodeForToken(code string) (*config.UserTokens, error) {
	cfg := h.manager.Get()
	h.log.Debug("Starting Twitch token exchange",
		slog.String("client_id", cfg.UserAccess.ClientID),
		slog.String("redirect_uri", cfg.UserAccess.RedirectURL),
		slog.String("grant_type", "authorization_code"),
	)

	data := url.Values{}
	data.Set("client_id", cfg.UserAccess.ClientID)
	data.Set("client_secret", cfg.UserAccess.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", cfg.UserAccess.RedirectURL)

	resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", data)
	if err != nil {
		h.log.Error("Failed to send token exchange request to Twitch", err)
		return nil, err
	}
	defer resp.Body.Close()

	h.log.Debug("Received response from Twitch token endpoint", slog.Int("status_code", resp.StatusCode))
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		h.log.Error("Unexpected status code from Twitch token endpoint", nil,
			slog.Int("status_code", resp.StatusCode),
			slog.String("response_body", string(body)),
		)
		return nil, fmt.Errorf("twitch returned %d", resp.StatusCode)
	}

	var tokens config.UserTokens
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		h.log.Error("Failed to decode Twitch token response", err)
		return nil, err
	}

	tokens.ObtainedAt = time.Now()
	expiry := tokens.ObtainedAt.Add(time.Duration(tokens.ExpiresIn) * time.Second)

	h.log.Info("Successfully obtained new Twitch tokens",
		slog.Int("expires_in_sec", tokens.ExpiresIn),
		slog.Time("obtained_at", tokens.ObtainedAt),
		slog.Time("expires_at", expiry),
		slog.Bool("has_refresh_token", tokens.RefreshToken != ""),
	)
	return &tokens, nil
}
