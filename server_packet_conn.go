package slipstream

import (
	"net"
	"sync"
	"time"

	sdns "github.com/TheCitadelX/libslipstream-go/pkg/dns"
)

const (
	defaultServerPacketQueueSize = 4096
	defaultResponseWait          = 10 * time.Millisecond
)

type serverPacketBundle struct {
	data []byte
	addr net.Addr
}

type ServerPacketConn struct {
	incoming    chan serverPacketBundle
	outgoing    map[string]chan []byte
	reassembly  map[string]*reassembler
	fragmenters map[string]*fragmenter
	domain      string
	done        chan struct{}
	once        sync.Once
	mu          sync.Mutex
}

func NewServerPacketConn(domain string, queueSize int) *ServerPacketConn {
	if queueSize <= 0 {
		queueSize = defaultServerPacketQueueSize
	}
	return &ServerPacketConn{
		incoming:    make(chan serverPacketBundle, queueSize),
		outgoing:    make(map[string]chan []byte),
		reassembly:  make(map[string]*reassembler),
		fragmenters: make(map[string]*fragmenter),
		domain:      domain,
		done:        make(chan struct{}),
	}
}

func (c *ServerPacketConn) InjectPacket(data []byte, peer *net.UDPAddr) {
	if len(data) == 1 && data[0] == pollFrame {
		c.ensureOutgoing(peer.String())
		return
	}
	packet := c.reassemblerFor(peer.String()).ingest(data)
	if len(packet) == 0 {
		return
	}
	addr := serverPeerAddr{key: peer.String(), addr: peer}
	c.ensureOutgoing(addr.key)
	select {
	case c.incoming <- serverPacketBundle{data: packet, addr: addr}:
	default:
	}
}

func (c *ServerPacketConn) WaitResponse(peer *net.UDPAddr, timeout time.Duration) []byte {
	if timeout <= 0 {
		timeout = defaultResponseWait
	}
	ch := c.ensureOutgoing(peer.String())
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case packet := <-ch:
		return packet
	case <-timer.C:
		return nil
	case <-c.done:
		return nil
	}
}

func (c *ServerPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	select {
	case bundle := <-c.incoming:
		return copy(p, bundle.data), bundle.addr, nil
	case <-c.done:
		return 0, nil, net.ErrClosed
	}
}

func (c *ServerPacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	key := addr.String()
	if peer, ok := addr.(serverPeerAddr); ok {
		key = peer.key
	}
	maxPayload, err := sdns.MaxPayloadLenForDomain(c.domain)
	if err != nil {
		return 0, err
	}
	fragments, err := c.fragmenterFor(key).split(p, maxPayload)
	if err != nil {
		return 0, err
	}
	ch := c.ensureOutgoing(key)
	for _, fragment := range fragments {
		select {
		case ch <- fragment:
		default:
			return 0, nil
		}
	}
	return len(p), nil
}

func (c *ServerPacketConn) Close() error {
	c.once.Do(func() {
		close(c.done)
	})
	return nil
}

func (c *ServerPacketConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 53}
}

func (c *ServerPacketConn) SetDeadline(time.Time) error      { return nil }
func (c *ServerPacketConn) SetReadDeadline(time.Time) error  { return nil }
func (c *ServerPacketConn) SetWriteDeadline(time.Time) error { return nil }

func (c *ServerPacketConn) ensureOutgoing(key string) chan []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := c.outgoing[key]
	if ch == nil {
		ch = make(chan []byte, defaultServerPacketQueueSize)
		c.outgoing[key] = ch
	}
	return ch
}

func (c *ServerPacketConn) reassemblerFor(key string) *reassembler {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := c.reassembly[key]
	if r == nil {
		r = newReassembler()
		c.reassembly[key] = r
	}
	return r
}

func (c *ServerPacketConn) fragmenterFor(key string) *fragmenter {
	c.mu.Lock()
	defer c.mu.Unlock()
	f := c.fragmenters[key]
	if f == nil {
		f = &fragmenter{}
		c.fragmenters[key] = f
	}
	return f
}

type serverPeerAddr struct {
	key  string
	addr *net.UDPAddr
}

func (a serverPeerAddr) Network() string { return "udp" }
func (a serverPeerAddr) String() string  { return a.key }
