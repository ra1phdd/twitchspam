package message

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
	Text      Text
	EmoteOnly bool     // если Fragments type == "text" отсутствует
	Emotes    []string // text в Fragments при type == "emote"
	IsFirst   func() bool
}

type Text struct {
	Original string

	cacheText  map[uint64]string
	cacheWords map[uint64][]string
}

func (t *Text) ReplaceOriginal(text string) {
	t.Original = strings.TrimSpace(text)
	t.cacheText = make(map[uint64]string)
	t.cacheWords = make(map[uint64][]string)
}

type TextOption uint64

const (
	LowerOption TextOption = iota + 1
	RemovePunctuationOption
	RemoveDuplicateLettersOption
	NormalizeCommaSpacesOption
)

func (t *Text) Text(opts ...TextOption) string {
	if t.cacheText == nil {
		t.cacheText = make(map[uint64]string)
	}

	key := computeCacheKey(opts)
	if val, ok := t.cacheText[key]; ok {
		return val
	}

	hashOpts := parseOptions(opts)
	result := normalizeText(t.Original, hashOpts)

	t.cacheText[key] = result
	return result
}

func (t *Text) Words(opts ...TextOption) []string {
	if t.cacheWords == nil {
		t.cacheWords = make(map[uint64][]string)
	}

	key := computeCacheKey(opts)
	if val, ok := t.cacheWords[key]; ok {
		return val
	}

	result := strings.Fields(t.Text(opts...))
	t.cacheWords[key] = result
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

func computeCacheKey(opts []TextOption) uint64 {
	var key uint64
	for _, opt := range opts {
		key = key*31 + uint64(opt)
	}
	return key
}

func parseOptions(opts []TextOption) map[TextOption]bool {
	hashOpts := make(map[TextOption]bool)
	for _, opt := range opts {
		hashOpts[opt] = true
	}
	return hashOpts
}

func normalizeText(original string, hashOpts map[TextOption]bool) string {
	if hashOpts[NormalizeCommaSpacesOption] {
		original = strings.TrimSpace(original)
		original = strings.ReplaceAll(original, " ,", ",")
		original = strings.ReplaceAll(original, ", ", ",")
	}

	var b strings.Builder
	b.Grow(len(original))

	var prev rune
	var lastWasSpace bool
	word := make([]rune, 0, 16)

	flushWord := func() {
		if len(word) == 0 {
			return
		}
		processWord(&b, word, hashOpts, &prev, &lastWasSpace)
		word = word[:0]
	}

	for _, r := range original {
		if isInvisibleRune(r) {
			continue
		}
		if unicode.IsSpace(r) {
			flushWord()
			lastWasSpace = true
			continue
		}
		if len(word) == 0 && r == '!' {
			word = append(word, r)
			continue
		}
		if hashOpts[RemovePunctuationOption] && !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			continue
		}
		word = append(word, r)
	}

	flushWord()
	return b.String()
}

func processWord(b *strings.Builder, word []rune, hashOpts map[TextOption]bool, prev *rune, lastWasSpace *bool) {
	layout := dominantLayout(word)
	for _, r := range word {
		switch layout {
		case "rus":
			if mapped, ok := engToRus[r]; ok {
				r = mapped
			}
		case "eng":
			if mapped, ok := rusToEng[r]; ok {
				r = mapped
			}
		}
		if hashOpts[LowerOption] && unicode.IsUpper(r) {
			r = unicode.ToLower(r)
		}
		if hashOpts[RemoveDuplicateLettersOption] && r == *prev {
			continue
		}
		if *lastWasSpace && b.Len() > 0 {
			b.WriteRune(' ')
			*lastWasSpace = false
		}
		b.WriteRune(r)
		*prev = r
	}
}
