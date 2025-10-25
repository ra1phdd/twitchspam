package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// BotEnabled - включен ли бот.
	BotEnabled = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bot_enabled",
		Help: "Whether the bot is enabled (1) or disabled (0)",
	})

	// AntiSpamEnabled - включен ли антиспам.
	AntiSpamEnabled = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "anti_spam_enabled",
			Help: "Whether the anti-spam is enabled (1) or disabled (0)",
		}, []string{"type"},
	)

	// StreamActive - включен ли стрим по каналам.
	StreamActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "stream_active",
			Help: "Whether the stream is currently active or not",
		},
		[]string{"channel"},
	)

	// OnlineViewers - онлайн по каналам.
	OnlineViewers = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bot_online_viewers",
			Help: "Current number of online viewers per channel",
		},
		[]string{"channel"},
	)

	// StreamStartTime - время начала стрима по каналам (Unix timestamp).
	StreamStartTime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "stream_start_timestamp",
			Help: "Start time of the stream as Unix timestamp per channel",
		},
		[]string{"channel"},
	)

	// StreamEndTime - время конца стрима по каналам (Unix timestamp).
	StreamEndTime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "stream_end_timestamp",
			Help: "End time of the stream as Unix timestamp per channel",
		},
		[]string{"channel"},
	)

	// MessagesPerStream - количество сообщений за стрим по каналам.
	MessagesPerStream = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bot_messages_total",
			Help: "Total number of messages per stream channel",
		},
		[]string{"channel"},
	)

	// MessageProcessingTime - время обработки сообщений.
	MessageProcessingTime = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "bot_message_processing_milliseconds",
			Help:    "Average time to process a message",
			Buckets: prometheus.ExponentialBuckets(0.00005, 1.5, 25),
		},
	)

	// ModerationActions - количество банов, мутов и удалений сообщений по каналам.
	ModerationActions = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bot_moderation_actions",
			Help: "Number of moderation actions per channel",
		},
		[]string{"channel", "action"},
	)

	// UserCommands - количество вызовов пользовательских команд по каналам.
	UserCommands = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bot_user_commands_total",
			Help: "Total number of user commands called per channel and per command",
		},
		[]string{"channel", "command"},
	)
)
