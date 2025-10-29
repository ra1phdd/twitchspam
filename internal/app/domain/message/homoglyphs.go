package message

import (
	"strings"
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

func dominantLayout(word string) string {
	var rus, eng int
	for _, r := range word {
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

func removeHomoglyphs(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	var word []rune
	flushWord := func() {
		if len(word) == 0 {
			return
		}
		layout := dominantLayout(string(word))
		for _, r := range word {
			switch layout {
			case "rus":
				if mapped, ok := engToRus[r]; ok {
					b.WriteRune(mapped)
				} else {
					b.WriteRune(r)
				}
			case "eng":
				if mapped, ok := rusToEng[r]; ok {
					b.WriteRune(mapped)
				} else {
					b.WriteRune(r)
				}
			}
		}
		word = word[:0]
	}

	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			flushWord()
			b.WriteRune(r)
		} else {
			word = append(word, r)
		}
	}

	flushWord()
	return b.String()
}
