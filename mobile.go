package slipstream

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/quic-go"

	sdns "slipstream-go/pkg/dns"
)

type ClientConfig struct {
	ResolverAddress   string
	Resolvers         []string
	Domain            string
	TCPListenAddress  string
	CertFingerprint   string
	PinnedCertPEM     []byte
	ServerName        string
	AllowInsecure     bool
	InitialPacketSize uint16
}

func (c ClientConfig) Validate() error {
	if len(c.resolverList()) == 0 {
		return errConfig("at least one resolver is required")
	}
	if strings.Trim(strings.TrimSpace(c.Domain), ".") == "" {
		return errConfig("domain is required")
	}
	return nil
}

type ServerConfig struct {
	DNSListenAddress string
	TargetAddress    string
	Domain           string
	Domains          []string
	CertPEM          []byte
	KeyPEM           []byte
	ResponseWait     time.Duration
	PacketQueueSize  int
}

func (c ServerConfig) Validate() error {
	if strings.TrimSpace(c.DNSListenAddress) == "" {
		return errConfig("dns listen address is required")
	}
	if len(c.domainList()) == 0 {
		return errConfig("at least one domain is required")
	}
	return nil
}

type Client struct {
	config ClientConfig
	tunnel *Tunnel
	cancel context.CancelFunc

	proxyMu       sync.Mutex
	proxyListener net.Listener
	proxyCancel   context.CancelFunc
}

func NewClient(config ClientConfig) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Client{config: config}, nil
}

