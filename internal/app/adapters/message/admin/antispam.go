package admin

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"regexp"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/ports"
)

type PauseAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *PauseAntispam) Execute(_ *config.Config, _ string, msg *message.ChatMessage) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(msg.Message.Text.Text()) // !am as <значение>
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

func (a *OnOffAntispam) Execute(cfg *config.Config, channel string, _ *message.ChatMessage) *ports.AnswerType {
	targetMap := map[string]*bool{
		"vip":     &cfg.Channels[channel].Spam.SettingsVIP.Enabled,
		"emote":   &cfg.Channels[channel].Spam.SettingsEmotes.Enabled,
		"default": &cfg.Channels[channel].Spam.SettingsDefault.Enabled,
	}

	if target, ok := targetMap[a.typeSpam]; ok {
		*target = a.enabled

		metrics.AntiSpamEnabled.With(prometheus.Labels{"channel": channel, "type": a.typeSpam}).Set(map[bool]float64{true: 1, false: 0}[*target])
		a.template.SpamPause().Pause(0)
		return success
	}

	return notFoundCmd
}

type InfoAntispam struct {
	template ports.TemplatePort
	fs       ports.FileServerPort
}

func (a *InfoAntispam) Execute(cfg *config.Config, channel string, _ *message.ChatMessage) *ports.AnswerType {
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
	switch cfg.Channels[channel].Spam.Mode {
	case config.OnlineMode:
		mode = "только в онлайне"
	case config.OfflineMode:
		mode = "только в оффлайне"
	default:
		mode = "всегда"
	}

	parts := []string{
		"- режим: " + mode,
		"\nобщие:",
		"- включен: " + strconv.FormatBool(cfg.Channels[channel].Spam.SettingsDefault.Enabled),
		"- порог схожести сообщений: " + fmt.Sprint(cfg.Channels[channel].Spam.SettingsDefault.SimilarityThreshold),
		"- кол-во похожих сообщений: " + strconv.Itoa(cfg.Channels[channel].Spam.SettingsDefault.MessageLimit),
		"- наказания: " + strings.Join(a.template.Punishment().FormatAll(cfg.Channels[channel].Spam.SettingsDefault.Punishments), ", "),
		"- время сброса счётчика наказаний: " + strconv.Itoa(cfg.Channels[channel].Spam.SettingsDefault.DurationResetPunishments),
		"- ограничение максимальной длины слова: " + strconv.Itoa(cfg.Channels[channel].Spam.SettingsDefault.MaxWordLength),
		"- наказание за превышение длины слова: " + a.template.Punishment().Format(cfg.Channels[channel].Spam.SettingsDefault.MaxWordPunishment),
		"- минимальное количество разных сообщений между спамом: " + strconv.Itoa(cfg.Channels[channel].Spam.SettingsDefault.MinGapMessages),
		"\nвиперы:",
		"- включен: " + strconv.FormatBool(cfg.Channels[channel].Spam.SettingsVIP.Enabled),
		"- порог схожести сообщений: " + fmt.Sprint(cfg.Channels[channel].Spam.SettingsVIP.SimilarityThreshold),
		"- кол-во похожих сообщений: " + strconv.Itoa(cfg.Channels[channel].Spam.SettingsVIP.MessageLimit),
		"- наказания: " + strings.Join(a.template.Punishment().FormatAll(cfg.Channels[channel].Spam.SettingsVIP.Punishments), ", "),
		"- время сброса счётчика наказаний: " + strconv.Itoa(cfg.Channels[channel].Spam.SettingsVIP.DurationResetPunishments),
		"- ограничение максимальной длины слова: " + strconv.Itoa(cfg.Channels[channel].Spam.SettingsVIP.MaxWordLength),
		"- наказание за превышение длины слова: " + a.template.Punishment().Format(cfg.Channels[channel].Spam.SettingsVIP.MaxWordPunishment),
		"- минимальное количество разных сообщений между спамом: " + strconv.Itoa(cfg.Channels[channel].Spam.SettingsVIP.MinGapMessages),
		"\nэмоуты:",
		"- включен: " + strconv.FormatBool(cfg.Channels[channel].Spam.SettingsEmotes.Enabled),
		"- кол-во похожих сообщений: " + strconv.Itoa(cfg.Channels[channel].Spam.SettingsEmotes.MessageLimit),
		"- наказания: " + strings.Join(a.template.Punishment().FormatAll(cfg.Channels[channel].Spam.SettingsEmotes.Punishments), ", "),
		"- время сброса счётчика наказаний: " + strconv.Itoa(cfg.Channels[channel].Spam.SettingsEmotes.DurationResetPunishments),
		"- ограничение количества эмоутов в сообщении: " + strconv.Itoa(cfg.Channels[channel].Spam.SettingsEmotes.MaxEmotesLength),
		"- наказание за превышение количества эмоутов в сообщении: " + a.template.Punishment().Format(cfg.Channels[channel].Spam.SettingsEmotes.MaxEmotesPunishment),
		"\nисключения:",
		formatExceptions(cfg.Channels[channel].Spam.Exceptions),
		"\nисключения эмоутов:",
		formatExceptions(cfg.Channels[channel].Spam.SettingsEmotes.Exceptions),
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

func (a *ModeAntispam) Execute(cfg *config.Config, channel string, _ *message.ChatMessage) *ports.AnswerType {
	cfg.Channels[channel].Spam.Mode = a.mode
	return success
}

type SimAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	messages ports.StorePort[storage.Message]
	typeSpam string
}

func (a *SimAntispam) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	target := &cfg.Channels[channel].Spam.SettingsDefault.SimilarityThreshold
	if a.typeSpam == "vip" {
		target = &cfg.Channels[channel].Spam.SettingsVIP.SimilarityThreshold
	}

	matches := a.re.FindStringSubmatch(msg.Message.Text.Text()) // !am sim <значение> или !am vip sim <значение>
	if len(matches) != 2 {
		return nonParametr
	}

	return applyParsedValue[float64](cfg, a.template, a.messages, channel, target, strings.TrimSpace(matches[1]), 0.1, 1, "значение порога схожести сообщений должно быть от 0.1 до 1.0!")
}

type MsgAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	messages ports.StorePort[storage.Message]
	typeSpam string
}

func (a *MsgAntispam) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	target := &cfg.Channels[channel].Spam.SettingsDefault.MessageLimit
	switch a.typeSpam {
	case "vip":
		target = &cfg.Channels[channel].Spam.SettingsVIP.MessageLimit
	case "emote":
		target = &cfg.Channels[channel].Spam.SettingsEmotes.MessageLimit
	}

	matches := a.re.FindStringSubmatch(msg.Message.Text.Text()) // !am msg <значение> или !am vip/emote msg <значение>
	if len(matches) != 2 {
		return nonParametr
	}

	return applyParsedValue[int](cfg, a.template, a.messages, channel, target, strings.TrimSpace(matches[1]), 2, 15, invalidMessageLimitValue.Text[0])
}

type PunishmentsAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	typeSpam string
}

func (a *PunishmentsAntispam) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	target := &cfg.Channels[channel].Spam.SettingsDefault.Punishments
	switch a.typeSpam {
	case "vip":
		target = &cfg.Channels[channel].Spam.SettingsVIP.Punishments
	case "emote":
		target = &cfg.Channels[channel].Spam.SettingsEmotes.Punishments
	}

	matches := a.re.FindStringSubmatch(msg.Message.Text.Text()) // !am p <наказания через запятую> или !am vip/emote p <наказания через запятую>
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
				punishments = cfg.Channels[channel].Spam.SettingsDefault.Punishments
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

func (a *ResetPunishmentsAntispam) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	target := &cfg.Channels[channel].Spam.SettingsDefault.DurationResetPunishments
	switch a.typeSpam {
	case "vip":
		target = &cfg.Channels[channel].Spam.SettingsVIP.DurationResetPunishments
	case "emote":
		target = &cfg.Channels[channel].Spam.SettingsEmotes.DurationResetPunishments
	}

	matches := a.re.FindStringSubmatch(msg.Message.Text.Text()) // !am rp <значение> или !am vip/emote rp <значение>
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

