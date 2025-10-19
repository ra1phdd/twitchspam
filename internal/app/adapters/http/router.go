package http

import (
	"crypto/tls"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/acme/autocert"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/http/handlers"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/pkg/logger"
)

type Router struct {
	router   *gin.Engine
	handlers *handlers.Handlers

	log     logger.Logger
	manager *config.Manager
}

func NewRouter(log logger.Logger, manager *config.Manager) (*Router, error) {
	h, err := handlers.New(log, manager)
	if err != nil {
		return nil, err
	}

	r := &Router{
		router:   gin.Default(),
		handlers: h,
		log:      log,
		manager:  manager,
	}

	r.router.GET("/", r.handlers.IndexHandler)
	r.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	return r, nil
}

func (r *Router) Run() error {
	cfg := r.manager.Get()
	certManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(cfg.CertDomains...),
		Cache:      autocert.DirCache(".cache"),
	}

	r.log.Info("Starting server", slog.String("cert_domains", strings.Join(cfg.CertDomains, ", ")))
	go func() {
		httpServer := r.newServer(":80", certManager.HTTPHandler(nil))
		if err := httpServer.ListenAndServe(); err != nil {
			r.log.Error("HTTP server error", err)
		}
	}()

	server := r.newServer(":443", r.router)
	server.TLSConfig = &tls.Config{
		GetCertificate: certManager.GetCertificate,
		MinVersion:     tls.VersionTLS12,
	}

	if err := server.ListenAndServeTLS("", ""); err != nil {
		r.log.Error("Failed starting server", err)
		return err
	}

	return nil
}

func (r *Router) newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
}
