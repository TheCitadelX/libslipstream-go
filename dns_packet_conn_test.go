package slipstream

import (
	"bytes"
	"net"
	"testing"
	"time"

	sdns "github.com/TheCitadelX/libslipstream-go/pkg/dns"
)

func TestDNSPacketConnKeepsFragmentsOnOneResolver(t *testing.T) {
	resolver1 := listenUDPTest(t)
	defer resolver1.Close()
	resolver2 := listenUDPTest(t)
	defer resolver2.Close()

	conn, err := NewDNSPacketConn(DNSPacketConnConfig{
		Resolvers:     []string{resolver1.LocalAddr().String(), resolver2.LocalAddr().String()},
		Domain:        "test.com",
		PollInterval:  time.Hour,
		IdleThreshold: time.Hour,
	})
	if err != nil {
		t.Fatalf("new packet conn: %v", err)
	}
	defer conn.Close()

	packet := bytes.Repeat([]byte{0xAB}, 1200)
	expectedFragments := expectedDNSFragments(t, packet, "test.com")
	if _, err := conn.WriteTo(packet, nil); err != nil {
		t.Fatalf("write first packet: %v", err)
	}
	firstCounts := readResolverPackets(t, expectedFragments, resolver1, resolver2)
	if firstCounts[0] != expectedFragments || firstCounts[1] != 0 {
		t.Fatalf("first packet fragments split across resolvers: got %v want [%d 0]", firstCounts, expectedFragments)
	}

	if _, err := conn.WriteTo(packet, nil); err != nil {
		t.Fatalf("write second packet: %v", err)
	}
	secondCounts := readResolverPackets(t, expectedFragments, resolver1, resolver2)
	if secondCounts[0] != 0 || secondCounts[1] != expectedFragments {
		t.Fatalf("second packet fragments split across resolvers: got %v want [0 %d]", secondCounts, expectedFragments)
	}
}

func listenUDPTest(t *testing.T) *net.UDPConn {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("resolve udp: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	return conn
}

func expectedDNSFragments(t *testing.T, packet []byte, domain string) int {
	t.Helper()
	maxPayload, err := sdns.MaxPayloadLenForDomain(domain)
	if err != nil {
		t.Fatalf("max payload: %v", err)
	}
	var f fragmenter
	fragments, err := f.split(packet, maxPayload)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	return len(fragments)
}

func readResolverPackets(t *testing.T, wantTotal int, resolvers ...*net.UDPConn) []int {
	t.Helper()
	counts := make([]int, len(resolvers))
	buf := make([]byte, 2048)
	deadline := time.Now().Add(time.Second)
	for sum(counts) < wantTotal && time.Now().Before(deadline) {
		for i, resolver := range resolvers {
			_ = resolver.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			n, _, err := resolver.ReadFromUDP(buf)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				t.Fatalf("read resolver: %v", err)
			}
			if n > 0 {
				counts[i]++
			}
		}
	}
	return counts
}

func sum(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}
