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
- [x] Local Android AAR compilation with generated Java API inspection.
- [x] Gradle-based Android smoke app for AVD testing.
- [x] Minimal Android integration example.
- [x] Runtime event queue suitable for mobile apps.
- [x] Android smoke test instructions.
- [x] iOS smoke test instructions.
- [x] Production TLS and certificate pinning documentation.
- [x] CI for Go tests.
- [x] Transport benchmarks for DNS fragmentation overhead.
- [x] CI workflow for Android mobile binding compilation.

## Next

- [ ] Add a minimal iOS integration example.

## Later

- [ ] Tune resolver failover, retry, and backoff behavior.
- [ ] Add tagged release workflow.
- [ ] Compare protocol behavior against `Mygod/slipstream-rust` on real DNS
      resolvers.
- [ ] Review API stability before the first public `v0.x` tag.
