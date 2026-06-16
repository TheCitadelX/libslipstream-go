param(
    [string]$Output = "build/android/libslipstream.aar",
    [string]$Target = "android",
    [string]$JavaPackage = "com.thecitadelx.slipstream",
    [int]$AndroidAPI = 21
)

$ErrorActionPreference = "Stop"

$gomobile = Get-Command gomobile -ErrorAction SilentlyContinue
if (-not $gomobile) {
    throw "gomobile is not installed. Run: go install golang.org/x/mobile/cmd/gomobile@latest; gomobile init"
}

if (-not $env:ANDROID_HOME -and (Test-Path "E:\PROG\AndroidSDK")) {
    $env:ANDROID_HOME = "E:\PROG\AndroidSDK"
}
if (-not $env:ANDROID_SDK_ROOT -and $env:ANDROID_HOME) {
    $env:ANDROID_SDK_ROOT = $env:ANDROID_HOME
}
if (-not $env:ANDROID_NDK_HOME -and $env:ANDROID_HOME) {
    $ndkRoot = Join-Path $env:ANDROID_HOME "ndk"
    if (Test-Path $ndkRoot) {
        $latestNdk = Get-ChildItem -Path $ndkRoot -Directory |
            Sort-Object Name -Descending |
            Select-Object -First 1
        if ($latestNdk) {
            $env:ANDROID_NDK_HOME = $latestNdk.FullName
        }
    }
}

$outputDir = Split-Path -Parent $Output
if ($outputDir) {
    New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
}

$args = @(
    "bind",
    "-target=$Target",
    "-androidapi=$AndroidAPI",
    "-javapkg=$JavaPackage",
    "-trimpath",
    "-o",
    $Output,
    "./mobile"
)

& gomobile @args
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
