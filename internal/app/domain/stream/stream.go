package stream

type Stream struct {
	channelID   string
	channelName string
	isLive      bool
	category    string
}

func NewStream(channelID string, channelName string) *Stream {
	return &Stream{
		channelID:   channelID,
		channelName: channelName,
	}
}

func (s *Stream) IsLive() bool {
	return s.isLive
}

func (s *Stream) SetIslive(isLive bool) {
	s.isLive = isLive
}

func (s *Stream) ChannelID() string {
	return s.channelID
}

func (s *Stream) ChannelName() string {
	return s.channelName
}

func (s *Stream) SetCategory(category string) {
	s.category = category
}

func (s *Stream) Category() string {
	return s.category
}
