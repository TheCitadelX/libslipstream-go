param(
    [string]$AAR = "build/android/libslipstream.aar"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $AAR)) {
    throw "AAR not found: $AAR"
}
if (-not (Get-Command jar -ErrorAction SilentlyContinue)) {
    throw "jar is not available. Install a JDK and make sure it is on PATH."
}
if (-not (Get-Command javap -ErrorAction SilentlyContinue)) {
    throw "javap is not available. Install a JDK and make sure it is on PATH."
}

$aarPath = Resolve-Path $AAR
$tmp = Join-Path $env:TEMP ("libslipstream-aar-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
try {
    Push-Location $tmp
    jar xf $aarPath classes.jar
    Write-Output "AAR entries:"
    jar tf $aarPath
    Write-Output ""
    Write-Output "Java API:"
    javap -classpath classes.jar `
        com.thecitadelx.slipstream.mobile.Mobile `
        com.thecitadelx.slipstream.mobile.ClientConfig `
        com.thecitadelx.slipstream.mobile.Client `
        com.thecitadelx.slipstream.mobile.ServerConfig `
        com.thecitadelx.slipstream.mobile.Server
}
finally {
    Pop-Location
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
