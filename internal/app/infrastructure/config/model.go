package config

import (
	"golang.org/x/time/rate"
	"regexp"
	"time"
)

type Config struct {
	App           App                              `json:"app"`
	Enabled       bool                             `json:"enabled"`
	Limiter       Limiter                          `json:"limiter"`
	WindowSecs    int                              `json:"-"`
	Spam          Spam                             `json:"spam"`
	Automod       Automod                          `json:"automod"`
	Mword         map[string]*Mword                `json:"mword"`
	MwordGroup    map[string]*MwordGroup           `json:"mword_group"`
	Markers       map[string]map[string][]*Markers `json:"markers"` // первый ключ - юзернейм, второй ключ - название маркера
	Commands      map[string]*Commands             `json:"commands"`
	Aliases       map[string]string                `json:"aliases"`       // ключ - алиас, значение - оригинальная команда
	AliasGroups   map[string]*AliasGroups          `json:"aliases_group"` // первый ключ - название группы, второй ключ - алиас, значение - оригинальная команда
	GlobalAliases map[string]string                `json:"global_aliases"`
	Banwords      Banwords                         `json:"banwords"`
}

type App struct {
	LogLevel    string   `json:"log_level"`
	OAuth       string   `json:"oauth,required"`
	ClientID    string   `json:"client_id,required"`
	Username    string   `json:"username,required"`
	UserID      string   `json:"user_id,required"`
	ModChannels []string `json:"mod_channels,required"`
}

type Spam struct {
	Mode            string                         `json:"mode"`            // !am online/always - только на стриме/всегда
	WhitelistUsers  map[string]struct{}            `json:"whitelist_users"` // !am add/del <список>
	SettingsDefault SpamSettings                   `json:"settings_default"`
	SettingsVIP     SpamSettings                   `json:"settings_vip"`
	SettingsEmotes  SpamSettingsEmote              `json:"settings_emotes"`
	Exceptions      map[string]*ExceptionsSettings `json:"exceptions"`
}

type SpamSettings struct {
	Enabled                  bool         `json:"enabled"`
	SimilarityThreshold      float64      `json:"similarity_threshold"`       // !am sim <0.1-1.0>
	MessageLimit             int          `json:"message_limit"`              // !am msg <2-15 или off>
	Punishments              []Punishment `json:"punishments"`                // !am p <значения через запятую>
	DurationResetPunishments int          `json:"duration_reset_punishments"` // !am rto <значение>
	MaxWordLength            int          `json:"max_word_length"`            // !am mwlen <значение или 0 для оффа>
	MaxWordPunishment        Punishment   `json:"max_word_punishment"`
	MinGapMessages           int          `json:"min_gap_messages"` // !am min_gap <0-15>
}

type SpamSettingsEmote struct {
	Enabled                  bool                           `json:"enabled"`
	EmoteThreshold           float64                        `json:"emote_threshold"`
	MessageLimit             int                            `json:"message_limit"`
	Punishments              []Punishment                   `json:"punishments"`
	DurationResetPunishments int                            `json:"duration_reset_punishments"`
	MaxEmotesLength          int                            `json:"max_emotes_length"`
	MaxEmotesPunishment      Punishment                     `json:"max_emotes_punishment"`
	Exceptions               map[string]*ExceptionsSettings `json:"exceptions"`
}

type Automod struct {
	Enabled bool `json:"enabled"`
	Delay   int  `json:"delay"`
}

type ExceptionsSettings struct {
	Enabled      bool           `json:"enabled"`
	MessageLimit int            `json:"message_limit"`
	Punishments  []Punishment   `json:"punishments"`
	Options      ExceptOptions  `json:"options"`
	Regexp       *regexp.Regexp `json:"regexp"`
}

type AliasGroups struct {
	Enabled  bool                `json:"enabled"`
	Aliases  map[string]struct{} `json:"aliases"`
	Original string              `json:"original"`
}

type Mword struct {
	Punishments []Punishment   `json:"punishments"`
	Options     MwordOptions   `json:"options"`
	Regexp      *regexp.Regexp `json:"regexp"`
}

type MwordGroup struct {
	Enabled     bool                      `json:"enabled"`
	Punishments []Punishment              `json:"punishments"`
	Options     MwordOptions              `json:"options"`
	Words       []string                  `json:"words"`
	Regexp      map[string]*regexp.Regexp `json:"regexp"`
}

type Markers struct {
	StreamID  string        `json:"stream_id"`
	CreatedAt time.Time     `json:"date"`
	Timecode  time.Duration `json:"time_code"`
}

type Commands struct {
	Text    string         `json:"text"`
	Options CommandOptions `json:"options"`
	Timer   *Timers        `json:"timer"`
	Limiter *Limiter       `json:"limiter"`
}

type Timers struct {
	Enabled  bool          `json:"enabled"`
	Interval time.Duration `json:"interval"`
	Count    int           `json:"count"`
	Options  TimerOptions  `json:"options"`
}

type Banwords struct {
	Words  []string         `json:"words"`
	Regexp []*regexp.Regexp `json:"regexp"`
}

type Punishment struct {
	Action   string `json:"action"`   // "delete", "ban", "timeout"
	Duration int    `json:"duration"` // только для таймаута
}

type ExceptOptions struct {
	NoSub         bool `json:"no_sub"`
	NoVip         bool `json:"no_vip"`
	NoRepeat      bool `json:"norepeat"`
	OneWord       bool `json:"one_word"`
	Contains      bool `json:"contains"`
	CaseSensitive bool `json:"case_sensitive"`
}

type MwordOptions struct {
	IsFirst       bool `json:"is_first"`
	NoSub         bool `json:"no_sub"`
	NoVip         bool `json:"no_vip"`
	NoRepeat      bool `json:"norepeat"`
	OneWord       bool `json:"one_word"`
	Contains      bool `json:"contains"`
	CaseSensitive bool `json:"case_sensitive"`
}

type TimerOptions struct {
	IsAnnounce    bool   `json:"is_announce"`
	ColorAnnounce string `json:"color_announce"`
	IsAlways      bool   `json:"is_always"`
}

type CommandOptions struct {
	IsPrivate bool `json:"is_private"`
	Mode      int  `json:"mode"`
}

type Limiter struct {
	Requests int           `json:"requests"` // сколько запросов
	Per      time.Duration `json:"per"`      // за какое время
	Rate     *rate.Limiter `json:"-"`
}
