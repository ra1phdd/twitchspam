package handlers

import "twitchspam/internal/app/infrastructure/config"

type Handlers struct {
	manager *config.Manager
}

func New(manager *config.Manager) *Handlers {
	return &Handlers{
		manager: manager,
	}
}
