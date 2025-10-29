package message

import "strings"

var zeroWidthRunes = map[rune]struct{}{
	'\u200B': {}, // ZERO WIDTH SPACE
	'\u200C': {}, // ZERO WIDTH NON-JOINER
	'\u200D': {}, // ZERO WIDTH JOINER
	'\u200E': {}, // LEFT-TO-RIGHT MARK
	'\u200F': {}, // RIGHT-TO-LEFT MARK
	'\u202A': {}, // LEFT-TO-RIGHT EMBEDDING
	'\u202B': {}, // RIGHT-TO-LEFT EMBEDDING
	'\u202C': {}, // POP DIRECTIONAL FORMATTING
	'\u202D': {}, // LEFT-TO-RIGHT OVERRIDE
	'\u202E': {}, // RIGHT-TO-LEFT OVERRIDE
	'\u2060': {}, // WORD JOINER
	'\u2061': {}, // FUNCTION APPLICATION
	'\u2062': {}, // INVISIBLE TIMES
	'\u2063': {}, // INVISIBLE SEPARATOR
	'\u2064': {}, // INVISIBLE PLUS
	'\u2066': {}, // LEFT-TO-RIGHT ISOLATE
	'\u2067': {}, // RIGHT-TO-LEFT ISOLATE
	'\u2068': {}, // FIRST STRONG ISOLATE
	'\u2069': {}, // POP DIRECTIONAL ISOLATE
	'\uFEFF': {}, // ZERO WIDTH NO-BREAK SPACE (BOM)
	'\u180E': {}, // MONGOLIAN VOWEL SEPARATOR (deprecated, still invisible)
}

func isInvisibleRune(r rune) bool {
	if _, bad := zeroWidthRunes[r]; bad {
		return true
	}

	switch {
	// Tag characters
	case r >= 0xE0020 && r <= 0xE007F:
		return true

	// Variation Selectors
	case r >= 0xFE00 && r <= 0xFE0F:
		return true

	// Variation Selectors Supplement
	case r >= 0xE0100 && r <= 0xE01EF:
		return true

	// Language tag & private-use invisible (Plane 14)
	case r >= 0xE0000 && r <= 0xE007F:
		return true

	// General control characters (C0 + DEL + C1)
	case r >= 0x0000 && r <= 0x001F, r == 0x007F, r >= 0x0080 && r <= 0x009F:
		return true

	// Bidirectional & format controls (RLM, LRM, ZWNJ, etc.)
	case r >= 0x200B && r <= 0x200F:
		return true
	case r >= 0x202A && r <= 0x202E:
		return true
	case r >= 0x2060 && r <= 0x206F:
		return true

	default:
		return false
	}
}

func removeInvisibleRune(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		if isInvisibleRune(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
