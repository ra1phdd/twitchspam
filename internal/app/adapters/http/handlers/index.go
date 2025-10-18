package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
)

func (h *Handlers) IndexHandler(c *gin.Context) {
	if c.Query("code") != "" && c.Query("state") == h.state {
		h.CallbackHandler(c)
		c.JSON(http.StatusOK, gin.H{"status": "успешно"})
		return
	}

	cfg := h.manager.Get()
	authURL := fmt.Sprintf(
		"https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		url.QueryEscape(cfg.UserAccess.ClientID),
		url.QueryEscape(cfg.UserAccess.RedirectURL),
		url.QueryEscape("channel:read:vips channel:manage:vips channel:read:polls channel:manage:polls channel:read:predictions channel:manage:predictions moderator:read:followers channel:read:subscriptions"),
		h.state,
	)

	html := fmt.Sprintf(`<!DOCTYPE html>
		<html lang="ru">
		<head>
		<meta charset="UTF-8">
		<title>aFsYGGA Bot Auth</title>
		<style>
		  body { display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #9146FF; }
		  a.button {
			padding: 1em 2em;
			font-size: 1.2em;
			color: white;
			background-color: #9146FF;
			border: 2px solid white;
			border-radius: 6px;
			text-decoration: none;
			font-weight: bold;
			transition: background 0.3s, color 0.3s;
		  }
		  a.button:hover {
			background-color: white;
			color: #9146FF;
		  }
		</style>
		</head>
		<body>
		<a class="button" href="%s">Авторизоваться через Twitch</a>
		</body>
		</html>`, authURL)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func generateSecureRandomString(length int) (string, error) {
	bytes := make([]byte, (length*3)/4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
