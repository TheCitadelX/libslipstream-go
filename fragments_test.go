package slipstream

import (
	"bytes"
	"net"
	"testing"
	"time"

	sdns "github.com/TheCitadelX/libslipstream-go/pkg/dns"
)

func TestFragmentRoundTrip(t *testing.T) {
	maxPayload, err := sdns.MaxPayloadLenForDomain("test.com")
	if err != nil {
		t.Fatalf("max payload: %v", err)
	}

	packet := bytes.Repeat([]byte{0xAB}, 1200)
	var f fragmenter
	fragments, err := f.split(packet, maxPayload)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if len(fragments) <= 1 {
		t.Fatalf("expected multiple fragments, got %d", len(fragments))
	}
	for _, fragment := range fragments {
		if len(fragment) > maxPayload {
			t.Fatalf("fragment len = %d, max payload = %d", len(fragment), maxPayload)
		}
	}

	r := newReassembler()
	var full []byte
	for _, fragment := range fragments {
		if assembled := r.ingest(fragment); assembled != nil {
			full = assembled
		}
	}
	if !bytes.Equal(full, packet) {
		t.Fatalf("reassembled packet mismatch")
	}
}

func TestServerPacketConnFragmentsOutgoingPackets(t *testing.T) {
	conn := NewServerPacketConn("test.com", 32)
	peer := serverPeerAddr{
		key:  "127.0.0.1:53000",
		addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 53000},
	}
	packet := bytes.Repeat([]byte{0xCD}, 1200)

	n, err := conn.WriteTo(packet, peer)
	if err != nil {
		t.Fatalf("write to: %v", err)
	}
	if n != len(packet) {
		t.Fatalf("write len = %d, want %d", n, len(packet))
	}

	r := newReassembler()
	var full []byte
	for full == nil {
		fragment := conn.WaitResponse(peer.addr, time.Millisecond)
		if fragment == nil {
			t.Fatalf("expected queued fragment")
		}
		full = r.ingest(fragment)
	}
	if !bytes.Equal(full, packet) {
		t.Fatalf("reassembled outgoing packet mismatch")
	}
}
