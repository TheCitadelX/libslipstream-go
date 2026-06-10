package slipstream

import (
	"bytes"
	"io"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestSocks5ProxyEcho(t *testing.T) {
	certPEM, keyPEM := generateTestCertificate(t)
	echoAddr, closeEcho := startEchoServer(t)
	defer closeEcho()

	server, err := NewServer(ServerConfig{
		DNSListenAddress: "127.0.0.1:0",
		Domain:           "test.com",
		CertPEM:          certPEM,
		KeyPEM:           keyPEM,
		ResponseWait:     50 * time.Millisecond,
		PacketQueueSize:  8192,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer server.Stop()

	dnsAddr := server.udpConn.LocalAddr().String()
	client, err := NewClient(ClientConfig{
		Resolvers:         []string{dnsAddr},
		Domain:            "test.com",
		AllowInsecure:     true,
		InitialPacketSize: 1200,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.Start(); err != nil {
		t.Fatalf("start client: %v", err)
	}
	defer client.Stop()

	proxyAddr, err := client.StartSOCKS5("127.0.0.1:0")
	if err != nil {
		t.Fatalf("start socks5: %v", err)
	}
	defer client.StopSOCKS5()

	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		t.Fatalf("greeting write: %v", err)
	}
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		t.Fatalf("greeting read: %v", err)
	}
	if !bytes.Equal(resp, []byte{0x05, 0x00}) {
		t.Fatalf("greeting resp = %v", resp)
	}

	host, port, err := net.SplitHostPort(echoAddr)
	if err != nil {
		t.Fatalf("split echo addr: %v", err)
	}
	ip := net.ParseIP(host).To4()
	if ip == nil {
		t.Fatalf("expected ipv4 echo addr")
	}
	portNum := parsePort(t, port)
	req := append([]byte{0x05, 0x01, 0x00, 0x01}, ip...)
	req = append(req, byte(portNum>>8), byte(portNum))
	if _, err := conn.Write(req); err != nil {
		t.Fatalf("connect write: %v", err)
	}
	reply := make([]byte, 10)
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatalf("connect read: %v", err)
	}
	if reply[1] != 0x00 {
		t.Fatalf("connect reply code = %d", reply[1])
	}

	want := []byte("socks5 e2e")
	if _, err := conn.Write(want); err != nil {
		t.Fatalf("payload write: %v", err)
	}
	got := make([]byte, len(want))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("payload read: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("echo mismatch: got %q want %q", got, want)
	}
}

func parsePort(t *testing.T, port string) int {
	t.Helper()
	value, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return value
}
