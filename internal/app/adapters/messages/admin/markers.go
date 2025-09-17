package admin

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type AddMarker struct {
	log      logger.Logger
	stream   ports.StreamPort
	api      ports.APIPort
	username string
}

func (m *AddMarker) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMarkersAdd(cfg, text, m.username)
}

func (m *AddMarker) SetUsername(username string) {
	m.username = username
}

func (m *AddMarker) handleMarkersAdd(cfg *config.Config, text *ports.MessageText, username string) *ports.AnswerType {
	if !m.stream.IsLive() {
		return &ports.AnswerType{
			Text:    []string{"стрим выключен!"},
			IsReply: true,
		}
	}
	words := text.Words()

	// !am mark <имя маркера> или !am mark add <имя маркера>
	if len(words) < 3 || (words[2] == "add" && len(words) < 4) {
		return NonParametr
	}

	markerName := text.Tail(2)
	if words[2] == "add" {
		markerName = text.Tail(3)
	}

	userKey := username + "_" + m.stream.ChannelID()
	if _, ok := cfg.Markers[userKey]; !ok {
		cfg.Markers[userKey] = make(map[string][]*config.Markers)
	}

	live, err := m.api.GetLiveStream()
	if err != nil {
		m.log.Error("Failed to get live stream", err, slog.String("channelID", m.stream.ChannelID()))
		return UnknownError
	}

	marker := &config.Markers{
		StreamID:  live.ID,
		CreatedAt: time.Now(),
		Timecode:  time.Since(live.StartedAt),
	}

	cfg.Markers[userKey][markerName] = append(cfg.Markers[userKey][markerName], marker)
	return nil
}

type ClearMarker struct {
	stream   ports.StreamPort
	username string
}

func (m *ClearMarker) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMarkersClear(cfg, text, m.username)
}

func (m *ClearMarker) SetUsername(username string) {
	m.username = username
}

func (m *ClearMarker) handleMarkersClear(cfg *config.Config, text *ports.MessageText, username string) *ports.AnswerType {
	words := text.Words()
	userKey := username + "_" + m.stream.ChannelID()
	if userMarkers, ok := cfg.Markers[userKey]; ok {
		if len(words) > 3 { // !am mark clear <имя маркера>
			delete(userMarkers, text.Tail(3))
			return nil
		}

		delete(cfg.Markers, userKey) // !am mark clear
	}
	return nil
}

type ListMarker struct {
	stream   ports.StreamPort
	api      ports.APIPort
	fs       ports.FileServerPort
	username string
}

func (m *ListMarker) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMarkersList(cfg, text, m.username)
}

func (m *ListMarker) SetUsername(username string) {
	m.username = username
}

func (m *ListMarker) handleMarkersList(cfg *config.Config, text *ports.MessageText, username string) *ports.AnswerType {
	userMarkers, ok := cfg.Markers[username+"_"+m.stream.ChannelID()]
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
		vods, err := m.api.GetUrlVOD(markers)
		if err != nil {
			return err
		}
		parts = append(parts, name+":")
		for _, m := range markers {
			parts = append(parts, formatMarker(m, vods))
		}
		return nil
	}

	words := text.Words()
	if len(words) > 3 { // !am mark list <имя маркера>
		name := text.Tail(3)
		markers, ok := userMarkers[name]
		if !ok || len(markers) == 0 {
			return &ports.AnswerType{Text: []string{"маркеры не найдены!"}, IsReply: true}
		}
		if len(markers) == 1 {
			vods, err := m.api.GetUrlVOD(markers)
			if err != nil {
				return UnknownError
			}
			return &ports.AnswerType{
				Text:    []string{formatMarker(markers[0], vods)},
				IsReply: true,
			}
		}
		if err := processMarkers(name, markers); err != nil {
			return UnknownError
		}
	} else { // !am mark list
		for name, markers := range userMarkers {
			if err := processMarkers(name, markers); err != nil {
				return UnknownError
			}
			parts = append(parts, "")
		}
	}

	msg := strings.Join(parts, "\n")
	key, err := m.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}

	return &ports.AnswerType{
		Text:    []string{m.fs.GetURL(key)},
		IsReply: true,
	}
}
