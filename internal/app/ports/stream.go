package ports

type StreamPort interface {
	IsLive() bool
	SetIslive(isLive bool)
	ChannelID() string
	ChannelName() string
	SetCategory(category string)
	Category() string
}
