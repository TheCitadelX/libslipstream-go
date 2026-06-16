package slipstream

import (
	"bytes"
	"testing"

	sdns "github.com/TheCitadelX/libslipstream-go/pkg/dns"
)

func BenchmarkFragmenterSplit(b *testing.B) {
	maxPayload := benchmarkMaxPayload(b)
	packet := bytes.Repeat([]byte{0xAB}, 1200)
	var f fragmenter

	b.ReportAllocs()
	b.SetBytes(int64(len(packet)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fragments, err := f.split(packet, maxPayload)
		if err != nil {
			b.Fatal(err)
		}
		if len(fragments) == 0 {
			b.Fatal("expected fragments")
		}
	}
}

func BenchmarkFragmenterSplitEncodeDNSQueries(b *testing.B) {
	const domain = "test.com"
	maxPayload := benchmarkMaxPayload(b)
	packet := bytes.Repeat([]byte{0xCD}, 1200)
	var f fragmenter

	b.ReportAllocs()
	b.SetBytes(int64(len(packet)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fragments, err := f.split(packet, maxPayload)
		if err != nil {
			b.Fatal(err)
		}
		for seq, fragment := range fragments {
			query, _, err := sdns.EncodePayloadQuery(uint16(seq), fragment, domain)
			if err != nil {
				b.Fatal(err)
			}
			if len(query) == 0 {
				b.Fatal("expected query")
			}
		}
	}
}

func BenchmarkFragmenterSplitReassemble(b *testing.B) {
	maxPayload := benchmarkMaxPayload(b)
	packet := bytes.Repeat([]byte{0xEF}, 1200)
	var f fragmenter

	b.ReportAllocs()
	b.SetBytes(int64(len(packet)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fragments, err := f.split(packet, maxPayload)
		if err != nil {
			b.Fatal(err)
		}
		r := newReassembler()
		var full []byte
		for _, fragment := range fragments {
			if assembled := r.ingest(fragment); assembled != nil {
				full = assembled
			}
		}
		if len(full) != len(packet) {
			b.Fatalf("reassembled len = %d, want %d", len(full), len(packet))
		}
	}
}

func benchmarkMaxPayload(b *testing.B) int {
	b.Helper()
	maxPayload, err := sdns.MaxPayloadLenForDomain("test.com")
	if err != nil {
		b.Fatalf("max payload: %v", err)
	}
	return maxPayload
}
