# Android Smoke App

This is a minimal Android app that verifies the generated AAR can be loaded on
an emulator. It does not connect to a real Slipstream server; it checks the
Java binding, native library loading, `Mobile.newClient(...)`, and the mobile
runtime event queue.

Build the APK:

```powershell
.\scripts\build-android-smoke.ps1
```

Run it on the default local AVD:

```powershell
.\scripts\run-android-smoke.ps1 -AvdName 'Medium_Phone_API_36.1'
```

The smoke test passes when logcat contains:

```text
SLIPSTREAM_SMOKE_OK connected=false event=info:client created
```
