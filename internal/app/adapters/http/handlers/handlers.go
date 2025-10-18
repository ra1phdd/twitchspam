package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"twitchspam/internal/app/infrastructure/config"
)

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

func generateSecureRandomString(length int) (string, error) {
	bytes := make([]byte, (length*3)/4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
