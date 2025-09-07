package main

import (
	"fmt"
	"log"
	"twitchspam/internal/app/infrastructure/config"
)

func main() {
	manager, err := config.New("config.json")
	if err != nil {
		log.Fatal("Error loading config", err)
	}

	tests := map[string]bool{
		"пидр":           true,
		"аспидорас":      false,
		"пидорасище":     true,
		"негры пидорасы": true,
		"неграмотный":    false,
		"педики":         true,
		"педикюр":        false,
		"хачи":           true,
		"хачики":         true,
		"хачу":           false,
		"жид":            true,
		"жидом":          true,
		"неожиданно":     false,
		"хохляцкий":      true,
		"хохляц":         false,
		"хохол":          true,
		"хохохохол":      true,
		"хахлы":          true,
		"ХЗЫАВЗХВАПХЗПАХОХОЛАВЫАВЫАВ": true,
		"хохот": false,
		"к русне нет сочувствия":   true,
		"пендосы уебки":            true,
		"пиндосу":                  true,
		"кацапы":                   true,
		"ХВПАЗХАПВЗХПА ЧЕРНОЖОПЫЙ": true,
		"он чернажопый":            true,
		"нигга":                    true,
		"нигер":                    false,
		"нигере":                   false,
		"нигермания":               false,
		"нигеры":                   true,
		"гомосеки":                 true,
		"гомики":                   true,
		"инцелы":                   true,
		"куколды":                  true,
		"он симп":                  true,
		"симптом болезни":          false,
		"симпатия":                 false,
		"ytuhs gbljhfcs":           true,
		"gblh":                     true,
		"ytuh":                     true,
		"ytuhfvjnysq":              false,
		"кук олд":                  true,
		"кук олды":                 true,
		"отпидорасить":             true,
	}

	for input, expected := range tests {
		result := checkBadWords(manager.Get(), input)
		if result != expected {
			fmt.Printf("строка %q: ожидалось %v, получили %v\n", input, expected, result)
		}
	}
}

func checkBadWords(cfg *config.Config, s string) bool {
	for _, re := range cfg.Banwords.Regexp {
		if isMatch, _ := re.MatchString(s); isMatch {
			return true
		}
	}
	return false
}
