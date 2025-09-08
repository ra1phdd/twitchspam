package main

import (
	"fmt"
	"twitchspam/internal/app/domain/aliases"
)

func main() {
	als := map[string]string{
		"!olimp":     "!am mw add {query} фу,каз,хуйня",
		"!инта":      "!am mark donatov_net",
		"!olimp ban": "!am mw add 600 фу,каз,хуйня",
	}

	al := aliases.New(als)
	tests := map[string]string{
		"!olimp 600": "!am mw add 600 фу,каз,хуйня",
		"!инта":      "!am mark donatov_net",
		"!olimp ban": "!am mw add 600 фу,каз,хуйня",
	}

	for input, expected := range tests {
		result := al.ReplaceOne(input)
		if result != expected {
			fmt.Printf("строка %q: ожидалось %v, получили %v\n", input, expected, result)
		}
	}
}
