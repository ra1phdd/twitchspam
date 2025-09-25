package stream

import (
	"strconv"
	"testing"
)

func BenchmarkStream_IsLive(b *testing.B) {
	s := NewStream("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.IsLive()
	}
}

func BenchmarkStream_SetIslive(b *testing.B) {
	s := NewStream("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.SetIslive(i%2 == 0)
	}
}

func BenchmarkStream_StreamID(b *testing.B) {
	s := NewStream("test")
	s.SetStreamID("12345")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.StreamID()
	}
}

func BenchmarkStream_SetStreamID(b *testing.B) {
	s := NewStream("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.SetStreamID(strconv.Itoa(i))
	}
}

func BenchmarkStream_ChannelID(b *testing.B) {
	s := NewStream("test")
	s.SetChannelID("chan123")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.ChannelID()
	}
}

func BenchmarkStream_SetChannelID(b *testing.B) {
	s := NewStream("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.SetChannelID("chan" + strconv.Itoa(i))
	}
}

func BenchmarkStream_ChannelName(b *testing.B) {
	s := NewStream("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.ChannelName()
	}
}

func BenchmarkStream_SetCategory(b *testing.B) {
	s := NewStream("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.SetCategory("cat" + strconv.Itoa(i))
	}
}

func BenchmarkStream_Category(b *testing.B) {
	s := NewStream("test")
	s.SetCategory("gaming")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Category()
	}
}
