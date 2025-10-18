package handlers

import "twitchspam/internal/app/infrastructure/config"

type Handlers struct {
	manager *config.Manager
	state   string
}

func New(manager *config.Manager) *Handlers {
	s, err := generateSecureRandomString(52)
	if err != nil {
		panic(err)
	}

	return &Handlers{
		manager: manager,
		state:   s,
	}
}
