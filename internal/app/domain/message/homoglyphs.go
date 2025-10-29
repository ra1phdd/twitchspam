package message

import (
	"unicode"
)

var rusToEng = map[rune]rune{
	'а': 'a', 'А': 'A',
	'е': 'e', 'Е': 'E',
	'о': 'o', 'О': 'O',
	'р': 'p', 'Р': 'P',
	'с': 'c', 'С': 'C',
	'у': 'y', 'У': 'Y',
	'х': 'x', 'Х': 'X',
}

var engToRus = map[rune]rune{
	'a': 'а', 'A': 'А',
	'e': 'е', 'E': 'Е',
	'o': 'о', 'O': 'О',
	'p': 'р', 'P': 'Р',
	'c': 'с', 'C': 'С',
	'y': 'у', 'Y': 'У',
	'x': 'х', 'X': 'Х',
}

func dominantLayout(runes []rune) string {
	var rus, eng int
	for _, r := range runes {
		if unicode.In(r, unicode.Cyrillic) {
			rus++
		} else if unicode.In(r, unicode.Latin) {
			eng++
		}
	}
	if rus >= eng {
		return "rus"
	}
	return "eng"
}
