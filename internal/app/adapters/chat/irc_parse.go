package chat

import (
	"strings"
	"twitchspam/internal/app/ports"
)

func ParseIRC(line string) *ports.IRCMessage {
	msg := &ports.IRCMessage{Tags: make(map[string]string)}

	if len(line) > 0 && line[0] == '@' {
		spaceIdx := strings.IndexByte(line, ' ')
		if spaceIdx != -1 {
			rawTags := line[1:spaceIdx]
			line = line[spaceIdx+1:]

			start := 0
			for i := 0; i <= len(rawTags); i++ {
				if i == len(rawTags) || rawTags[i] == ';' {
					tag := rawTags[start:i]
					if tag != "" {
						if eq := strings.IndexByte(tag, '='); eq != -1 {
							k, v := tag[:eq], tag[eq+1:]
							msg.Tags[k] = v
						} else {
							msg.Tags[tag] = ""
						}
					}
					start = i + 1
				}
			}
		}
	}

	if len(line) > 0 && line[0] == ':' {
		spaceIdx := strings.IndexByte(line, ' ')
		if spaceIdx != -1 {
			prefix := line[1:spaceIdx]
			line = line[spaceIdx+1:]

			if excl := strings.IndexByte(prefix, '!'); excl != -1 {
				msg.Username = prefix[:excl]
			}
		}
	}

	// Извлекаем полезное из тегов
	if v, ok := msg.Tags["badges"]; ok && strings.Contains(v, "broadcaster") {
		msg.IsMod = true
	}
	if v, ok := msg.Tags["mod"]; ok && v == "1" {
		msg.IsMod = true
	}
	if v, ok := msg.Tags["first-msg"]; ok && v == "1" {
		msg.IsFirst = true
	}
	if v, ok := msg.Tags["id"]; ok {
		msg.MessageID = v
	}
	if v, ok := msg.Tags["badges"]; ok && strings.Contains(v, "vip") {
		msg.IsVIP = true
	}
	if v, ok := msg.Tags["subscriber"]; ok && v == "1" {
		msg.IsSubscriber = true
	}
	if v, ok := msg.Tags["user-id"]; ok {
		msg.UserID = v
	}
	if v, ok := msg.Tags["display-name"]; ok {
		msg.Username = v
	}
	if v, ok := msg.Tags["room-id"]; ok {
		msg.RoomID = v
	}

	// Текст сообщения (после " :")
	if idx := strings.Index(line, " :"); idx != -1 {
		msg.Text = line[idx+2:]
	}

	return msg
}
