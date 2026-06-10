package slipstream

import (
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"
)

const fragmentHeaderLen = 4

type fragmenter struct {
	nextID atomic.Uint32
}

func (f *fragmenter) split(packet []byte, maxPayload int) ([][]byte, error) {
	maxChunk := maxPayload - fragmentHeaderLen
	if maxChunk <= 0 {
		return nil, errConfig("dns payload capacity is too small for fragmentation")
	}
	total := (len(packet) + maxChunk - 1) / maxChunk
	if total > 255 {
		return nil, errConfig("packet requires too many DNS fragments")
	}
	packetID := uint16(f.nextID.Add(1))
	fragments := make([][]byte, 0, total)
	for seq := 0; seq < total; seq++ {
		start := seq * maxChunk
		end := start + maxChunk
		if end > len(packet) {
			end = len(packet)
		}
		fragment := make([]byte, fragmentHeaderLen+end-start)
		binary.BigEndian.PutUint16(fragment[0:2], packetID)
		fragment[2] = byte(total)
		fragment[3] = byte(seq)
		copy(fragment[4:], packet[start:end])
		fragments = append(fragments, fragment)
	}
	return fragments, nil
}

type reassembler struct {
	mu        sync.Mutex
	pending   map[uint16]*pendingFragments
	completed map[uint16]time.Time
}

type pendingFragments struct {
	chunks   [][]byte
	received int
	created  time.Time
}

func newReassembler() *reassembler {
	return &reassembler{
		pending:   make(map[uint16]*pendingFragments),
		completed: make(map[uint16]time.Time),
	}
}

func (r *reassembler) ingest(fragment []byte) []byte {
	if len(fragment) < fragmentHeaderLen {
		return nil
	}
	packetID := binary.BigEndian.Uint16(fragment[0:2])
	total := int(fragment[2])
	seq := int(fragment[3])
	if total == 0 || seq >= total {
		return nil
	}
	payload := fragment[4:]

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for id, completedAt := range r.completed {
		if now.Sub(completedAt) > 30*time.Second {
			delete(r.completed, id)
		}
	}
	if _, ok := r.completed[packetID]; ok {
		return nil
	}
	if len(r.pending) > 4096 {
		r.pending = make(map[uint16]*pendingFragments)
	}

	pending := r.pending[packetID]
	if pending == nil {
		pending = &pendingFragments{
			chunks:  make([][]byte, total),
			created: now,
		}
		r.pending[packetID] = pending
	}
	if len(pending.chunks) != total {
		return nil
	}
	if pending.chunks[seq] == nil {
		pending.chunks[seq] = append([]byte(nil), payload...)
		pending.received++
	}
	if pending.received != total {
		return nil
	}

	delete(r.pending, packetID)
	r.completed[packetID] = now
	var packet []byte
	for _, chunk := range pending.chunks {
		packet = append(packet, chunk...)
	}
	return packet
}
