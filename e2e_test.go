package slipstream

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
	"testing"
	"time"
)

func TestEndToEndEcho(t *testing.T) {
	certPEM, keyPEM := generateTestCertificate(t)

	echoAddr1, closeEcho1 := startEchoServer(t)
	defer closeEcho1()
	echoAddr2, closeEcho2 := startEchoServer(t)
	defer closeEcho2()

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

	runEchoRoundTrip := func(target string, payload string) {
		tcpStream, err := client.DialTCP(target)
		if err != nil {
			t.Fatalf("dial tcp: %v", err)
		}
		defer tcpStream.Close()

		want := []byte(payload)
		if _, err := tcpStream.Write(want); err != nil {
			t.Fatalf("write stream: %v", err)
		}

		got := make([]byte, len(want))
		if _, err := io.ReadFull(tcpStream, got); err != nil {
			t.Fatalf("read stream: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("echo mismatch: got %q want %q", got, want)
		}
	}

	runEchoRoundTrip(echoAddr1, "slipstream-go e2e one")
	runEchoRoundTrip(echoAddr2, "slipstream-go e2e two")
}

func startEchoServer(t *testing.T) (string, func()) {
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

func generateTestCertificate(t *testing.T) ([]byte, []byte) {
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
			CommonName: "slipstream-go-test",
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
