package ports

import "encoding/json"

type SevenTVPort interface {
	GetUserChannel() (*User, error)
	IsOnlyEmotes(words []string) bool
	CountEmotes(words []string) int
}

type User struct {
	ID          string   `json:"id"`
	Platform    string   `json:"platform"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	LinkedAt    int64    `json:"linked_at"`
	EmoteCap    int      `json:"emote_capacity"`
	EmoteSetID  string   `json:"emote_set_id"`
	EmoteSet    EmoteSet `json:"emote_set"`
}

type EmoteSet struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Flags      int      `json:"flags"`
	Tags       []string `json:"tags"`
	Immutable  bool     `json:"immutable"`
	Privileged bool     `json:"privileged"`
	Emotes     []Emote  `json:"emotes"`
}

type Emote struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Flags     int       `json:"flags"`
	Timestamp int64     `json:"timestamp"`
	ActorID   string    `json:"actor_id"`
	Data      EmoteData `json:"data"`
	OriginID  *string   `json:"origin_id"`
}

type EmoteData struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Flags     int      `json:"flags"`
	Lifecycle int      `json:"lifecycle"`
	State     []string `json:"state"`
	Listed    bool     `json:"listed"`
	Animated  bool     `json:"animated"`
	Owner     Owner    `json:"owner"`
	Host      Host     `json:"host"`
}

type Owner struct {
	ID          string       `json:"id"`
	Username    string       `json:"username"`
	DisplayName string       `json:"display_name"`
	AvatarURL   string       `json:"avatar_url"`
	Style       interface{}  `json:"style"`
	RoleIDs     []string     `json:"role_ids"`
	Connections []Connection `json:"connections"`
}

type Connection struct {
	ID          string `json:"id"`
	Platform    string `json:"platform"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	LinkedAt    int64  `json:"linked_at"`
	EmoteCap    int    `json:"emote_capacity"`
	EmoteSetID  string `json:"emote_set_id"`
}

type Host struct {
	URL   string     `json:"url"`
	Files []HostFile `json:"files"`
}

type HostFile struct {
	Name       string `json:"name"`
	StaticName string `json:"static_name"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	FrameCount int    `json:"frame_count"`
	Size       int    `json:"size"`
	Format     string `json:"format"`
}

type SevenTVMessage struct {
	Op int             `json:"op"`
	T  int             `json:"t"`
	D  json.RawMessage `json:"d"`
}

type EmoteSetUpdate struct {
	ID     string `json:"id"`
	Pushed []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"pushed"`
	Pulled []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"pulled"`
}
