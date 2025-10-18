package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
	"twitchspam/internal/app/infrastructure/config"
)

func (h *Handlers) CallbackHandler(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	fmt.Println(state)
	if code == "" {
		c.String(400, "Нет кода авторизации")
		return
	}

	token, err := h.exchangeCodeForToken(code)
	if err != nil {
		c.String(http.StatusInternalServerError, "Ошибка обмена токена: %v", err)
		return
	}

	req, _ := http.NewRequest("GET", "https://api.twitch.tv/helix/users", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Client-Id", h.manager.Get().UserAccess.ClientID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var userResp struct {
		Data []struct {
			ID          string `json:"id"`
			Login       string `json:"login"`
			DisplayName string `json:"display_name"`
			Email       string `json:"email"`
		} `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&userResp)
	if err != nil {
		c.String(http.StatusInternalServerError, "Ошибка обмена токена: %v", err)
		return
	}

	fmt.Println("User login:", userResp.Data[0].Login)

	h.manager.Update(func(cfg *config.Config) {
		cfg.UsersTokens[userResp.Data[0].ID] = *token
	})

	c.String(200, "Успешно!")
}

func (h *Handlers) exchangeCodeForToken(code string) (*config.UserTokens, error) {
	cfg := h.manager.Get()

	data := url.Values{}
	data.Set("client_id", cfg.UserAccess.ClientID)
	data.Set("client_secret", cfg.UserAccess.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", cfg.UserAccess.RedirectURL)

	resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("twitch returned %d: %s", resp.StatusCode, string(body))
	}

	var tokens config.UserTokens
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}

	tokens.ObtainedAt = time.Now()
	return &tokens, nil
}
