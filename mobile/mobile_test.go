package mobile

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestMobileClientServerDialTCP(t *testing.T) {
	echoAddr, closeEcho := startMobileEchoServer(t)
	defer closeEcho()

	client := startMobileTunnel(t)
	stream, err := client.DialTCP(echoAddr)
	if err != nil {
		t.Fatalf("dial tcp: %v", err)
	}
	defer stream.Close()

	want := []byte("mobile dial tcp")
	if _, err := stream.Write(want); err != nil {
		t.Fatalf("write stream: %v", err)
	}

	got := make([]byte, len(want))
	if _, err := io.ReadFull(stream, got); err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("echo mismatch: got %q want %q", got, want)
	}
}

func TestMobileSOCKS5ProxyEcho(t *testing.T) {
	echoAddr, closeEcho := startMobileEchoServer(t)
	defer closeEcho()

	client := startMobileTunnel(t)
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
	portNum := parseMobilePort(t, port)
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

	want := []byte("mobile socks5")
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

func startMobileTunnel(t *testing.T) *Client {
	t.Helper()
	certPEM, keyPEM := generateMobileTestCertificate(t)
	server, err := NewServer(&ServerConfig{
		DNSListenAddress: "127.0.0.1:0",
		Domain:           "test.com",
		CertPEM:          certPEM,
		KeyPEM:           keyPEM,
		ResponseWaitMs:   50,
		PacketQueueSize:  8192,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() { _ = server.Stop() })

	dnsAddr := server.LocalDNSAddress()
	if dnsAddr == "" {
		t.Fatalf("empty server DNS address")
	}
	client, err := NewClient(&ClientConfig{
		ResolversCSV:      dnsAddr,
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
	t.Cleanup(func() { _ = client.Stop() })
	return client
}

func startMobileEchoServer(t *testing.T) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen echo: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()
	return listener.Addr().String(), func() {
		_ = listener.Close()
		<-done
	}
}

func generateMobileTestCertificate(t *testing.T) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("generate serial: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "slipstream-go-mobile-test",
		},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}

func parseMobilePort(t *testing.T, port string) int {
	t.Helper()
	value, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return value
}
