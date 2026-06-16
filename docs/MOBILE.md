# Mobile Integration

This repository exposes `github.com/TheCitadelX/libslipstream-go/mobile` as the
binding surface for `gomobile bind`.

## Toolchain

Install and initialize gomobile once:

```sh
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
```

Android builds require a JDK plus Android SDK/NDK. On Windows, the PowerShell
script auto-detects `E:\PROG\AndroidSDK` when `ANDROID_HOME` is not set. iOS
builds must run on macOS with Xcode installed.

## Build Android

PowerShell:

```powershell
.\scripts\build-android.ps1
```

POSIX shell:

```sh
./scripts/build-android.sh
```

The default output is `build/android/libslipstream.aar` and the default Android
API level is 21. Override the defaults with script parameters or environment
variables:

```sh
OUTPUT=build/android/slipstream-arm64.aar TARGET=android/arm64 ./scripts/build-android.sh
```

For an x86_64 AVD build on Windows:

```powershell
.\scripts\build-android.ps1 -Target android/amd64 -Output build/android/libslipstream-amd64.aar
```

## Build iOS

Run this on macOS:

```sh
./scripts/build-ios.sh
```

The default output is `build/ios/Libslipstream.framework`.

## Client

```go
import "github.com/TheCitadelX/libslipstream-go/mobile"

client, err := mobile.NewClient(&mobile.ClientConfig{
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
Use `Events()` to poll app-friendly runtime events without wiring Go callbacks
through gomobile.

Android apps can use the generated `mobile.ClientConfig` and `mobile.Client`
classes from the AAR. A typical integration starts the tunnel and then exposes a
local SOCKS5 proxy:

```java
ClientConfig config = new ClientConfig();
config.setResolversCSV("1.1.1.1:53,8.8.8.8:53");
config.setDomain("example.com");
config.setAllowInsecure(false);
config.setCertFingerprint("sha256-hex-fingerprint");
config.setInitialPacketSize(1200);
config.setEventQueueSize(128);

Client client = Mobile.newClient(config);
client.start();
String proxy = client.startSOCKS5("127.0.0.1:0");
Event event = client.events().next(1000);
```

`EventQueue.next(timeoutMs)` returns immediately for `0`, waits forever for a
negative timeout, or waits up to the given timeout in milliseconds.

## Server

```go
import "github.com/TheCitadelX/libslipstream-go/mobile"

server, err := mobile.NewServer(&mobile.ServerConfig{
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

Swift apps use the generated Objective-C framework. The exact type prefix
depends on the `PREFIX` value passed to `scripts/build-ios.sh`; the default is
`Slipstream`.
