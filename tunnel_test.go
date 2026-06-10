package slipstream

import "testing"

func TestTunnelUsesQUICCompatibleInitialPacketSize(t *testing.T) {
	tunnel, err := NewTunnel(TunnelConfig{
		Resolvers: []string{"127.0.0.1:53"},
		Domain:    "test.com",
		TLSConfig: InsecureTLSConfig(),
	})
	if err != nil {
		t.Fatalf("new tunnel: %v", err)
	}
	config, err := tunnel.quicConfig()
	if err != nil {
		t.Fatalf("quic config: %v", err)
	}
	if config.InitialPacketSize != 1200 {
		t.Fatalf("initial packet size = %d, want 1200", config.InitialPacketSize)
	}
}

func TestTunnelAllowsQUICPacketSizeAboveDNSCapacity(t *testing.T) {
	tunnel, err := NewTunnel(TunnelConfig{
		Resolvers:         []string{"127.0.0.1:53"},
		Domain:            "test.com",
		TLSConfig:         InsecureTLSConfig(),
		InitialPacketSize: 151,
	})
	if err != nil {
		t.Fatalf("new tunnel: %v", err)
	}
	if _, err := tunnel.quicConfig(); err != nil {
		t.Fatalf("quic config: %v", err)
	}
}

func TestClientConfigResolverFallback(t *testing.T) {
	config := ClientConfig{
		ResolverAddress: "1.1.1.1",
		Domain:          "test.com",
		AllowInsecure:   true,
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got := config.resolverList(); len(got) != 1 || got[0] != "1.1.1.1" {
		t.Fatalf("resolver list = %#v", got)
	}
	if _, err := config.tlsConfig(); err != nil {
		t.Fatalf("tls config: %v", err)
	}
}
