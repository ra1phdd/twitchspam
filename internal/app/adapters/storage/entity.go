package storage

type Message struct {
	Username string
	Text     string
	Time     int64 // UnixNano
	TTL      int64 // time + ttl nano sec
}