func (c *Client) Start() error {
	if c.tunnel != nil && c.tunnel.Connected() {
		return nil
	}

	tlsConfig, err := c.config.tlsConfig()
	if err != nil {
		return err
	}
	tunnel, err := NewTunnel(TunnelConfig{
		Resolvers:         c.config.resolverList(),
		Domain:            c.config.Domain,
		TLSConfig:         tlsConfig,
		InitialPacketSize: c.config.InitialPacketSize,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := tunnel.Connect(ctx); err != nil {
		cancel()
		return err
	}
	tunnel.StartHealthCheck(ctx, 0)

	c.tunnel = tunnel
	c.cancel = cancel
	return nil
}

func (c *Client) Stop() error {
	if err := c.StopSOCKS5(); err != nil {
		return err
	}
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	if c.tunnel == nil {
		return nil
	}
	err := c.tunnel.Close()
	c.tunnel = nil
	return err
}

func (c *Client) Connected() bool {
	return c.tunnel != nil && c.tunnel.Connected()
}

func (c *Client) OpenStreamSync(ctx context.Context) (*quic.Stream, error) {
	if c.tunnel == nil {
		return nil, errConfig("client is not started")
	}
	return c.tunnel.OpenStreamSync(ctx)
}

func (c *Client) DialTCP(target string) (*Stream, error) {
	if c.tunnel == nil {
		return nil, errConfig("client is not started")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := c.tunnel.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	if err := writeTargetAddress(stream, target); err != nil {
		_ = stream.Close()
		return nil, err
	}
	return &Stream{stream: stream}, nil
}

func (c *Client) StartSOCKS5(listenAddr string) (string, error) {
	if c.tunnel == nil {
		return "", errConfig("client is not started")
	}

	if strings.TrimSpace(listenAddr) == "" {
		listenAddr = strings.TrimSpace(c.config.TCPListenAddress)
	}
	if strings.TrimSpace(listenAddr) == "" {
		listenAddr = "127.0.0.1:1080"
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return "", err
	}

	c.proxyMu.Lock()
	if c.proxyListener != nil {
		c.proxyMu.Unlock()
		_ = listener.Close()
		return "", errConfig("socks5 proxy already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.proxyListener = listener
	c.proxyCancel = cancel
	c.proxyMu.Unlock()

	go c.serveSOCKS5(ctx, listener)
	return listener.Addr().String(), nil
}

func (c *Client) StopSOCKS5() error {
	c.proxyMu.Lock()
	listener := c.proxyListener
	cancel := c.proxyCancel
	c.proxyListener = nil
	c.proxyCancel = nil
	c.proxyMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if listener != nil {
		return listener.Close()
	}
	return nil
}

type Server struct {
	config     ServerConfig
	udpConn    *net.UDPConn
	packetConn *ServerPacketConn
	listener   *quic.Listener
	cancel     context.CancelFunc
	once       sync.Once
}

func NewServer(config ServerConfig) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Server{config: config}, nil
}

func (s *Server) Start() error {
	if err := s.config.Validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	tlsConfig, err := TLSConfigFromKeyPairPEM(s.config.CertPEM, s.config.KeyPEM)
	if err != nil {
		cancel()
		return err
	}

	addr, err := net.ResolveUDPAddr("udp", s.config.DNSListenAddress)
	if err != nil {
		cancel()
		return err
	}
	udpConn, err := net.ListenUDP("udp", addr)
	if err != nil {
		cancel()
		return err
	}
	s.udpConn = udpConn

	packetConn := NewServerPacketConn(s.config.capacityDomain(), s.config.PacketQueueSize)
	s.packetConn = packetConn

	transport := &quic.Transport{
		Conn: packetConn,
		VerifySourceAddress: func(net.Addr) bool {
			return true
		},
	}
	listener, err := transport.Listen(tlsConfig, &quic.Config{
		KeepAlivePeriod:            defaultKeepAlivePeriod,
		MaxIdleTimeout:             5 * time.Minute,
		MaxIncomingStreams:         1000,
		MaxIncomingUniStreams:      1000,
		MaxStreamReceiveWindow:     6 * 1024 * 1024,
		MaxConnectionReceiveWindow: 15 * 1024 * 1024,
		InitialPacketSize:          900,
		DisablePathMTUDiscovery:    true,
	})
	if err != nil {
		_ = udpConn.Close()
		_ = packetConn.Close()
		cancel()
		return err
	}
	s.listener = listener

	go s.dnsLoop(ctx)
	go s.acceptLoop(ctx)
	return nil
}

func (s *Server) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	if s.packetConn != nil {
		_ = s.packetConn.Close()
	}
	if s.udpConn != nil {
		return s.udpConn.Close()
	}
	return nil
}

func (s *Server) LocalDNSAddress() string {
	if s.udpConn == nil {
		return ""
	}
	return s.udpConn.LocalAddr().String()
}

func (c ClientConfig) resolverList() []string {
	var resolvers []string
	for _, resolver := range c.Resolvers {
		resolver = strings.TrimSpace(resolver)
		if resolver != "" {
			resolvers = append(resolvers, resolver)
		}
	}
	if len(resolvers) == 0 {
		resolver := strings.TrimSpace(c.ResolverAddress)
		if resolver != "" {
			resolvers = append(resolvers, resolver)
		}
	}
	return resolvers
}

func (c ClientConfig) tlsConfig() (*tls.Config, error) {
	switch {
	case len(c.PinnedCertPEM) > 0:
		return TLSConfigFromPinnedCertPEM(c.PinnedCertPEM)
	case strings.TrimSpace(c.CertFingerprint) != "":
		return TLSConfigFromCertSHA256(c.CertFingerprint)
	case strings.TrimSpace(c.ServerName) != "":
		return &tls.Config{
			ServerName: strings.TrimSpace(c.ServerName),
			NextProtos: []string{nextProto},
		}, nil
	case c.AllowInsecure:
		return InsecureTLSConfig(), nil
	default:
		return nil, errConfig("certificate pinning, server name, or AllowInsecure is required")
	}
}

func (s *Server) dnsLoop(ctx context.Context) {
	buf := make([]byte, 4096)
	for {
		n, peer, err := s.udpConn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
		packet := append([]byte(nil), buf[:n]...)
		go s.handleDNSPacket(packet, peer)
	}
}

func (s *Server) handleDNSPacket(packet []byte, peer *net.UDPAddr) {
	decoded, err := sdns.DecodeQueryWithDomains(packet, s.config.domainList())
	if err != nil {
		if response, ok := sdns.ReplyFromDecodeError(err); ok {
			if out, encodeErr := sdns.EncodeResponse(*response); encodeErr == nil {
				_, _ = s.udpConn.WriteToUDP(out, peer)
			}
		}
		return
	}

	s.packetConn.InjectPacket(decoded.Payload, peer)
	payload := s.packetConn.WaitResponse(peer, s.config.ResponseWait)
	response, err := sdns.EncodeResponse(sdns.ResponseParams{
		ID:       decoded.ID,
		RD:       decoded.RD,
		CD:       decoded.CD,
		Question: decoded.Question,
		Payload:  payload,
	})
	if err != nil {
		return
	}
	_, _ = s.udpConn.WriteToUDP(response, peer)
}

func (s *Server) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.listener.Accept(ctx)
		if err != nil {
			return
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn *quic.Conn) {
	defer conn.CloseWithError(0, "")
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			return
		}
		go s.handleStream(stream)
	}
}

func (s *Server) handleStream(stream *quic.Stream) {
	defer stream.Close()
	targetAddr, err := readTargetAddress(stream)
	if err != nil {
		if strings.TrimSpace(s.config.TargetAddress) == "" {
			return
		}
		targetAddr = s.config.TargetAddress
	}
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		return
	}
	defer targetConn.Close()

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(targetConn, stream)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(stream, targetConn)
		done <- struct{}{}
	}()
	<-done
}

