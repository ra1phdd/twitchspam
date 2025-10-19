package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/pkg/logger"
)

type Handlers struct {
	log     logger.Logger
	manager *config.Manager
	state   string
}

func New(log logger.Logger, manager *config.Manager) (*Handlers, error) {
	s, err := generateSecureRandomString(52)
	if err != nil {
		log.Error("Failed to generate secure random string", err)
		return nil, err
	}

	return &Handlers{
		log:     log,
		manager: manager,
		state:   s,
	}, nil
}

func (h *Handlers) IndexHandler(c *gin.Context) {
	if c.Query("code") != "" && c.Query("state") == h.state {
		h.CallbackHandler(c)
		return
	}

	cfg := h.manager.Get()
	authURL := fmt.Sprintf(
		"https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		url.QueryEscape(cfg.UserAccess.ClientID),
		url.QueryEscape(cfg.UserAccess.RedirectURL),
		url.QueryEscape("channel:manage:broadcast channel:manage:raids channel:manage:vips channel:manage:polls channel:manage:predictions moderator:read:followers channel:read:subscriptions moderator:manage:announcements moderator:manage:automod_settings moderator:manage:chat_settings moderator:manage:shield_mode moderator:manage:warnings"),
		h.state,
	)

	c.Redirect(http.StatusFound, authURL)
}

func generateSecureRandomString(length int) (string, error) {
	bytes := make([]byte, (length*3)/4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
