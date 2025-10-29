package domain

import (
	"strings"
	"unicode"
)

type ChatMessage struct {
	Broadcaster Broadcaster
	Chatter     Chatter
	Message     Message
	Reply       *Reply
}

type Broadcaster struct {
	UserID   string
	Login    string
	Username string
}

type Chatter struct {
	UserID        string
	Login         string
	Username      string
	IsBroadcaster bool
	IsMod         bool
	IsVip         bool
	IsSubscriber  bool
}

type Reply struct {
	ParentChatter Chatter
	ParentMessage Message
}

type Message struct {
	ID        string
	Text      MessageText
	EmoteOnly bool     // если Fragments type == "text" отсутствует
	Emotes    []string // text в Fragments при type == "emote"
	IsFirst   func() bool
}

type MessageText struct {
	Original string

	cacheText  map[uint64]string
	cacheWords map[uint64][]string
}

func (m *MessageText) ReplaceOriginal(text string) {
	m.Original = text
	m.cacheText = make(map[uint64]string)
	m.cacheWords = make(map[uint64][]string)
}

func lower(s string) string {
	return strings.ToLower(s)
}

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

func removePunctuation(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	atWordStart := true
	lastWasSpace := false

	for _, r := range s {
		if isInvisibleRune(r) {
			continue
		}

		if atWordStart {
			if r == '!' {
				b.WriteRune(r)
				atWordStart = false
				continue
			}
			atWordStart = false
		}

		if unicode.IsSpace(r) {
			atWordStart = true
			if b.Len() != 0 {
				lastWasSpace = true
			}

			continue
		}

		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if lastWasSpace {
				b.WriteRune(' ')
				lastWasSpace = false
			}
			b.WriteRune(r)
		}
	}

	return b.String()
}

func removeDuplicateLetters(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	var prev rune
	for _, r := range s {
		if r == prev && unicode.IsLetter(r) {
			continue
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}

type TextOptionFuncWithID struct {
	Fn func(string) string
	ID uint64
}

var LowerOption = TextOptionFuncWithID{
	Fn: lower,
	ID: 1,
}

var RemovePunctuationOption = TextOptionFuncWithID{
	Fn: removePunctuation,
	ID: 2,
}

var RemoveDuplicateLettersOption = TextOptionFuncWithID{
	Fn: removeDuplicateLetters,
	ID: 3,
}

func (m *MessageText) Text(opts ...TextOptionFuncWithID) string {
	if m.cacheText == nil {
		m.cacheText = make(map[uint64]string)
	}

	var key uint64
	for _, opt := range opts {
		key = key*31 + opt.ID
	}

	if val, ok := m.cacheText[key]; ok {
		return val
	}

	result := m.Original
	isRemovePunctuation := false
	for _, opt := range opts {
		if opt.ID == 2 {
			isRemovePunctuation = true
		}
		result = opt.Fn(result)
	}

	if !isRemovePunctuation {
		result = removeInvisibleRune(result)
	}

	m.cacheText[key] = result
	return result
}

func (m *MessageText) Words(opts ...TextOptionFuncWithID) []string {
	if m.cacheWords == nil {
		m.cacheWords = make(map[uint64][]string)
	}

	var key uint64
	for _, opt := range opts {
		key = key*31 + opt.ID
	}

	if val, ok := m.cacheWords[key]; ok {
		return val
	}

	result := strings.Fields(m.Text(opts...))
	m.cacheWords[key] = result
	return result
}

func HasDoubleLetters(s string) bool {
	var prev rune
	for _, r := range s {
		if prev == r {
			return true
		}
		prev = r
	}
	return false
}

func HasSpecialSymbols(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != ' ' {
			return true
		}
	}
	return false
}
