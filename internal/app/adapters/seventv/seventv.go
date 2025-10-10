package seventv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
	"unicode/utf8"
)

type SevenTV struct {
	log    logger.Logger
	cfg    *config.Config
	stream ports.StreamPort

	setID    string
	emoteSet map[string]struct{}
}

func New(log logger.Logger, cfg *config.Config, stream ports.StreamPort) *SevenTV {
	s := &SevenTV{
		log:      log,
		cfg:      cfg,
		stream:   stream,
		emoteSet: make(map[string]struct{}),
	}

	user, err := s.GetUserChannel()
	if err != nil {
		s.log.Error("Error getting emotes channel", err)
		return nil
	}

	s.setID = user.EmoteSetID
	for _, e := range user.EmoteSet.Emotes {
		s.emoteSet[strings.TrimSpace(e.Name)] = struct{}{}
	}

	return s
}

func (sv *SevenTV) GetUserChannel() (*ports.User, error) {
	resp, err := http.Get("https://7tv.io/v3/users/twitch/" + sv.stream.ChannelID())
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	defer resp.Body.Close()

	var user ports.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (sv *SevenTV) EmoteStats(words []string) (count int, onlyEmotes bool) {
	if len(words) == 0 {
		return 0, false
	}

	emoteChars := 0
	textChars := 0

	for _, w := range words {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}

		if _, ok := sv.emoteSet[w]; ok {
			count++
			emoteChars += utf8.RuneCountInString(w)
		} else {
			textChars += utf8.RuneCountInString(w)
		}
	}

	total := emoteChars + textChars
	if total > 0 && float64(emoteChars)/float64(total) >= sv.cfg.Spam.SettingsEmotes.EmoteThreshold {
		onlyEmotes = true
	}

	return count, onlyEmotes
}
