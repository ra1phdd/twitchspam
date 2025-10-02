package domain

import (
	"strings"
	"unicode"
	"unsafe"
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
}

type MessageText struct {
	Original string

	cacheText  map[uintptr]string
	cacheWords map[uintptr][]string
}

func (m *MessageText) ReplaceOriginal(text string) {
	m.Original = text
	m.cacheText = make(map[uintptr]string)
	m.cacheWords = make(map[uintptr][]string)
}

type TextOptionFunc func(string) string

func Lower(s string) string {
	return strings.ToLower(s)
}

func RemovePunctuation(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	atWordStart := true

	for _, r := range s {
		if atWordStart {
			if r == '!' {
				b.WriteRune(r)
				atWordStart = false
				continue
			}
			atWordStart = false
		}

		if unicode.IsSpace(r) {
			b.WriteRune(r)
			atWordStart = true
		} else if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
		}
	}

	return b.String()
}

func RemoveDuplicateLetters(s string) string {
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

func (m *MessageText) Text(opts ...TextOptionFunc) string {
	if m.cacheText == nil {
		m.cacheText = make(map[uintptr]string)
	}

	var key uintptr
	for _, opt := range opts {
		key ^= uintptr(unsafe.Pointer(&opt))
	}

	if val, ok := m.cacheText[key]; ok {
		return val
	}

	result := m.Original
	for _, opt := range opts {
		result = opt(result)
	}

	m.cacheText[key] = result
	return result
}

func (m *MessageText) Words(opts ...TextOptionFunc) []string {
	if m.cacheWords == nil {
		m.cacheWords = make(map[uintptr][]string)
	}

	var key uintptr
	for _, opt := range opts {
		key ^= uintptr(unsafe.Pointer(&opt))
	}

	if val, ok := m.cacheWords[key]; ok {
		return val
	}

	result := strings.Fields(m.Text(opts...))
	m.cacheWords[key] = result
	return result
}
