package irc

import "strings"

func (i *IRC) parseMessage(line string) *Message {
	msg := &Message{}

	if len(line) == 0 || line[0] != '@' {
		return msg
	}

	spaceIdx := strings.IndexByte(line, ' ')
	if spaceIdx == -1 {
		return msg
	}

	rawTags := line[1:spaceIdx]

	parseTag := func(tag string) {
		if tag == "" {
			return
		}
		if eq := strings.IndexByte(tag, '='); eq != -1 {
			k, v := tag[:eq], tag[eq+1:]
			switch k {
			case "id":
				msg.MessageID = v
			case "first-msg":
				msg.IsFirst = v == "1"
			}
		}
	}

	start := 0
	for i := 0; i <= len(rawTags); i++ {
		if i == len(rawTags) || rawTags[i] == ';' {
			parseTag(rawTags[start:i])
			start = i + 1
		}
	}

	return msg
}
