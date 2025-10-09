package irc

import "strings"

func (i *IRC) parseMessage(line string) *Message {
	msg := &Message{}

	if len(line) > 0 && line[0] == '@' {
		spaceIdx := strings.IndexByte(line, ' ')
		if spaceIdx != -1 {
			rawTags := line[1:spaceIdx]

			start := 0
			for i := 0; i <= len(rawTags); i++ {
				if i == len(rawTags) || rawTags[i] == ';' {
					tag := rawTags[start:i]
					if tag != "" {
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
					start = i + 1
				}
			}
		}
	}

	return msg
}
