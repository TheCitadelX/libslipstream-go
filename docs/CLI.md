# CLI

The repository includes small command-line tools for local and VPS testing.

Build them:

```sh
go build ./cmd/slipstream-server
go build ./cmd/slipstream-client
go build ./cmd/slipstream-cert
```

## Certificate Helper

For local or VPS smoke tests, generate a self-signed certificate and copy the
printed SHA-256 fingerprint into the client command:

```sh
slipstream-cert -hosts tunnel.example.com,203.0.113.10 -cert server.crt -key server.key
```

## Server

```sh
slipstream-server \
  -dns :53 \
  -domain tunnel.example.com \
  -cert server.crt \
  -key server.key
```

Useful flags:

- `-dns`: UDP DNS listen address.
- `-domain`: primary tunnel domain.
- `-domains`: comma-separated alternate tunnel domains.
- `-target`: optional fallback TCP target for legacy clients.
- `-response-wait`: time to wait for queued QUIC packets before replying to a
  DNS query.
- `-queue`: packet queue size.

## Client

```sh
slipstream-client \
  -resolver 203.0.113.10:53 \
  -domain tunnel.example.com \
  -cert-fingerprint sha256-hex-fingerprint \
  -socks 127.0.0.1:1080
```

Useful flags:

- `-resolver`: one DNS resolver address.
- `-resolvers`: comma-separated resolver addresses.
- `-domain`: tunnel domain.
- `-socks`: local SOCKS5 listen address.
- `-cert-fingerprint`: SHA-256 certificate fingerprint pin.
- `-pinned-cert`: PEM certificate pin file.
- `-server-name`: TLS server name for platform certificate verification.
- `-allow-insecure`: disables TLS verification for local testing only.

The client starts a local SOCKS5 proxy and keeps running until Ctrl+C.
