# libslipstream-go

Go port of the Slipstream DNS tunnel protocol, based on:

- <https://github.com/Mygod/slipstream-rust>
- <https://github.com/EndPositive/slipstream>

The port is being shaped as a small Go library first so it can later be wrapped
with `gomobile bind` for Android and iOS.

## Current status

- DNS TXT query/response codec compatible with the Rust golden vectors.
- RFC4648 base32 without padding plus Slipstream inline-dot formatting.
- Stream receive chunk insertion helper ported from `slipstream-core`.
- Rust-compatible DNS payload limits and MTU calculation.
- Client-side `net.PacketConn` that carries QUIC packets over DNS queries.
- Mobile-friendly client API backed by `quic-go`.
- Server-side DNS listener and virtual `PacketConn` backed by `quic-go`.
- Local client/server/TCP echo end-to-end test.
- Client streams now carry a per-stream target address, so one tunnel can talk
  to multiple TCP targets.
- Client can expose a local SOCKS5 proxy for apps that expect a standard proxy.
- A separate `github.com/TheCitadelX/libslipstream-go/mobile` package wraps the
  core API in gomobile-friendly types.

The client runtime now follows a hybrid strategy: keep Rust wire compatibility as
the source of truth, while borrowing practical transport ideas from
`minor-way/slipstream-go` such as pure-Go `quic-go`, multi-resolver support, and
health checks. Because `quic-go` requires 1200-byte Initial packets, QUIC packets
are fragmented into DNS-safe payloads before being encoded as Slipstream DNS
queries.

## Mobile API

Use the `github.com/TheCitadelX/libslipstream-go/mobile` package as the binding
surface for Android and iOS. It exposes string/byte/int config structs plus
simple `Start`, `Stop`, `DialTCP`, and `StartSOCKS5` methods.

```go
import "github.com/TheCitadelX/libslipstream-go/mobile"

client, err := mobile.NewClient(mobile.ClientConfig{
	ResolversCSV:      "127.0.0.1:5353",
	Domain:            "test.com",
	AllowInsecure:     true,
	InitialPacketSize: 1200,
})
```

Android bindings:

```sh
./scripts/build-android.sh
```

iOS bindings must be built on macOS with Xcode:

```sh
./scripts/build-ios.sh
```

## Verify

```sh
go test ./...
```

## Docs

- [Mobile integration](docs/MOBILE.md)
- [Android smoke test](docs/ANDROID.md)
- [iOS smoke test](docs/IOS.md)
- [TLS and certificate pinning](docs/TLS.md)
- [Roadmap](docs/ROADMAP.md)
