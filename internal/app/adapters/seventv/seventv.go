package seventv

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	client *http.Client

	setID    string
	emoteSet map[string]struct{}
}

func New(log logger.Logger, cfg *config.Config, stream ports.StreamPort, client *http.Client) *SevenTV {
	log.Trace("Initializing SevenTV instance")

	s := &SevenTV{
		log:      log,
		cfg:      cfg,
		stream:   stream,
		emoteSet: make(map[string]struct{}),
		client:   client,
	}

	log.Debug("Fetching user channel from 7TV API", slog.String("channel", s.stream.ChannelName()))

	user, err := s.GetUserChannel()
	if err != nil {
		s.log.Error("Failed to get 7TV user channel", err, slog.String("channel", s.stream.ChannelName()))
		return nil
	}

	s.setID = user.EmoteSetID
	log.Info("Fetched 7TV user data", slog.String("emote_set_id", s.setID), slog.Int("emote_count", len(user.EmoteSet.Emotes)))

	for _, e := range user.EmoteSet.Emotes {
		name := strings.TrimSpace(e.Name)
		if name == "" {
			s.log.Warn("Skipping empty emote name", slog.Any("emote", e))
			continue
		}
		s.emoteSet[name] = struct{}{}
	}

	s.log.Debug("Successfully initialized SevenTV instance",
		slog.String("set_id", s.setID),
		slog.Int("emote_count", len(s.emoteSet)),
	)

	return s
}

func (sv *SevenTV) GetUserChannel() (*ports.User, error) {
	url := "https://7tv.io/v3/users/twitch/" + sv.stream.ChannelID()
	sv.log.Debug("Sending request to 7TV API", slog.String("url", url))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		sv.log.Error("Failed to create HTTP request", err, slog.String("url", url))
		return nil, err
	}

	resp, err := sv.client.Do(req)
	if err != nil {
		sv.log.Error("HTTP request to 7TV API failed", err, slog.String("url", url))
		return nil, fmt.Errorf("failed to perform request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			sv.log.Warn("Failed to close response body", slog.String("url", url), slog.Any("error", cerr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		sv.log.Warn("7TV API returned non-OK status",
			slog.String("url", url),
			slog.Int("status_code", resp.StatusCode),
		)
	}

	var user ports.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		sv.log.Error("Failed to decode 7TV API response", err, slog.String("url", url))
		return nil, err
	}

	sv.log.Debug("Successfully decoded 7TV user response",
		slog.String("user_id", user.ID),
		slog.String("emote_set_id", user.EmoteSetID),
		slog.Int("emote_count", len(user.EmoteSet.Emotes)),
	)

	return &user, nil
}

func (sv *SevenTV) EmoteStats(words []string) (count int, onlyEmotes bool) {
	sv.log.Trace("Calculating emote statistics",
		slog.Int("word_count", len(words)),
		slog.String("channel_id", sv.stream.ChannelID()),
	)

	if len(words) == 0 {
		sv.log.Debug("No words provided for emote statistics calculation")
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
			sv.log.Trace("Detected emote word",
				slog.String("emote", w),
				slog.Int("current_emote_count", count),
			)
		} else {
			textChars += utf8.RuneCountInString(w)
		}
	}

	total := emoteChars + textChars
	if total == 0 {
		sv.log.Warn("No valid characters counted in emote analysis")
		return count, false
	}

	ratio := float64(emoteChars) / float64(total)
	sv.log.Trace("Computed emote ratio",
		slog.Int("emote_chars", emoteChars),
		slog.Int("text_chars", textChars),
		slog.Float64("ratio", ratio),
		slog.Float64("threshold", sv.cfg.Spam.SettingsEmotes.EmoteThreshold),
	)

	if ratio >= sv.cfg.Spam.SettingsEmotes.EmoteThreshold {
		onlyEmotes = true
		sv.log.Debug("Message detected as emote-only",
			slog.Float64("ratio", ratio),
			slog.Int("emote_count", count),
			slog.Float64("threshold", sv.cfg.Spam.SettingsEmotes.EmoteThreshold),
		)
	} else {
		sv.log.Trace("Message contains mixed content",
			slog.Float64("ratio", ratio),
			slog.Int("emote_count", count),
		)
	}

	return count, onlyEmotes
}
