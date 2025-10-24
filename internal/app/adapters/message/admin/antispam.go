package admin

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"regexp"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/ports"
)

type PauseAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *PauseAntispam) Execute(_ *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleAntiSpamPause(text)
}

func (a *PauseAntispam) handleAntiSpamPause(text *domain.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am as <значение>
	if len(matches) != 2 {
		return nonParametr
	}

	if val, ok := a.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 0, 3600); ok {
		a.template.SpamPause().Pause(time.Duration(val) * time.Second)
		return success
	}

	return &ports.AnswerType{
		Text:    []string{"значение паузы должно быть от 1 до 3600!"},
		IsReply: true,
	}
}

type OnOffAntispam struct {
	enabled  bool
	typeSpam string
	template ports.TemplatePort
}

func (a *OnOffAntispam) Execute(cfg *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return a.handleAntiSpamOnOff(cfg)
}

func (a *OnOffAntispam) handleAntiSpamOnOff(cfg *config.Config) *ports.AnswerType {
	targetMap := map[string]*bool{
		"vip":     &cfg.Spam.SettingsVIP.Enabled,
		"emote":   &cfg.Spam.SettingsEmotes.Enabled,
		"default": &cfg.Spam.SettingsDefault.Enabled,
	}

	if target, ok := targetMap[a.typeSpam]; ok {
		*target = a.enabled

		metrics.AntiSpamEnabled.With(prometheus.Labels{"type": a.typeSpam}).Set(map[bool]float64{true: 1, false: 0}[*target])
		a.template.SpamPause().Pause(0)
		return success
	}

	return notFoundCmd
}

type InfoAntispam struct {
	template ports.TemplatePort
	fs       ports.FileServerPort
}

func (a *InfoAntispam) Execute(cfg *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return a.handleAntiSpamInfo(cfg)
}

func (a *InfoAntispam) handleAntiSpamInfo(cfg *config.Config) *ports.AnswerType {
	whitelistUsers := "не найдены"
	if len(cfg.Spam.WhitelistUsers) > 0 {
		var sb strings.Builder
		first := true
		for user := range cfg.Spam.WhitelistUsers {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(user)
			first = false
		}
		whitelistUsers = sb.String()
	}

	formatExceptions := func(exceptions map[string]*config.ExceptionsSettings) string {
		if len(exceptions) == 0 {
			return "не найдены"
		}

		var sb strings.Builder
		for word, ex := range exceptions {
			if ex.Regexp != nil {
				sb.WriteString(fmt.Sprintf("- %s (название исключения: %s, включено: %v, лимит сообщений: %d, наказания: %s, опции: %s)\n",
					ex.Regexp.String(), word, ex.Enabled, ex.MessageLimit,
					strings.Join(a.template.Punishment().FormatAll(ex.Punishments), ", "),
					a.template.Options().ExceptToString(ex.Options),
				))
				continue
			}

			sb.WriteString(fmt.Sprintf("- %s (включено: %v, лимит сообщений: %d, наказания: %s, опции: %s)\n",
				word, ex.Enabled, ex.MessageLimit,
				strings.Join(a.template.Punishment().FormatAll(ex.Punishments), ", "),
				a.template.Options().ExceptToString(ex.Options),
			))
		}
		return sb.String()
	}

	var mode string
	switch cfg.Spam.Mode {
	case config.OnlineMode:
		mode = "только в онлайне"
	case config.OfflineMode:
		mode = "только в оффлайне"
	default:
		mode = "всегда"
	}

	parts := []string{
		"- режим: " + mode,
		"- разрешенные пользователи: " + whitelistUsers,
		"\nобщие:",
		"- включен: " + strconv.FormatBool(cfg.Spam.SettingsDefault.Enabled),
		"- порог схожести сообщений: " + fmt.Sprint(cfg.Spam.SettingsDefault.SimilarityThreshold),
		"- кол-во похожих сообщений: " + strconv.Itoa(cfg.Spam.SettingsDefault.MessageLimit),
		"- наказания: " + strings.Join(a.template.Punishment().FormatAll(cfg.Spam.SettingsDefault.Punishments), ", "),
		"- время сброса счётчика наказаний: " + strconv.Itoa(cfg.Spam.SettingsDefault.DurationResetPunishments),
		"- ограничение максимальной длины слова: " + strconv.Itoa(cfg.Spam.SettingsDefault.MaxWordLength),
		"- наказание за превышение длины слова: " + a.template.Punishment().Format(cfg.Spam.SettingsDefault.MaxWordPunishment),
		"- минимальное количество разных сообщений между спамом: " + strconv.Itoa(cfg.Spam.SettingsDefault.MinGapMessages),
		"\nвиперы:",
		"- включен: " + strconv.FormatBool(cfg.Spam.SettingsVIP.Enabled),
		"- порог схожести сообщений: " + fmt.Sprint(cfg.Spam.SettingsVIP.SimilarityThreshold),
		"- кол-во похожих сообщений: " + strconv.Itoa(cfg.Spam.SettingsVIP.MessageLimit),
		"- наказания: " + strings.Join(a.template.Punishment().FormatAll(cfg.Spam.SettingsVIP.Punishments), ", "),
		"- время сброса счётчика наказаний: " + strconv.Itoa(cfg.Spam.SettingsVIP.DurationResetPunishments),
		"- ограничение максимальной длины слова: " + strconv.Itoa(cfg.Spam.SettingsVIP.MaxWordLength),
		"- наказание за превышение длины слова: " + a.template.Punishment().Format(cfg.Spam.SettingsVIP.MaxWordPunishment),
		"- минимальное количество разных сообщений между спамом: " + strconv.Itoa(cfg.Spam.SettingsVIP.MinGapMessages),
		"\nэмоуты:",
		"- включен: " + strconv.FormatBool(cfg.Spam.SettingsEmotes.Enabled),
		"- кол-во похожих сообщений: " + strconv.Itoa(cfg.Spam.SettingsEmotes.MessageLimit),
		"- наказания: " + strings.Join(a.template.Punishment().FormatAll(cfg.Spam.SettingsEmotes.Punishments), ", "),
		"- время сброса счётчика наказаний: " + strconv.Itoa(cfg.Spam.SettingsEmotes.DurationResetPunishments),
		"- ограничение количества эмоутов в сообщении: " + strconv.Itoa(cfg.Spam.SettingsEmotes.MaxEmotesLength),
		"- наказание за превышение количества эмоутов в сообщении: " + a.template.Punishment().Format(cfg.Spam.SettingsEmotes.MaxEmotesPunishment),
		"\nисключения:",
		formatExceptions(cfg.Spam.Exceptions),
		"\nисключения эмоутов:",
		formatExceptions(cfg.Spam.SettingsEmotes.Exceptions),
	}
	msg := "настройки:\n" + strings.Join(parts, "\n")

	key, err := a.fs.UploadToHaste(msg)
	if err != nil {
		return unknownError
	}

	return &ports.AnswerType{
		Text:    []string{a.fs.GetURL(key)},
		IsReply: true,
	}
}