func (a *MaxLenAntispam) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	params := map[string]struct {
		target *int
		max    int
		errMsg string
	}{
		"vip":     {&cfg.Channels[channel].Spam.SettingsVIP.MaxWordLength, 500, "значение максимальной длины слова должно быть от 0 до 500!"},
		"emote":   {&cfg.Channels[channel].Spam.SettingsEmotes.MaxEmotesLength, 50, "значение максимального количества эмоутов должно быть от 0 до 30!"},
		"default": {&cfg.Channels[channel].Spam.SettingsDefault.MaxWordLength, 500, "значение максимальной длины слова должно быть от 0 до 500!"},
	}

	if param, ok := params[a.typeSpam]; ok {
		matches := a.re.FindStringSubmatch(msg.Message.Text.Text()) // !am mlen <значение> или !am vip/emote mlen <значение>
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

func (a *MaxPunishmentAntispam) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	target := &cfg.Channels[channel].Spam.SettingsDefault.MaxWordPunishment
	switch a.typeSpam {
	case "vip":
		target = &cfg.Channels[channel].Spam.SettingsVIP.MaxWordPunishment
	case "emote":
		target = &cfg.Channels[channel].Spam.SettingsEmotes.MaxEmotesPunishment
	}

	matches := a.re.FindStringSubmatch(msg.Message.Text.Text()) // !am mp <наказание> или !am vip/emote mp <наказание>
	if len(matches) != 2 {
		return nonParametr
	}

	p, err := a.template.Punishment().Parse(strings.TrimSpace(matches[1]), true)
	if err != nil {
		return errorPunishmentParse
	}

	if p.Action == "inherit" {
		defaults := map[string]config.Punishment{
			"default": cfg.Channels[channel].Spam.SettingsDefault.Punishments[0],
			"vip":     cfg.Channels[channel].Spam.SettingsVIP.Punishments[0],
			"emote":   cfg.Channels[channel].Spam.SettingsEmotes.Punishments[0],
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

func (a *MinGapAntispam) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	target := &cfg.Channels[channel].Spam.SettingsDefault.MinGapMessages
	if a.typeSpam == "vip" {
		target = &cfg.Channels[channel].Spam.SettingsVIP.MinGapMessages
	}

	matches := a.re.FindStringSubmatch(msg.Message.Text.Text()) // !am mg <значение> или !am vip mg <значение>
	if len(matches) != 2 {
		return nonParametr
	}

	return applyParsedValue[int](cfg, a.template, a.messages, channel, target, strings.TrimSpace(matches[1]), 0, 15, "значение должно быть от 0 до 15!")
}

func applyParsedValue[T int | float64](
	cfg *config.Config, template ports.TemplatePort, messages ports.StorePort[storage.Message],
	channel string, target *T, rawValue string, minValue, maxValue T, errMsg string,
) *ports.AnswerType {
	var val T
	var ok bool

	switch any(*target).(type) {
	case int:
		var v int
		v, ok = template.Parser().ParseIntArg(strings.TrimSpace(rawValue), int(minValue), int(maxValue))
		val = any(v).(T)
	case float64:
		var v float64
		v, ok = template.Parser().ParseFloatArg(strings.TrimSpace(rawValue), float64(minValue), float64(maxValue))
		val = any(v).(T)
	default:
		return &ports.AnswerType{
			Text:    []string{"неподдерживаемый тип"},
			IsReply: true,
		}
	}

	if !ok {
		return &ports.AnswerType{
			Text:    []string{errMsg},
			IsReply: true,
		}
	}
	*target = val

	capacity := calculateCapacity(cfg, channel)
	if capacity > 50 {
		messages.SetCapacity(capacity)
	}

	return success
}

func calculateCapacity(cfg *config.Config, channel string) int32 {
	defLimit := float64(cfg.Channels[channel].Spam.SettingsDefault.MessageLimit*cfg.Channels[channel].Spam.SettingsDefault.MinGapMessages) / cfg.Channels[channel].Spam.SettingsDefault.SimilarityThreshold
	vipLimit := float64(cfg.Channels[channel].Spam.SettingsVIP.MessageLimit*cfg.Channels[channel].Spam.SettingsVIP.MinGapMessages) / cfg.Channels[channel].Spam.SettingsVIP.SimilarityThreshold
	emoteLimit := float64(cfg.Channels[channel].Spam.SettingsEmotes.MessageLimit) / cfg.Channels[channel].Spam.SettingsEmotes.EmoteThreshold
	return int32(max(defLimit, vipLimit, emoteLimit))
}