func (c *Client) serveSOCKS5(ctx context.Context, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
		go c.handleSOCKS5Conn(conn)
	}
}

func (c *Client) handleSOCKS5Conn(conn net.Conn) {
	defer conn.Close()

	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil || header[0] != 0x05 {
		return
	}
	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(conn, methods); err != nil {
		return
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return
	}

	req := make([]byte, 4)
	if _, err := io.ReadFull(conn, req); err != nil {
		return
	}
	if req[0] != 0x05 || req[1] != 0x01 {
		_, _ = conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	target, err := readSOCKS5Target(conn, req[3])
	if err != nil {
		writeSOCKS5Error(conn, 0x08)
		return
	}

	stream, err := c.DialTCP(target)
	if err != nil {
		writeSOCKS5Error(conn, 0x05)
		return
	}
	defer stream.Close()

	if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(stream, conn)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(conn, stream)
		done <- struct{}{}
	}()
	<-done
}

func readSOCKS5Target(r io.Reader, atyp byte) (string, error) {
	var host string
	switch atyp {
	case 0x01:
		ipBuf := make([]byte, 4)
		if _, err := io.ReadFull(r, ipBuf); err != nil {
			return "", err
		}
		host = net.IP(ipBuf).String()
	case 0x03:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			return "", err
		}
		domainBuf := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(r, domainBuf); err != nil {
			return "", err
		}
		host = string(domainBuf)
	case 0x04:
		ipBuf := make([]byte, 16)
		if _, err := io.ReadFull(r, ipBuf); err != nil {
			return "", err
		}
		host = net.IP(ipBuf).String()
	default:
		return "", fmt.Errorf("unsupported address type: %d", atyp)
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, portBuf); err != nil {
		return "", err
	}
	port := int(portBuf[0])<<8 | int(portBuf[1])
	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}

func writeSOCKS5Error(conn net.Conn, code byte) {
	_, _ = conn.Write([]byte{0x05, code, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
}

func (c ServerConfig) domainList() []string {
	var domains []string
	for _, domain := range c.Domains {
		domain = strings.Trim(strings.TrimSpace(domain), ".")
		if domain != "" {
			domains = append(domains, domain)
		}
	}
	if len(domains) == 0 {
		domain := strings.Trim(strings.TrimSpace(c.Domain), ".")
		if domain != "" {
			domains = append(domains, domain)
		}
	}
	return domains
}

func (c ServerConfig) capacityDomain() string {
	domains := c.domainList()
	if len(domains) == 0 {
		return ""
	}
	best := domains[0]
	for _, domain := range domains[1:] {
		if len(domain) > len(best) {
			best = domain
		}
	}
	return best
}
