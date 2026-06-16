# TLS and Certificate Pinning

Production clients should verify the server identity. Do not use
`AllowInsecure` outside local tests.

## Recommended Client Modes

Use one of these client configuration modes:

- `PinnedCertPEM`: pin a specific server certificate.
- `CertFingerprint`: pin a SHA-256 certificate fingerprint.
- `ServerName`: use normal platform certificate verification for a public name.

`AllowInsecure` disables verification and is intended only for tests and local
development.

## Fingerprint Pinning

Set `CertFingerprint` to the server certificate SHA-256 fingerprint in hex.

```go
client, err := mobile.NewClient(mobile.ClientConfig{
	ResolversCSV:      "1.1.1.1:53,8.8.8.8:53",
	Domain:            "example.com",
	CertFingerprint:   "sha256-hex-fingerprint",
	InitialPacketSize: 1200,
})
```

## PEM Pinning

Set `PinnedCertPEM` when the app ships with the exact server certificate:

```go
client, err := mobile.NewClient(mobile.ClientConfig{
	ResolversCSV:      "1.1.1.1:53,8.8.8.8:53",
	Domain:            "example.com",
	PinnedCertPEM:     certPEM,
	InitialPacketSize: 1200,
})
```

## Server Certificates

Servers require `CertPEM` and `KeyPEM`:

```go
server, err := mobile.NewServer(mobile.ServerConfig{
	DNSListenAddress: "0.0.0.0:5353",
	Domain:           "example.com",
	CertPEM:          certPEM,
	KeyPEM:           keyPEM,
	ResponseWaitMs:   50,
	PacketQueueSize:  8192,
})
```

Rotate pinned certificates carefully: clients that pin the old certificate must
receive an app update or a remote configuration update before the server starts
using only the new certificate.
