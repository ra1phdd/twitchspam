package admin

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/platform/twitch/api"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type CreatePoll struct {
	re       *regexp.Regexp
	stream   ports.StreamPort
	api      ports.APIPort
	template ports.TemplatePort
	poll     *ports.Poll
}

func (p *CreatePoll) Execute(_ *config.Config, text *domain.MessageText) *ports.AnswerType {
	return p.handleCreatePoll(text)
}

func (p *CreatePoll) handleCreatePoll(text *domain.MessageText) *ports.AnswerType {
	if p.poll != nil && p.poll.Status == "ACTIVE" {
		return &ports.AnswerType{
			Text:    []string{"перед открытием опроса нужно закрыть предыдущий!"},
			IsReply: true,
		}
	}

	matches := p.re.FindStringSubmatch(text.Text()) // !am poll <*длительность> <*кол-во баллов> <заголовок> / <вариант> / <вариант> / <вариант> / ... (до 5 вариантов)
	if len(matches) != 4 {
		return incorrectSyntax
	}

	dur := 60
	if strings.TrimSpace(matches[1]) != "" {
		if val, ok := p.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 30, 1800); ok {
			dur = val
		} else {
			return &ports.AnswerType{
				Text:    []string{"длительность прогноза должна быть от 30 до 1800 секунд!"},
				IsReply: true,
			}
		}
	}

	enablePoints, pointPerVote := false, 0
	if strings.TrimSpace(matches[2]) != "" {
		if val, ok := p.template.Parser().ParseIntArg(strings.TrimSpace(matches[2]), 1, 1000000); ok {
			enablePoints = true
			pointPerVote = val
		}
	}

	title := strings.TrimSpace(matches[3])
	if len(title) > 60 {
		return &ports.AnswerType{
			Text:    []string{"заголовок прогноза не может быть больше 60 символов!"},
			IsReply: true,
		}
	}

	choices := strings.Split(strings.TrimSpace(matches[4]), "/")
	for i := range choices {
		choices[i] = strings.TrimSpace(choices[i])
		if len(choices[i]) > 25 {
			return &ports.AnswerType{
				Text:    []string{"заголовок исхода не может быть больше 25 символов!"},
				IsReply: true,
			}
		}
	}

	if len(choices) > 5 {
		choices = choices[:5]
	}

	if len(choices) < 2 {
		choices = choices[:0]
		choices = append(choices, "да", "нет")
	}

	poll, err := p.api.CreatePoll(p.stream.ChannelID(), title, choices, dur, enablePoints, pointPerVote)
	if err != nil {
		if errors.Is(err, api.ErrUserAuthNotCompleted) {
			return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
		}
		if errors.Is(err, api.ErrBadRequest) {
			return &ports.AnswerType{Text: []string{"некорректный запрос!"}, IsReply: true}
		}
		return unknownError
	}
	p.poll = poll

	return success
}

type EndPoll struct {
	re       *regexp.Regexp
	stream   ports.StreamPort
	api      ports.APIPort
	template ports.TemplatePort
	poll     *ports.Poll
}

func (p *EndPoll) Execute(_ *config.Config, text *domain.MessageText) *ports.AnswerType {
	return p.handleEndPoll(text)
}

func (p *EndPoll) handleEndPoll(text *domain.MessageText) *ports.AnswerType {
	if p.poll == nil || p.poll.ID == "" {
		return &ports.AnswerType{
			Text:    []string{"опрос не найден!"},
			IsReply: true,
		}
	}

	// !am pred del/end
	matches := p.re.FindStringSubmatch(text.Text())
	if len(matches) != 2 {
		return incorrectSyntax
	}

	var status string
	switch strings.TrimSpace(matches[1]) {
	case "end":
		status = "TERMINATED"
	case "del":
		status = "ARCHIVED"
	default:
		return notFoundCmd
	}

	err := p.api.EndPoll(p.stream.ChannelID(), p.poll.ID, status)
	if err != nil {
		if errors.Is(err, api.ErrUserAuthNotCompleted) {
			return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
		}
		if errors.Is(err, api.ErrBadRequest) {
			return &ports.AnswerType{Text: []string{"некорректный запрос!"}, IsReply: true}
		}
		return unknownError
	}

	p.poll.EndedAt = time.Now()
	return success
}

type RePoll struct {
	stream   ports.StreamPort
	api      ports.APIPort
	template ports.TemplatePort
	poll     *ports.Poll
}

func (p *RePoll) Execute(_ *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return p.handleRePoll()
}

func (p *RePoll) handleRePoll() *ports.AnswerType {
	if p.poll == nil || p.poll.ID == "" {
		return &ports.AnswerType{
			Text:    []string{"предыдущий опрос не найден!"},
			IsReply: true,
		}
	}

	if p.poll.Status == "ACTIVE" {
		return &ports.AnswerType{
			Text:    []string{"перед открытием опроса нужно закрыть предыдущий!"},
			IsReply: true,
		}
	}

	choices := make([]string, 0, len(p.poll.Choices))
	for _, choice := range p.poll.Choices {
		choices = append(choices, choice.ID)
	}

	poll, err := p.api.CreatePoll(p.stream.ChannelID(), p.poll.Title, choices, p.poll.Duration, p.poll.ChannelPointsVotingEnabled, p.poll.ChannelPointsPerVote)
	if err != nil {
		if errors.Is(err, api.ErrUserAuthNotCompleted) {
			return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
		}
		if errors.Is(err, api.ErrBadRequest) {
			return &ports.AnswerType{Text: []string{"некорректный запрос!"}, IsReply: true}
		}
		return unknownError
	}
	p.poll = poll

	return success
}
