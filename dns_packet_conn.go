package slipstream

import (
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sdns "github.com/TheCitadelX/libslipstream-go/pkg/dns"
)

const (
	defaultPacketQueueSize = 2048
	defaultUDPReadBuffer   = 4 * 1024 * 1024
	defaultPollInterval    = 25 * time.Millisecond
	defaultIdleThreshold   = 100 * time.Millisecond
	pollFrame              = 0x20
)

type DNSPacketConnConfig struct {
	Resolvers          []string
	Domain             string
	PacketQueueSize    int
	UDPReadBufferBytes int
	PollInterval       time.Duration
	IdleThreshold      time.Duration
}

type DNSPacketConn struct {
	conn      *net.UDPConn
	resolvers []*net.UDPAddr
	domain    string
	localAddr net.Addr

	rxQueue chan []byte
	done    chan struct{}
	closed  atomic.Bool
	once    sync.Once

	nextID       atomic.Uint32
	nextResolver atomic.Uint64
	lastWrite    atomic.Int64
	fragments    fragmenter
	reassembler  *reassembler

	mu            sync.Mutex
	readDeadline  time.Time
	writeDeadline time.Time
	pollInterval  time.Duration
	idleThreshold time.Duration
}

func NewDNSPacketConn(config DNSPacketConnConfig) (*DNSPacketConn, error) {
	domain := strings.Trim(strings.TrimSpace(config.Domain), ".")
	if domain == "" {
		return nil, errConfig("domain is required")
	}
	if len(config.Resolvers) == 0 {
		return nil, errConfig("at least one resolver is required")
	}

	resolvers := make([]*net.UDPAddr, 0, len(config.Resolvers))
	for _, resolver := range config.Resolvers {
		addr, err := net.ResolveUDPAddr("udp", normalizeResolver(resolver))
		if err != nil {
			return nil, err
		}
		resolvers = append(resolvers, addr)
	}

	udp, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}

	readBuffer := config.UDPReadBufferBytes
	if readBuffer <= 0 {
		readBuffer = defaultUDPReadBuffer
	}
	_ = udp.SetReadBuffer(readBuffer)

	queueSize := config.PacketQueueSize
	if queueSize <= 0 {
		queueSize = defaultPacketQueueSize
	}

	c := &DNSPacketConn{
		conn:        udp,
		resolvers:   resolvers,
		domain:      domain,
		localAddr:   &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0},
		rxQueue:     make(chan []byte, queueSize),
		done:        make(chan struct{}),
		reassembler: newReassembler(),
	}
	c.pollInterval = config.PollInterval
	if c.pollInterval <= 0 {
		c.pollInterval = defaultPollInterval
	}
	c.idleThreshold = config.IdleThreshold
	if c.idleThreshold <= 0 {
		c.idleThreshold = defaultIdleThreshold
	}
	c.lastWrite.Store(time.Now().UnixNano())
	go c.readLoop()
	go c.pollLoop()
	return c, nil
}

func (c *DNSPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	for {
		deadline := c.getReadDeadline()
		if deadline.IsZero() {
			select {
			case packet := <-c.rxQueue:
				return copy(p, packet), c.localAddr, nil
			case <-c.done:
				return 0, nil, net.ErrClosed
			}
		}

		wait := time.Until(deadline)
		if wait <= 0 {
			return 0, nil, os.ErrDeadlineExceeded
		}
		timer := time.NewTimer(wait)
		select {
		case packet := <-c.rxQueue:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return copy(p, packet), c.localAddr, nil
		case <-timer.C:
			return 0, nil, os.ErrDeadlineExceeded
		case <-c.done:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return 0, nil, net.ErrClosed
		}
	}
}

func (c *DNSPacketConn) WriteTo(p []byte, _ net.Addr) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if c.closed.Load() {
		return 0, net.ErrClosed
	}
	if deadline := c.getWriteDeadline(); !deadline.IsZero() && time.Now().After(deadline) {
		return 0, os.ErrDeadlineExceeded
	}

	c.lastWrite.Store(time.Now().UnixNano())
	maxPayload, err := sdns.MaxPayloadLenForDomain(c.domain)
	if err != nil {
		return 0, err
	}
	fragments, err := c.fragments.split(p, maxPayload)
	if err != nil {
		return 0, err
	}
	for _, fragment := range fragments {
		packet, _, err := sdns.EncodePayloadQuery(c.nextQueryID(), fragment, c.domain)
		if err != nil {
			return 0, err
		}
		if _, err := c.conn.WriteToUDP(packet, c.nextResolverAddr()); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (c *DNSPacketConn) Close() error {
	var err error
	c.once.Do(func() {
		c.closed.Store(true)
		close(c.done)
		err = c.conn.Close()
	})
	return err
}

func (c *DNSPacketConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *DNSPacketConn) SetDeadline(t time.Time) error {
	c.mu.Lock()
	c.readDeadline = t
	c.writeDeadline = t
	c.mu.Unlock()
	_ = c.conn.SetDeadline(t)
	return nil
}

func (c *DNSPacketConn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	c.readDeadline = t
	c.mu.Unlock()
	_ = c.conn.SetReadDeadline(t)
	return nil
}

func (c *DNSPacketConn) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	c.writeDeadline = t
	c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(t)
	return nil
}

func (c *DNSPacketConn) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, _, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-c.done:
				return
			default:
				continue
			}
		}
		payload := sdns.DecodeResponse(buf[:n])
		if len(payload) == 0 {
			continue
		}
		packet := c.reassembler.ingest(payload)
		if len(packet) == 0 {
			continue
		}
		select {
		case c.rxQueue <- packet:
		default:
		}
	}
}

func (c *DNSPacketConn) pollLoop() {
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			lastWrite := time.Unix(0, c.lastWrite.Load())
			if time.Since(lastWrite) < c.idleThreshold {
				continue
			}
			_ = c.sendPoll()
		case <-c.done:
			return
		}
	}
}

func (c *DNSPacketConn) sendPoll() error {
	packet, _, err := sdns.EncodePayloadQuery(c.nextQueryID(), []byte{pollFrame}, c.domain)
	if err != nil {
		return err
	}
	_, err = c.conn.WriteToUDP(packet, c.nextResolverAddr())
	return err
}

func (c *DNSPacketConn) nextQueryID() uint16 {
	return uint16(c.nextID.Add(1))
}

func (c *DNSPacketConn) nextResolverAddr() *net.UDPAddr {
	next := c.nextResolver.Add(1) - 1
	return c.resolvers[int(next%uint64(len(c.resolvers)))]
}

func (c *DNSPacketConn) getReadDeadline() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readDeadline
}

func (c *DNSPacketConn) getWriteDeadline() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writeDeadline
}

func normalizeResolver(resolver string) string {
	resolver = strings.TrimSpace(resolver)
	if resolver == "" {
		return resolver
	}
	if _, _, err := net.SplitHostPort(resolver); err == nil {
		return resolver
	}
	if strings.Count(resolver, ":") > 1 {
		return net.JoinHostPort(resolver, "53")
	}
	return net.JoinHostPort(resolver, "53")
}
