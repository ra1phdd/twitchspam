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
	t.Original = text
	t.cacheText = make(map[uint64]string)
	t.cacheWords = make(map[uint64][]string)
}

func lower(s string) string {
	return strings.ToLower(s)
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

func (t *Text) Text(opts ...TextOptionFuncWithID) string {
	if t.cacheText == nil {
		t.cacheText = make(map[uint64]string)
	}

	var key uint64
	for _, opt := range opts {
		key = key*31 + opt.ID
	}

	if val, ok := t.cacheText[key]; ok {
		return val
	}

	result := t.Original
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
	result = removeHomoglyphs(result)

	t.cacheText[key] = result
	return result
}

func (t *Text) Words(opts ...TextOptionFuncWithID) []string {
	if t.cacheWords == nil {
		t.cacheWords = make(map[uint64][]string)
	}

	var key uint64
	for _, opt := range opts {
		key = key*31 + opt.ID
	}

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
