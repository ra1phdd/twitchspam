package aliases

import (
	"strconv"
	"strings"
)

func (a *Aliases) replaceGamePlaceholder(template string) string {
	if _, ok := nonGameCategories[a.stream.Category()]; ok {
		return template
	}

	return strings.ReplaceAll(template, "{game}", a.stream.Category())
}

func (a *Aliases) replaceCategoryPlaceholder(template string) string {
	return strings.ReplaceAll(template, "{category}", a.stream.Category())
}

func (a *Aliases) replaceChannelPlaceholder(template string) string {
	return strings.ReplaceAll(template, "{channel}", a.stream.ChannelName())
}

func (a *Aliases) replaceRandintPlaceholder(template string) string {
	const placeholder = "{randint"
	for {
		start := strings.Index(template, placeholder)
		if start == -1 {
			break
		}
		end := strings.Index(template[start:], "}")
		if end == -1 {
			break
		}
		end += start
		arg := strings.TrimSpace(template[start+len(placeholder) : end])
		replacement := a.randint(arg)
		template = template[:start] + replacement + template[end+1:]
	}
	return template
}

func (a *Aliases) replaceCountdownPlaceholder(template string) string {
	const placeholder = "{countdown"
	for {
		start := strings.Index(template, placeholder)
		if start == -1 {
			break
		}
		end := strings.Index(template[start:], "}")
		if end == -1 {
			break
		}
		end += start
		arg := strings.TrimSpace(template[start+len(placeholder) : end])
		replacement := a.countdown(arg)
		template = template[:start] + replacement + template[end+1:]
	}
	return template
}

func (a *Aliases) replaceCountupPlaceholder(template string) string {
	const placeholder = "{countup"
	for {
		start := strings.Index(template, placeholder)
		if start == -1 {
			break
		}
		end := strings.Index(template[start:], "}")
		if end == -1 {
			break
		}
		end += start
		arg := strings.TrimSpace(template[start+len(placeholder) : end])
		replacement := a.countup(arg)
		template = template[:start] + replacement + template[end+1:]
	}
	return template
}

func (a *Aliases) replaceQueryPlaceholders(template string, queryParts []string) string {
	var sb strings.Builder
	nextIdx := 0
	last := 0

	for _, loc := range queryRe.FindAllStringSubmatchIndex(template, -1) {
		sb.WriteString(template[last:loc[0]])
		matchNum := template[loc[2]:loc[3]]
		if matchNum == "" {
			if nextIdx < len(queryParts) {
				sb.WriteString(queryParts[nextIdx])
				nextIdx++
			}
		} else {
			i, _ := strconv.Atoi(matchNum)
			if i >= 1 && i <= len(queryParts) {
				sb.WriteString(queryParts[i-1])
			}
		}
		last = loc[1]
	}

	sb.WriteString(template[last:])
	if nextIdx < len(queryParts) {
		sb.WriteByte(' ')
		for k := nextIdx; k < len(queryParts); k++ {
			sb.WriteString(queryParts[k])
			if k+1 < len(queryParts) {
				sb.WriteByte(' ')
			}
		}
	}

	return sb.String()
}
