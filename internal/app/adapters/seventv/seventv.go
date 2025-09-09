package seventv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type SevenTV struct {
	log    logger.Logger
	stream ports.StreamPort

	setID    string
	emoteSet map[string]struct{}
}

func New(log logger.Logger, stream ports.StreamPort) *SevenTV {
	s := &SevenTV{
		log:      log,
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
		s.emoteSet[e.Name] = struct{}{}
	}

	//go s.runEventLoop()
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

func (sv *SevenTV) IsOnlyEmotes(words []string) bool {
	if len(words) == 0 {
		return false
	}

	for _, w := range words {
		if _, ok := sv.emoteSet[w]; !ok {
			return false
		}
	}

	return true
}

func (sv *SevenTV) CountEmotes(words []string) int {
	if len(words) == 0 {
		return 0
	}

	var emotes int
	for _, w := range words {
		if _, ok := sv.emoteSet[w]; ok {
			emotes++
		}
	}

	return emotes
}

//func (sv *SevenTV) runEventLoop() {
//	for {
//		err := sv.connectAndHandleEvents()
//		if err != nil {
//			sv.log.Warn("7TV WS connection lost, retrying...", slog.String("error", err.Error()))
//			time.Sleep(5 * time.Second)
//		}
//	}
//}
//
//func (sv *SevenTV) connectAndHandleEvents() error {
//	conn, resp, err := websocket.DefaultDialer.Dial("wss://events.7tv.io/v3", nil)
//	if err != nil {
//		log.Fatalf("Dial error: %v (resp: %+v)", err, resp)
//	}
//	defer conn.Close()
//
//	sv.log.Info("Connected to 7TV EventAPI")
//	sub := map[string]interface{}{
//		"op": 35, // Subscribe
//		"d": map[string]interface{}{
//			"type": "emote_set.update",
//			"condition": map[string]string{
//				"object_id": sv.setID,
//			},
//		},
//	}
//	subBytes, _ := json.Marshal(sub)
//	conn.WriteMessage(websocket.TextMessage, subBytes)
//
//	for {
//		_, msgBytes, err := conn.ReadMessage()
//		if err != nil {
//			sv.log.Error("Error while reading 7TV event", err)
//			return err
//		}
//
//		var event ports.SevenTVMessage
//		if err := json.Unmarshal(msgBytes, &event); err != nil {
//			sv.log.Error("Failed to decode 7TV event", err, slog.String("event", string(msgBytes)))
//			continue
//		}
//
//		var m map[string]any
//		if err := json.Unmarshal(event.D, &m); err != nil {
//			log.Fatal("decode map error:", err)
//		}
//
//		eventType, _ := m["type"].(string)
//		switch eventType {
//		case "emote_set.update":
//			var upd ports.EmoteSetUpdate
//			if err := json.Unmarshal(event.D, &upd); err != nil {
//				sv.log.Error("Failed to decode emote_set.update", err)
//				break
//			}
//
//			for _, em := range upd.Pushed {
//				sv.log.Info("7TV: Emote added", slog.String("name", em.Name), slog.String("id", em.ID))
//			}
//			for _, em := range upd.Pulled {
//				sv.log.Info("7TV: Emote removed", slog.String("name", em.Name), slog.String("id", em.ID))
//			}
//		case "hello":
//			sv.log.Info("Received Hello from 7TV")
//		case "ack":
//			sv.log.Info("Subscription acknowledged")
//		default:
//			sv.log.Info("Unhandled 7TV event", slog.String("raw", string(msgBytes)))
//		}
//	}
//}
