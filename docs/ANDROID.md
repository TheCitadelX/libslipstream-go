# Android Smoke Test

Build the AAR:

```sh
./scripts/build-android.sh
```

On Windows:

```powershell
.\scripts\build-android.ps1
```

The default output is `build/android/libslipstream.aar`.

## Import

1. Add the AAR to your Android app project.
2. Add it as a dependency in the app module.
3. Make sure the app has network permissions:

```xml
<uses-permission android:name="android.permission.INTERNET" />
```

## Minimal Runtime Check

Create a client, start it, and expose a local SOCKS5 proxy:

```java
ClientConfig config = new ClientConfig();
config.setResolversCSV("1.1.1.1:53,8.8.8.8:53");
config.setDomain("example.com");
config.setCertFingerprint("sha256-hex-fingerprint");
config.setInitialPacketSize(1200);

Client client = Mobile.newClient(config);
client.start();
String proxyAddress = client.startSOCKS5("127.0.0.1:0");
```

The smoke test passes when the app receives a non-empty `proxyAddress` and a
TCP client can connect through that SOCKS5 endpoint.
