# Mobile Integration

This repository exposes `slipstream-go/mobile` as the binding surface for
`gomobile bind`.

## Client

```go
client, err := mobile.NewClient(mobile.ClientConfig{
	ResolversCSV:      "127.0.0.1:5353",
	Domain:            "example.com",
	AllowInsecure:     true,
	InitialPacketSize: 1200,
})
if err != nil {
	return err
}
if err := client.Start(); err != nil {
	return err
}
```

Use `DialTCP(target)` to open a remote TCP stream over the tunnel. Use
`StartSOCKS5(listenAddr)` if you need a local proxy for existing apps.

## Server

```go
server, err := mobile.NewServer(mobile.ServerConfig{
	DNSListenAddress: "0.0.0.0:5353",
	Domain:           "example.com",
	CertPEM:          certPEM,
	KeyPEM:           keyPEM,
	ResponseWaitMs:   50,
	PacketQueueSize:  8192,
})
if err != nil {
	return err
}
if err := server.Start(); err != nil {
	return err
}
```

The server returns its bound DNS endpoint through `LocalDNSAddress()`, which is
useful in tests and when binding to an ephemeral port.
