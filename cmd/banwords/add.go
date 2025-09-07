package main

import (
	"log"
	"twitchspam/internal/app/domain/regex"
	"twitchspam/internal/app/infrastructure/config"
)

func main() {
	manager, err := config.New("config.json")
	if err != nil {
		log.Fatal("Error loading config", err)
	}

	reg := regex.New()
	re, err := reg.Parse("r'пидор(а|у|ом|е|ы|ов|ам|ами|ах)?'")
	if err != nil {
		log.Fatal("Error parsing regexp", err)
	}

	manager.Update(func(cfg *config.Config) {
		cfg.Banwords.Regexp = append(cfg.Banwords.Regexp, re)
	})
}
