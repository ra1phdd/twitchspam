package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
)

func (h *Handlers) IndexHandler(c *gin.Context) {
	cfg := h.manager.Get()
	authURL := fmt.Sprintf(
		"https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&channel:read:vips+channel:manage:vips+channel:read:polls+channel:manage:polls+channel:read:predictions+channel:manage:predictions+moderator:read:followers+channel:read:subscriptions&state=RANDOM_STRING",
		url.QueryEscape(cfg.UserAccess.ClientID),
		url.QueryEscape(cfg.UserAccess.RedirectURL),
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
