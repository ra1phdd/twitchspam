package stream

type Stream struct {
	channelID   string
	channelName string
	isLive      bool
	streamID    string
	category    string
}

func NewStream(channelName string) *Stream {
	return &Stream{
		channelName: channelName,
	}
}

func (s *Stream) IsLive() bool {
	return s.isLive
}

func (s *Stream) SetIslive(isLive bool) {
	s.isLive = isLive
}

func (s *Stream) StreamID() string {
	return s.streamID
}

func (s *Stream) SetStreamID(streamID string) {
	s.streamID = streamID
}

func (s *Stream) ChannelID() string {
	return s.channelID
}

func (s *Stream) SetChannelID(channelID string) {
	s.channelID = channelID
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
