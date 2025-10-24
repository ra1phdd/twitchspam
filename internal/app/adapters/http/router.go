package http

import (
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"twitchspam/internal/app/adapters/http/handlers"
	"twitchspam/internal/app/adapters/http/middlewares"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/pkg/logger"
)

type Router struct {
	router      *gin.Engine
	handlers    *handlers.Handlers
	middlewares *middlewares.Middlewares

	log     logger.Logger
	manager *config.Manager
}

func NewRouter(log logger.Logger, manager *config.Manager, client *http.Client) (*Router, error) {
	h, err := handlers.New(log, manager, client)
	if err != nil {
		return nil, err
	}

	r := &Router{
		router:      gin.Default(),
		handlers:    h,
		middlewares: middlewares.New(),
		log:         log,
		manager:     manager,
	}
	cfg := manager.Get()

	pprofGroup := r.router.Group("/", gin.BasicAuth(gin.Accounts{
		"admin": cfg.App.AuthToken,
	}))
	pprof.Register(pprofGroup)

	r.router.GET("/metrics", gin.BasicAuth(gin.Accounts{
		"admin": cfg.App.AuthToken,
	}), gin.WrapH(promhttp.Handler()))

	r.router.GET("/", r.handlers.IndexHandler)
	return r, nil
}

func (r *Router) Run() error {
	return r.router.Run(":80")
}
