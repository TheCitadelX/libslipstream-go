package mobile

type Stream struct {
	inner interface {
		Read([]byte) (int, error)
		Write([]byte) (int, error)
		Close() error
	}
}

func (s *Stream) Read(p []byte) (int, error) {
	return s.inner.Read(p)
}

func (s *Stream) Write(p []byte) (int, error) {
	return s.inner.Write(p)
}

func (s *Stream) Close() error {
	return s.inner.Close()
}
