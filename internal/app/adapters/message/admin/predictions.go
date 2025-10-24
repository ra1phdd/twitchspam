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

type CreatePrediction struct {
	re       *regexp.Regexp
	stream   ports.StreamPort
	api      ports.APIPort
	template ports.TemplatePort
	pred     *ports.Predictions
}

func (p *CreatePrediction) Execute(_ *config.Config, text *domain.MessageText) *ports.AnswerType {
	return p.handleCreatePrediction(text)
}

func (p *CreatePrediction) handleCreatePrediction(text *domain.MessageText) *ports.AnswerType {
	matches := p.re.FindStringSubmatch(text.Text()) // !am pred <длительность> <заголовок> / <исход> / <исход> / <исход> / ... (до 10 вариантов)
	if len(matches) != 4 {
		return incorrectSyntax
	}

	dur := 60
	if strings.TrimSpace(matches[3]) != "" {
		if val, ok := p.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 30, 1800); ok {
			dur = val
		} else {
			return &ports.AnswerType{
				Text:    []string{"длительность прогноза должна быть от 30 до 1800 секунд!"},
				IsReply: true,
			}
		}
	}

	title := strings.TrimSpace(matches[2])
	if len(title) > 45 {
		return &ports.AnswerType{
			Text:    []string{"заголовок прогноза не может быть больше 45 символов!"},
			IsReply: true,
		}
	}

	outcomes := strings.Split(strings.TrimSpace(matches[3]), "/")
	for i := range outcomes {
		outcomes[i] = strings.TrimSpace(outcomes[i])
		if len(outcomes[i]) > 25 {
			return &ports.AnswerType{
				Text:    []string{"заголовок исхода не может быть больше 25 символов!"},
				IsReply: true,
			}
		}
	}

	if len(outcomes) > 10 {
		outcomes = outcomes[:10]
	}

	if len(outcomes) == 0 {
		outcomes = append(outcomes, "да", "нет")
	}

	pred, err := p.api.CreatePrediction(p.stream.ChannelID(), title, outcomes, dur)
	if err != nil {
		if errors.Is(err, api.ErrUserAuthNotCompleted) {
			return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
		}
		if errors.Is(err, api.ErrBadRequest) {
			return &ports.AnswerType{Text: []string{"некорректный запрос!"}, IsReply: true}
		}
		return unknownError
	}
	p.pred = pred

	return success
}

type EndPrediction struct {
	re       *regexp.Regexp
	stream   ports.StreamPort
	api      ports.APIPort
	template ports.TemplatePort
	pred     *ports.Predictions
}

func (p *EndPrediction) Execute(_ *config.Config, text *domain.MessageText) *ports.AnswerType {
	return p.handleEndPrediction(text)
}

func (p *EndPrediction) handleEndPrediction(text *domain.MessageText) *ports.AnswerType {
	if p.pred == nil || p.pred.ID == "" {
		return &ports.AnswerType{
			Text:    []string{"ставка не найдена!"},
			IsReply: true,
		}
	}

	// !am pred end <номер варианта>
	// !am pred del/lock
	matches := p.re.FindStringSubmatch(text.Text())
	if len(matches) < 2 || len(matches) > 3 {
		return incorrectSyntax
	}

	var status, winningOutcomeID string
	switch strings.TrimSpace(matches[1]) {
	case "end":
		if !p.pred.EndedAt.IsZero() {
			return &ports.AnswerType{
				Text:    []string{"ставка уже завершена!"},
				IsReply: true,
			}
		}

		val, ok := p.template.Parser().ParseIntArg(strings.TrimSpace(matches[2]), 1, 10)
		if !ok {
			return &ports.AnswerType{
				Text:    []string{"неверный номер варианта!"},
				IsReply: true,
			}
		}
		status, winningOutcomeID = "RESOLVED", p.pred.Outcomes[val].ID
	case "del":
		status = "CANCELED"
	case "lock":
		if !p.pred.LockedAt.IsZero() {
			return &ports.AnswerType{
				Text:    []string{"ставка уже заблокирована!"},
				IsReply: true,
			}
		}

		status = "LOCKED"
	default:
		return notFoundCmd
	}

	err := p.api.EndPrediction(p.stream.ChannelID(), p.pred.ID, status, winningOutcomeID)
	if err != nil {
		if errors.Is(err, api.ErrUserAuthNotCompleted) {
			return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
		}
		if errors.Is(err, api.ErrBadRequest) {
			return &ports.AnswerType{Text: []string{"некорректный запрос!"}, IsReply: true}
		}
		return unknownError
	}

	switch strings.TrimSpace(matches[1]) {
	case "end":
		p.pred.EndedAt = time.Now()
	case "lock":
		p.pred.LockedAt = time.Now()
	}
	return success
}
