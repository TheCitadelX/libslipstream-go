# iOS Smoke Test

iOS bindings must be built on macOS with Xcode installed.

Build the framework:

```sh
./scripts/build-ios.sh
```

The default output is `build/ios/Libslipstream.framework`.

## Import

1. Add `Libslipstream.framework` to the Xcode project.
2. Embed and sign the framework in the app target.
3. Enable network access in the app as needed.

## Minimal Runtime Check

The generated Objective-C names depend on the gomobile prefix. This repository
uses `Slipstream` by default. Treat the snippet below as the expected shape and
check the generated framework header for exact Swift names:

```swift
let config = SlipstreamClientConfig()
config.resolversCSV = "1.1.1.1:53,8.8.8.8:53"
config.domain = "example.com"
config.certFingerprint = "sha256-hex-fingerprint"
config.initialPacketSize = 1200

let client = try SlipstreamNewClient(config)
try client.start()
let proxyAddress = try client.startSOCKS5("127.0.0.1:0")
```

The smoke test passes when the app receives a non-empty `proxyAddress` and a
TCP client can connect through that SOCKS5 endpoint.
