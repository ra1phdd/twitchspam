package admin

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type AddMarker struct {
	re       *regexp.Regexp
	log      logger.Logger
	stream   ports.StreamPort
	api      ports.APIPort
	username string
}

func (m *AddMarker) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return m.handleMarkersAdd(cfg, channel, text)
}

func (m *AddMarker) SetUsername(username string) {
	m.username = username
}

func (m *AddMarker) handleMarkersAdd(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	if !m.stream.IsLive() {
		return streamOff
	}

	// !am mark <имя маркера> или !am mark add <имя маркера>
	matches := m.re.FindStringSubmatch(text.Text())
	if len(matches) != 2 {
		return nonParametr
	}

	userKey := m.username + "_" + m.stream.ChannelID()
	if _, ok := cfg.Channels[channel].Markers[userKey]; !ok {
		cfg.Channels[channel].Markers[userKey] = make(map[string][]*config.Markers)
	}

	s, err := m.api.GetLiveStreams([]string{m.stream.ChannelID()})
	if err != nil {
		m.log.Error("Failed to get live stream", err, slog.String("channelID", m.stream.ChannelID()))
		return unknownError
	}

	if len(s) == 0 {
		return streamOff
	}

	marker := &config.Markers{
		StreamID:  s[0].ID,
		CreatedAt: time.Now(),
		Timecode:  time.Since(s[0].StartedAt),
	}

	markerName := strings.TrimSpace(matches[1])
	cfg.Channels[channel].Markers[userKey][markerName] = append(cfg.Channels[channel].Markers[userKey][markerName], marker)
	return success
}

type ClearMarker struct {
	re       *regexp.Regexp
	stream   ports.StreamPort
	username string
}

func (m *ClearMarker) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return m.handleMarkersClear(cfg, channel, text)
}

func (m *ClearMarker) SetUsername(username string) {
	m.username = username
}

func (m *ClearMarker) handleMarkersClear(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	matches := m.re.FindStringSubmatch(text.Text()) // !am mark clear <имя маркера> или !am mark clear
	if len(matches) != 2 {
		return nonParametr
	}

	userKey := m.username + "_" + m.stream.ChannelID()
	if matches[1] == "" {
		delete(cfg.Channels[channel].Markers, userKey)
		return success
	}

	userMarkers, ok := cfg.Channels[channel].Markers[userKey]
	if !ok {
		return &ports.AnswerType{
			Text:    []string{"маркер не найден!"},
			IsReply: true,
		}
	}

	delete(userMarkers, strings.TrimSpace(matches[1]))
	return success
}

type ListMarker struct {
	re       *regexp.Regexp
	stream   ports.StreamPort
	api      ports.APIPort
	fs       ports.FileServerPort
	username string
}

func (m *ListMarker) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return m.handleMarkersList(cfg, channel, text)
}

func (m *ListMarker) SetUsername(username string) {
	m.username = username
}

func (m *ListMarker) handleMarkersList(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	userMarkers, ok := cfg.Channels[channel].Markers[m.username+"_"+m.stream.ChannelID()]
	if !ok || len(userMarkers) == 0 {
		return &ports.AnswerType{
			Text:    []string{"маркеры не найдены!"},
			IsReply: true,
		}
	}

	formatMarker := func(m *config.Markers, vods map[string]string) string {
		timecode := fmt.Sprintf("%02dh%02dm%02ds",
			int(m.Timecode.Hours()),
			int(m.Timecode.Minutes())%60,
			int(m.Timecode.Seconds())%60,
		)

		if vod, ok := vods[m.StreamID]; ok {
			return fmt.Sprintf("%s - %s?t=%s", m.CreatedAt.Format("02.01"), vod, timecode)
		}
		return fmt.Sprintf("%s - вод не найден (id стрима %s, таймкод %s)", m.CreatedAt.Format("02.01"), m.StreamID, timecode)
	}

	var parts []string
	processMarkers := func(name string, markers []*config.Markers) error {
		vods, err := m.api.GetUrlVOD(m.stream.ChannelID(), markers)
		if err != nil {
			return err
		}
		parts = append(parts, name+":")
		for _, m := range markers {
			parts = append(parts, formatMarker(m, vods))
		}
		return nil
	}

	matches := m.re.FindStringSubmatch(text.Text()) // !am mark list <имя маркера> или !am mark list
	if len(matches) < 1 || len(matches) > 2 {
		return nonParametr
	}

	if len(matches) == 2 {
		name := strings.TrimSpace(matches[1])
		markers, ok := userMarkers[name]
		if !ok || len(markers) == 0 {
			return &ports.AnswerType{Text: []string{"маркеры не найдены!"}, IsReply: true}
		}

		if err := processMarkers(name, markers); err != nil {
			return unknownError
		}
	} else {
		for name, markers := range userMarkers {
			if err := processMarkers(name, markers); err != nil {
				return unknownError
			}
			parts = append(parts, "")
		}
	}

	msg := strings.Join(parts, "\n")
	key, err := m.fs.UploadToHaste(msg)
	if err != nil {
		return unknownError
	}

	return &ports.AnswerType{
		Text:    []string{m.fs.GetURL(key)},
		IsReply: true,
	}
}
