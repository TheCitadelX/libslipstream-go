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

For the local Windows SDK at `E:\PROG\AndroidSDK`, the PowerShell script
auto-detects the SDK and latest installed NDK when the Android environment
variables are not already set.

For an x86_64 emulator AVD:

```powershell
.\scripts\build-android.ps1 -Target android/amd64 -Output build/android/libslipstream-amd64.aar
```

Inspect the generated classes:

```powershell
.\scripts\inspect-android-aar.ps1 -AAR build\android\libslipstream-amd64.aar
```

Build and run the smoke APK on the default local AVD:

```powershell
.\scripts\build-android-smoke.ps1
.\scripts\run-android-smoke.ps1 -AvdName 'Medium_Phone_API_36.1'
```

The runtime smoke test passes when logcat contains:

```text
SLIPSTREAM_SMOKE_OK connected=false event=info:client created
```

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
import com.thecitadelx.slipstream.mobile.Client;
import com.thecitadelx.slipstream.mobile.ClientConfig;
import com.thecitadelx.slipstream.mobile.Mobile;

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

## Runtime Events

The mobile API exposes a non-blocking event queue for app logs and status
updates:

```java
config.setEventQueueSize(128);
Client client = Mobile.newClient(config);
Event event = client.events().next(1000);
if (event != null) {
    Log.i("Slipstream", event.getLevel() + " " + event.getMessage());
}
```

Use a background thread or coroutine when waiting with a non-zero timeout.
