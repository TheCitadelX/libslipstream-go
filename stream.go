package slipstream

import "github.com/quic-go/quic-go"

type Stream struct {
	stream *quic.Stream
}

func (s *Stream) Read(p []byte) (int, error) {
	return s.stream.Read(p)
}

func (s *Stream) Write(p []byte) (int, error) {
	return s.stream.Write(p)
}

func (s *Stream) Close() error {
	return s.stream.Close()
}
