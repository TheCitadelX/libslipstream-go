param(
    [string]$Output = "build/android/libslipstream.aar",
    [string]$Target = "android",
    [string]$JavaPackage = "com.thecitadelx.slipstream"
)

$ErrorActionPreference = "Stop"

$gomobile = Get-Command gomobile -ErrorAction SilentlyContinue
if (-not $gomobile) {
    throw "gomobile is not installed. Run: go install golang.org/x/mobile/cmd/gomobile@latest; gomobile init"
}

$outputDir = Split-Path -Parent $Output
if ($outputDir) {
    New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
}

$args = @(
    "bind",
    "-target=$Target",
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
