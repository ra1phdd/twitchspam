package admin

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleMarkers(cfg *config.Config, text *ports.MessageText, username string) *ports.AnswerType {
	if len(text.Words()) < 3 { // !am mark add/clear/list
		return NonParametr
	}

	handlers := map[string]func(cfg *config.Config, text *ports.MessageText, username string) *ports.AnswerType{
		"clear": a.handleMarkersClear,
		"list":  a.handleMarkersList,
	}

	markerCmd := text.Words()[2]
	if handler, ok := handlers[markerCmd]; ok {
		return handler(cfg, text, username)
	}
	return a.handleMarkersAdd(cfg, text, username)
}

func (a *Admin) handleMarkersAdd(cfg *config.Config, text *ports.MessageText, username string) *ports.AnswerType {
	if !a.stream.IsLive() {
		return &ports.AnswerType{
			Text:    []string{"стрим выключен!"},
			IsReply: true,
		}
	}

	// !am mark <имя маркера> или !am mark add <имя маркера>
	if len(text.Words()) < 3 || (text.Words()[2] == "add" && len(text.Words()) < 4) {
		return NonParametr
	}

	markerName := text.Tail(2)
	if text.Words()[2] == "add" {
		markerName = text.Tail(3)
	}

	userKey := username + "_" + a.stream.ChannelID()
	if _, ok := cfg.Markers[userKey]; !ok {
		cfg.Markers[userKey] = make(map[string][]*config.Markers)
	}

	live, err := a.api.GetLiveStream()
	if err != nil {
		a.log.Error("Failed to get live stream", err, slog.String("channelID", a.stream.ChannelID()))
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

func (a *Admin) handleMarkersClear(cfg *config.Config, text *ports.MessageText, username string) *ports.AnswerType {
	userKey := username + "_" + a.stream.ChannelID()
	if userMarkers, ok := cfg.Markers[userKey]; ok {
		if len(text.Words()) > 3 { // !am mark clear <имя маркера>
			delete(userMarkers, text.Tail(3))
			return nil
		}

		delete(cfg.Markers, userKey) // !am mark clear
	}
	return nil
}

func (a *Admin) handleMarkersList(cfg *config.Config, text *ports.MessageText, username string) *ports.AnswerType {
	userMarkers, ok := cfg.Markers[username+"_"+a.stream.ChannelID()]
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
		vods, err := a.api.GetUrlVOD(markers)
		if err != nil {
			return err
		}
		parts = append(parts, name+":")
		for _, m := range markers {
			parts = append(parts, formatMarker(m, vods))
		}
		return nil
	}

	if len(text.Words()) > 3 { // !am mark list <имя маркера>
		name := text.Tail(3)
		markers, ok := userMarkers[name]
		if !ok || len(markers) == 0 {
			return &ports.AnswerType{Text: []string{"маркеры не найдены!"}, IsReply: true}
		}
		if len(markers) == 1 {
			vods, err := a.api.GetUrlVOD(markers)
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
	key, err := a.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}

	return &ports.AnswerType{
		Text:    []string{a.fs.GetURL(key)},
		IsReply: true,
	}
}
