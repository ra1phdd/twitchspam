package admin

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleMarkers(cfg *config.Config, _ string, args []string, username string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}
	markerCmd, markerArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string, username string) *ports.AnswerType{
		"clear": a.handleMarkersClear,
		"list":  a.handleMarkersList,
	}

	if handler, ok := handlers[markerCmd]; ok {
		return handler(cfg, markerCmd, markerArgs, username)
	}
	return a.handleMarkersAdd(cfg, markerCmd, markerArgs, username)
}

func (a *Admin) handleMarkersAdd(cfg *config.Config, markerCmd string, args []string, username string) *ports.AnswerType {
	if !a.stream.IsLive() {
		return &ports.AnswerType{
			Text:    []string{"стрим выключен!"},
			IsReply: true,
		}
	}

	if markerCmd == "add" && len(args) < 1 {
		return NonParametr
	}

	markerName := markerCmd
	//markerArgs := args
	if markerCmd == "add" {
		markerName = args[0]
		//markerArgs = args[1:]
	}

	userKey := username + "_" + a.stream.ChannelID()
	if _, ok := cfg.Markers[userKey]; !ok {
		cfg.Markers[userKey] = make(map[string][]*config.Markers)
	}

	live, err := a.api.GetLiveStream(a.stream.ChannelID())
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

func (a *Admin) handleMarkersClear(cfg *config.Config, _ string, args []string, username string) *ports.AnswerType {
	userKey := username + "_" + a.stream.ChannelID()
	if userMarkers, ok := cfg.Markers[userKey]; ok {
		if len(args) < 1 {
			delete(cfg.Markers, userKey)
		} else {
			delete(userMarkers, args[0])
		}
	}
	return nil
}

func (a *Admin) handleMarkersList(cfg *config.Config, _ string, args []string, username string) *ports.AnswerType {
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
		return fmt.Sprintf("%s - вод не найден (stream_id %s, таймкод %s)", m.CreatedAt.Format("02.01"), m.StreamID, timecode)
	}

	var parts []string
	if len(args) < 1 {
		for name, markers := range userMarkers {
			parts = append(parts, name+":")
			vods, err := a.api.GetUrlVOD(a.stream.ChannelID(), markers)
			if err != nil {
				return UnknownError
			}

			for _, m := range markers {
				parts = append(parts, formatMarker(m, vods))
			}
			parts = append(parts, "\n")
		}
	} else {
		markerName := args[0]
		markers, ok := userMarkers[markerName]
		if !ok || len(markers) == 0 {
			return &ports.AnswerType{
				Text:    []string{"маркеры не найдены!"},
				IsReply: true,
			}
		}

		vods, err := a.api.GetUrlVOD(a.stream.ChannelID(), markers)
		if err != nil {
			return UnknownError
		}

		if len(markers) == 1 {
			return &ports.AnswerType{
				Text:    []string{formatMarker(markers[0], vods)},
				IsReply: true,
			}
		}

		parts = append(parts, markerName+":")
		for _, m := range markers {
			parts = append(parts, formatMarker(m, vods))
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
