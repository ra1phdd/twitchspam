package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
)

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
		url.QueryEscape("channel:manage:broadcast channel:manage:vips channel:manage:polls channel:manage:predictions moderator:read:followers channel:read:subscriptions moderator:manage:announcements moderator:manage:automod_settings moderator:manage:chat_settings moderator:manage:shield_mode moderator:manage:warnings"),
		h.state,
	)

	c.Redirect(http.StatusFound, authURL)
}
