package mobile

import (
	"strconv"
	"strings"
	"time"

	core "slipstream-go"
)

type ClientConfig struct {
	ResolversCSV      string
	Domain            string
	CertFingerprint   string
	PinnedCertPEM     []byte
	ServerName        string
	AllowInsecure     bool
	InitialPacketSize int
	SOCKS5ListenAddr  string
}

type ServerConfig struct {
	DNSListenAddress string
	Domain           string
	DomainsCSV       string
	TargetAddress    string
	CertPEM          []byte
	KeyPEM           []byte
	ResponseWaitMs   int
	PacketQueueSize  int
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (c ClientConfig) toCore() core.ClientConfig {
	return core.ClientConfig{
		Resolvers:         splitCSV(c.ResolversCSV),
		Domain:            c.Domain,
		TCPListenAddress:  c.SOCKS5ListenAddr,
		CertFingerprint:   c.CertFingerprint,
		PinnedCertPEM:     c.PinnedCertPEM,
		ServerName:        c.ServerName,
		AllowInsecure:     c.AllowInsecure,
		InitialPacketSize: uint16(max(c.InitialPacketSize, 0)),
	}
}

func (c ServerConfig) toCore() core.ServerConfig {
	return core.ServerConfig{
		DNSListenAddress: c.DNSListenAddress,
		Domain:           c.Domain,
		Domains:          splitCSV(c.DomainsCSV),
		TargetAddress:    c.TargetAddress,
		CertPEM:          c.CertPEM,
		KeyPEM:           c.KeyPEM,
		ResponseWait:     time.Duration(milliseconds(c.ResponseWaitMs)),
		PacketQueueSize:  c.PacketQueueSize,
	}
}

func milliseconds(v int) int64 {
	if v <= 0 {
		return 0
	}
	return int64(v) * int64(1_000_000)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (c ClientConfig) String() string {
	return "ClientConfig{" + c.Domain + "," + strconv.Itoa(len(splitCSV(c.ResolversCSV))) + "}"
}
