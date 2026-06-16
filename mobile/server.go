package mobile

import (
	"fmt"

	core "github.com/TheCitadelX/libslipstream-go"
)

type Server struct {
	inner *core.Server
}

func NewServer(config *ServerConfig) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("server config is required")
	}
	inner, err := core.NewServer(config.toCore())
	if err != nil {
		return nil, err
	}
	return &Server{inner: inner}, nil
}

func (s *Server) Start() error {
	return s.inner.Start()
}

func (s *Server) Stop() error {
	return s.inner.Stop()
}

func (s *Server) LocalDNSAddress() string {
	return s.inner.LocalDNSAddress()
}
