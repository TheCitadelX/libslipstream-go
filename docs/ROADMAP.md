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
- [x] `github.com/TheCitadelX/libslipstream-go/mobile` wrapper package for
      gomobile-friendly APIs.
- [x] Mobile-layer tests for direct TCP streams and SOCKS5 proxying.
- [x] `gomobile bind` build scripts for Android.
- [x] `gomobile bind` build scripts for iOS.
- [x] Android smoke test instructions.
- [x] iOS smoke test instructions.
- [x] Production TLS and certificate pinning documentation.
- [x] CI for Go tests.

## Next

- [ ] Add a minimal Android integration example.
- [ ] Add a minimal iOS integration example.
- [ ] Add runtime logging hooks suitable for mobile apps.

## Later

- [ ] Add transport benchmarks for DNS fragmentation overhead.
- [ ] Tune resolver failover, retry, and backoff behavior.
- [ ] Add CI for mobile binding compilation.
- [ ] Add tagged release workflow.
- [ ] Compare protocol behavior against `Mygod/slipstream-rust` on real DNS
      resolvers.
- [ ] Review API stability before the first public `v0.x` tag.
