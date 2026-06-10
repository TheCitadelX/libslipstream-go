# Roadmap

This roadmap tracks the Go port of Slipstream and the mobile binding work.

## Done

- [x] Rust-compatible DNS TXT query/response codec.
- [x] Golden-vector tests copied from the Rust implementation.
- [x] DNS payload limit and MTU calculation.
- [x] QUIC transport over DNS using `quic-go`.
- [x] DNS-safe packet fragmentation and reassembly.
- [x] Client and server end-to-end TCP echo test.
- [x] Per-stream target routing.
- [x] Local SOCKS5 proxy for client apps.
- [x] Certificate pinning helpers.
- [x] `slipstream-go/mobile` wrapper package for gomobile-friendly APIs.
- [x] Mobile-layer tests for direct TCP streams and SOCKS5 proxying.

## Next

- [ ] Add `gomobile bind` build scripts for Android.
- [ ] Add `gomobile bind` build scripts for iOS.
- [ ] Add Android smoke test instructions.
- [ ] Add iOS smoke test instructions.
- [ ] Add a minimal Android integration example.
- [ ] Add a minimal iOS integration example.
- [ ] Document production TLS and certificate pinning setup.
- [ ] Add runtime logging hooks suitable for mobile apps.

## Later

- [ ] Add transport benchmarks for DNS fragmentation overhead.
- [ ] Tune resolver failover, retry, and backoff behavior.
- [ ] Add CI for tests and mobile binding compilation.
- [ ] Add tagged release workflow.
- [ ] Compare protocol behavior against `Mygod/slipstream-rust` on real DNS
      resolvers.
- [ ] Review API stability before the first public `v0.x` tag.
