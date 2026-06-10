package dns

import "testing"

func TestMaxPayloadLenForDomainAndBuildQName(t *testing.T) {
	maxPayload, err := MaxPayloadLenForDomain("test.com")
	if err != nil {
		t.Fatalf("max payload: %v", err)
	}
	if maxPayload != 150 {
		t.Fatalf("max payload = %d, want 150", maxPayload)
	}

	payload := make([]byte, maxPayload)
	if _, err := BuildQName(payload, "test.com"); err != nil {
		t.Fatalf("build max qname: %v", err)
	}
	if _, err := BuildQName(append(payload, 0), "test.com"); err == nil {
		t.Fatalf("expected payload overflow error")
	}
}

func TestComputeMTU(t *testing.T) {
	mtu, err := ComputeMTU("test.com")
	if err != nil {
		t.Fatalf("compute mtu: %v", err)
	}
	if mtu != 145 {
		t.Fatalf("mtu = %d, want 145", mtu)
	}
}
