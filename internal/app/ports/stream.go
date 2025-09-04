package ports

type StreamPort interface {
	IsLive() bool
	SetIslive(isLive bool)
	StreamID() string
	SetStreamID(streamID string)
	ChannelID() string
	ChannelName() string
	SetCategory(category string)
	Category() string
}
