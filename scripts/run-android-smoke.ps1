param(
    [string]$AvdName = "Medium_Phone_API_36.1",
    [int]$BootTimeoutSeconds = 180
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$sdkRoot = if ($env:ANDROID_HOME) { $env:ANDROID_HOME } else { "E:\PROG\AndroidSDK" }
$adb = Join-Path $sdkRoot "platform-tools/adb.exe"
$emulator = Join-Path $sdkRoot "emulator/emulator.exe"
$apk = Join-Path $repoRoot "examples/android-smoke/app/build/outputs/apk/debug/app-debug.apk"
$packageName = "com.thecitadelx.slipstream.smoke"
$activityName = "$packageName/.MainActivity"

if (-not (Test-Path $adb)) {
    throw "adb not found: $adb"
}
if (-not (Test-Path $emulator)) {
    throw "emulator not found: $emulator"
}
if (-not (Test-Path $apk)) {
    & (Join-Path $repoRoot "scripts/build-android-smoke.ps1")
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}

$devices = & $adb devices
$hasDevice = $devices | Select-String -Pattern "device$" -Quiet
if (-not $hasDevice) {
    Start-Process -FilePath $emulator -ArgumentList @("-avd", $AvdName, "-no-snapshot-load") -WindowStyle Hidden
}

$deadline = (Get-Date).AddSeconds($BootTimeoutSeconds)
do {
    Start-Sleep -Seconds 2
    $booted = & $adb shell getprop sys.boot_completed 2>$null
} while (($booted -notcontains "1") -and (Get-Date) -lt $deadline)

if ($booted -notcontains "1") {
    throw "AVD did not boot within $BootTimeoutSeconds seconds"
}

& $adb install -r $apk | Write-Output
& $adb logcat -c
& $adb shell am start -n $activityName | Write-Output
Start-Sleep -Seconds 3

$log = & $adb logcat -d -s SlipstreamSmoke:I '*:S'
$log | Write-Output
if (-not ($log | Select-String -Pattern "SLIPSTREAM_SMOKE_OK" -Quiet)) {
    throw "SLIPSTREAM_SMOKE_OK was not found in logcat"
}
