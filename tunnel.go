package slipstream

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
)

const (
	defaultKeepAlivePeriod   = 30 * time.Second
	defaultMaxIdleTimeout    = 60 * time.Second
	defaultInitialPacketSize = 1200
)

type TunnelConfig struct {
	Resolvers              []string
	Domain                 string
	TLSConfig              *tls.Config
	InitialPacketSize      uint16
	KeepAlivePeriod        time.Duration
	MaxIdleTimeout         time.Duration
	PacketQueueSize        int
	UDPReadBufferBytes     int
	MaxStreamReceiveWindow uint64
	MaxConnReceiveWindow   uint64
}

type Tunnel struct {
	config TunnelConfig

	mu         sync.RWMutex
	conn       *quic.Conn
	packetConn *DNSPacketConn

	connected    atomic.Bool
	reconnecting atomic.Bool
}

func NewTunnel(config TunnelConfig) (*Tunnel, error) {
	if err := validateTunnelConfig(config); err != nil {
		return nil, err
	}
	return &Tunnel{config: config}, nil
}

func (t *Tunnel) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.packetConn != nil {
		_ = t.packetConn.Close()
	}

	packetConn, err := NewDNSPacketConn(DNSPacketConnConfig{
		Resolvers:          t.config.Resolvers,
		Domain:             t.config.Domain,
		PacketQueueSize:    t.config.PacketQueueSize,
		UDPReadBufferBytes: t.config.UDPReadBufferBytes,
	})
	if err != nil {
		return err
	}

	quicConfig, err := t.quicConfig()
	if err != nil {
		_ = packetConn.Close()
		return err
	}

	dummyAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}
	conn, err := quic.Dial(ctx, packetConn, dummyAddr, t.config.TLSConfig, quicConfig)
	if err != nil {
		_ = packetConn.Close()
		return err
	}

	t.packetConn = packetConn
	t.conn = conn
	t.connected.Store(true)
	return nil
}

func (t *Tunnel) OpenStreamSync(ctx context.Context) (*quic.Stream, error) {
	t.mu.RLock()
	conn := t.conn
	t.mu.RUnlock()
	if conn == nil {
		return nil, errConfig("tunnel is not connected")
	}
	return conn.OpenStreamSync(ctx)
}

func (t *Tunnel) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.connected.Store(false)
	if t.conn != nil {
		_ = t.conn.CloseWithError(0, "")
		t.conn = nil
	}
	if t.packetConn != nil {
		err := t.packetConn.Close()
		t.packetConn = nil
		return err
	}
	return nil
}

func (t *Tunnel) Connected() bool {
	return t.connected.Load()
}

func (t *Tunnel) StartHealthCheck(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				t.mu.RLock()
				conn := t.conn
				t.mu.RUnlock()
				if conn == nil {
					continue
				}
				select {
				case <-conn.Context().Done():
					t.connected.Store(false)
				default:
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (t *Tunnel) quicConfig() (*quic.Config, error) {
	packetSize := t.config.InitialPacketSize
	if packetSize == 0 {
		packetSize = defaultInitialPacketSize
	}

	keepAlive := t.config.KeepAlivePeriod
	if keepAlive <= 0 {
		keepAlive = defaultKeepAlivePeriod
	}
	idleTimeout := t.config.MaxIdleTimeout
	if idleTimeout <= 0 {
		idleTimeout = defaultMaxIdleTimeout
	}
	streamWindow := t.config.MaxStreamReceiveWindow
	if streamWindow == 0 {
		streamWindow = 6 * 1024 * 1024
	}
	connWindow := t.config.MaxConnReceiveWindow
	if connWindow == 0 {
		connWindow = 15 * 1024 * 1024
	}

	return &quic.Config{
		KeepAlivePeriod:            keepAlive,
		MaxIdleTimeout:             idleTimeout,
		MaxStreamReceiveWindow:     streamWindow,
		MaxConnectionReceiveWindow: connWindow,
		InitialPacketSize:          packetSize,
		DisablePathMTUDiscovery:    true,
	}, nil
}

func validateTunnelConfig(config TunnelConfig) error {
	if len(config.Resolvers) == 0 {
		return errConfig("at least one resolver is required")
	}
	if config.Domain == "" {
		return errConfig("domain is required")
	}
	if config.TLSConfig == nil {
		return errConfig("tls config is required")
	}
	return nil
}
