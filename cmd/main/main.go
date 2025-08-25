package main

import (
	"log"
	"twitchspam/internal/pkg/app"
)

func main() {
	if err := app.New(); err != nil {
		log.Fatal(err)
	}

	select {}
}
