package mobile

import (
	"fmt"

	core "github.com/TheCitadelX/libslipstream-go"
)

type Server struct {
	inner  *core.Server
	events *EventQueue
}

func NewServer(config *ServerConfig) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("server config is required")
	}
	inner, err := core.NewServer(config.toCore())
	if err != nil {
		return nil, err
	}
	events := NewEventQueue(config.EventQueueSize)
	events.emit("info", "server", "server created")
	return &Server{inner: inner, events: events}, nil
}

func (s *Server) Start() error {
	s.events.emit("info", "server", "starting server")
	if err := s.inner.Start(); err != nil {
		s.events.emit("error", "server", "start failed: "+err.Error())
		return err
	}
	s.events.emit("info", "server", "server listening on "+s.LocalDNSAddress())
	return nil
}

func (s *Server) Stop() error {
	s.events.emit("info", "server", "stopping server")
	if err := s.inner.Stop(); err != nil {
		s.events.emit("error", "server", "stop failed: "+err.Error())
		return err
	}
	s.events.emit("info", "server", "server stopped")
	return nil
}

func (s *Server) LocalDNSAddress() string {
	return s.inner.LocalDNSAddress()
}

func (s *Server) Events() *EventQueue {
	return s.events
}