type ModeAntispam struct {
	mode int
}

func (a *ModeAntispam) Execute(cfg *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return a.handleAntispamMode(cfg, a.mode)
}

func (a *ModeAntispam) handleAntispamMode(cfg *config.Config, mode int) *ports.AnswerType {
	cfg.Spam.Mode = mode
	return success
}

type SimAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	messages ports.StorePort[storage.Message]
	typeSpam string
}

func (a *SimAntispam) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleSim(cfg, text, a.typeSpam)
}

func (a *SimAntispam) handleSim(cfg *config.Config, text *domain.MessageText, typeSpam string) *ports.AnswerType {
	target := &cfg.Spam.SettingsDefault.SimilarityThreshold
	if typeSpam == "vip" {
		target = &cfg.Spam.SettingsVIP.SimilarityThreshold
	}

	matches := a.re.FindStringSubmatch(text.Text()) // !am sim <значение> или !am vip sim <значение>
	if len(matches) != 2 {
		return nonParametr
	}

	if val, ok := a.template.Parser().ParseFloatArg(strings.TrimSpace(matches[1]), 0.1, 1); ok {
		*target = val

		capacity := func() int32 {
			defLimit := float64(cfg.Spam.SettingsDefault.MessageLimit*cfg.Spam.SettingsDefault.MinGapMessages) / cfg.Spam.SettingsDefault.SimilarityThreshold
			vipLimit := float64(cfg.Spam.SettingsVIP.MessageLimit*cfg.Spam.SettingsVIP.MinGapMessages) / cfg.Spam.SettingsVIP.SimilarityThreshold
			emoteLimit := float64(cfg.Spam.SettingsEmotes.MessageLimit) / cfg.Spam.SettingsEmotes.EmoteThreshold

			return int32(max(defLimit, vipLimit, emoteLimit))
		}()

		if capacity > 50 {
			a.messages.SetCapacity(capacity)
		}
		return success
	}

	return &ports.AnswerType{
		Text:    []string{"значение порога схожести сообщений должно быть от 0.1 до 1.0!"},
		IsReply: true,
	}
}

type MsgAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	messages ports.StorePort[storage.Message]
	typeSpam string
}

func (a *MsgAntispam) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleMsg(cfg, text)
}

func (a *MsgAntispam) handleMsg(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	target := &cfg.Spam.SettingsDefault.MessageLimit
	switch a.typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MessageLimit
	case "emote":
		target = &cfg.Spam.SettingsEmotes.MessageLimit
	}

	matches := a.re.FindStringSubmatch(text.Text()) // !am msg <значение> или !am vip/emote msg <значение>
	if len(matches) != 2 {
		return nonParametr
	}

	if val, ok := a.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 2, 15); ok {
		*target = val

		capacity := func() int32 {
			defLimit := float64(cfg.Spam.SettingsDefault.MessageLimit*cfg.Spam.SettingsDefault.MinGapMessages) / cfg.Spam.SettingsDefault.SimilarityThreshold
			vipLimit := float64(cfg.Spam.SettingsVIP.MessageLimit*cfg.Spam.SettingsVIP.MinGapMessages) / cfg.Spam.SettingsVIP.SimilarityThreshold
			emoteLimit := float64(cfg.Spam.SettingsEmotes.MessageLimit) / cfg.Spam.SettingsEmotes.EmoteThreshold

			return int32(max(defLimit, vipLimit, emoteLimit))
		}()

		if capacity > 50 {
			a.messages.SetCapacity(capacity)
		}
		return success
	}

	return invalidMessageLimitValue
}

type PunishmentsAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	typeSpam string
}

func (a *PunishmentsAntispam) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handlePunishments(cfg, text)
}

func (a *PunishmentsAntispam) handlePunishments(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	target := &cfg.Spam.SettingsDefault.Punishments
	switch a.typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.Punishments
	case "emote":
		target = &cfg.Spam.SettingsEmotes.Punishments
	}

	matches := a.re.FindStringSubmatch(text.Text()) // !am p <наказания через запятую> или !am vip/emote p <наказания через запятую>
	if len(matches) != 2 {
		return nonParametr
	}

	parts := strings.Split(strings.TrimSpace(matches[1]), ",")
	punishments := make([]config.Punishment, 0, len(parts))

	for i, str := range parts {
		if i >= 15 {
			break
		}

		str = strings.TrimSpace(str)
		if str == "" {
			continue
		}

		allowInherit := a.typeSpam != "default"
		p, err := a.template.Punishment().Parse(str, allowInherit)
		if err != nil {
			return errorPunishmentParse
		}

		if p.Action == "inherit" {
			if a.typeSpam != "default" {
				punishments = cfg.Spam.SettingsDefault.Punishments
				break
			}

			return errorPunishmentCopy
		}
		punishments = append(punishments, p)
	}

	if len(punishments) == 0 {
		return nonParametr
	}

	*target = punishments
	return success
}

type ResetPunishmentsAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	typeSpam string
}

func (a *ResetPunishmentsAntispam) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleDurationResetPunishments(cfg, text)
}

func (a *ResetPunishmentsAntispam) handleDurationResetPunishments(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	target := &cfg.Spam.SettingsDefault.DurationResetPunishments
	switch a.typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.DurationResetPunishments
	case "emote":
		target = &cfg.Spam.SettingsEmotes.DurationResetPunishments
	}

	matches := a.re.FindStringSubmatch(text.Text()) // !am rp <значение> или !am vip/emote rp <значение>
	if len(matches) != 2 {
		return nonParametr
	}

	if val, ok := a.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 1, 86400); ok {
		*target = val
		return success
	}

	return &ports.AnswerType{
		Text:    []string{"значение времени сброса наказаний должно быть от 1 до 86400!"},
		IsReply: true,
	}
}

type MaxLenAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	typeSpam string
}

func (a *MaxLenAntispam) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleMaxLen(cfg, text)
}

func (a *MaxLenAntispam) handleMaxLen(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	params := map[string]struct {
		target *int
		max    int
		errMsg string
	}{
		"vip":     {&cfg.Spam.SettingsVIP.MaxWordLength, 500, "значение максимальной длины слова должно быть от 0 до 500!"},
		"emote":   {&cfg.Spam.SettingsEmotes.MaxEmotesLength, 50, "значение максимального количества эмоутов должно быть от 0 до 30!"},
		"default": {&cfg.Spam.SettingsDefault.MaxWordLength, 500, "значение максимальной длины слова должно быть от 0 до 500!"},
	}

	if param, ok := params[a.typeSpam]; ok {
		matches := a.re.FindStringSubmatch(text.Text()) // !am mlen <значение> или !am vip/emote mlen <значение>
		if len(matches) != 2 {
			return nonParametr
		}

		if val, ok := a.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 0, param.max); ok {
			*param.target = val
			return success
		}

		return &ports.AnswerType{
			Text:    []string{param.errMsg},
			IsReply: true,
		}
	}

	return notFoundCmd
}

type MaxPunishmentAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	typeSpam string
}

func (a *MaxPunishmentAntispam) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleMaxPunishment(cfg, text)
}

func (a *MaxPunishmentAntispam) handleMaxPunishment(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	target := &cfg.Spam.SettingsDefault.MaxWordPunishment
	switch a.typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MaxWordPunishment
	case "emote":
		target = &cfg.Spam.SettingsEmotes.MaxEmotesPunishment
	}

	matches := a.re.FindStringSubmatch(text.Text()) // !am mp <наказание> или !am vip/emote mp <наказание>
	if len(matches) != 2 {
		return nonParametr
	}

	p, err := a.template.Punishment().Parse(strings.TrimSpace(matches[1]), true)
	if err != nil {
		return errorPunishmentParse
	}

	if p.Action == "inherit" {
		defaults := map[string]config.Punishment{
			"default": cfg.Spam.SettingsDefault.Punishments[0],
			"vip":     cfg.Spam.SettingsVIP.Punishments[0],
			"emote":   cfg.Spam.SettingsEmotes.Punishments[0],
		}
		if val, ok := defaults[a.typeSpam]; ok {
			p = val
		}
	}

	*target = p
	return success
}

type MinGapAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	messages ports.StorePort[storage.Message]
	typeSpam string
}

func (a *MinGapAntispam) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleMinGap(cfg, text)
}

func (a *MinGapAntispam) handleMinGap(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	target := &cfg.Spam.SettingsDefault.MinGapMessages
	if a.typeSpam == "vip" {
		target = &cfg.Spam.SettingsVIP.MinGapMessages
	}

	matches := a.re.FindStringSubmatch(text.Text()) // !am mg <значение> или !am vip mg <значение>
	if len(matches) != 2 {
		return nonParametr
	}

	if val, ok := a.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 0, 15); ok {
		*target = val

		capacity := func() int32 {
			defLimit := float64(cfg.Spam.SettingsDefault.MessageLimit*cfg.Spam.SettingsDefault.MinGapMessages) / cfg.Spam.SettingsDefault.SimilarityThreshold
			vipLimit := float64(cfg.Spam.SettingsVIP.MessageLimit*cfg.Spam.SettingsVIP.MinGapMessages) / cfg.Spam.SettingsVIP.SimilarityThreshold
			emoteLimit := float64(cfg.Spam.SettingsEmotes.MessageLimit) / cfg.Spam.SettingsEmotes.EmoteThreshold

			return int32(max(defLimit, vipLimit, emoteLimit))
		}()

		if capacity > 50 {
			a.messages.SetCapacity(capacity)
		}
		return success
	}

	return &ports.AnswerType{
		Text:    []string{"значение должно быть от 0 до 15!"},
		IsReply: true,
	}
}

type AddAntispam struct {
	re *regexp.Regexp
}

func (a *AddAntispam) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleAdd(cfg, text)
}

func (a *AddAntispam) handleAdd(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am add <пользователи через запятую>
	if len(matches) != 2 {
		return nonParametr
	}

	words := strings.Split(strings.TrimSpace(matches[1]), ",")
	added, exists := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimPrefix(strings.TrimSpace(word), "@")
		if word == "" {
			continue
		}

		if _, ok := cfg.Spam.WhitelistUsers[word]; ok {
			exists = append(exists, word)
		} else {
			cfg.Spam.WhitelistUsers[word] = struct{}{}
			added = append(added, word)
		}
	}

	return buildResponse("пользователи не указаны", RespArg{Items: added, Name: "добавлены в список"}, RespArg{Items: exists, Name: "уже есть в списке"})
}

type DelAntispam struct {
	re *regexp.Regexp
}

func (a *DelAntispam) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleDel(cfg, text)
}

func (a *DelAntispam) handleDel(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am del <пользователи через запятую>
	if len(matches) != 2 {
		return nonParametr
	}

	words := strings.Split(strings.TrimSpace(matches[1]), ",")
	removed, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimPrefix(strings.TrimSpace(word), "@")
		if word == "" {
			continue
		}

		if _, ok := cfg.Spam.WhitelistUsers[word]; ok {
			delete(cfg.Spam.WhitelistUsers, word)
			removed = append(removed, word)
		} else {
			notFound = append(notFound, word)
		}
	}

	return buildResponse("пользователи не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}
