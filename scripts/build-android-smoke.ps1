param(
    [string]$GradleVersion = "9.5.1",
    [string]$AndroidTarget = "android/amd64",
    [string]$AAR = "build/android/libslipstream-smoke.aar"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$smokeRoot = Join-Path $repoRoot "examples/android-smoke"
$localGradleRoot = Join-Path $repoRoot ".gradle-local"
$gradleHome = Join-Path $localGradleRoot "gradle-$GradleVersion"
$gradleBat = Join-Path $gradleHome "bin/gradle.bat"

& (Join-Path $repoRoot "scripts/build-android.ps1") `
    -Target $AndroidTarget `
    -Output (Join-Path $repoRoot $AAR)
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

$libsDir = Join-Path $smokeRoot "app/libs"
New-Item -ItemType Directory -Force -Path $libsDir | Out-Null
Copy-Item -Force -Path (Join-Path $repoRoot $AAR) -Destination (Join-Path $libsDir "libslipstream.aar")

if (-not (Test-Path $gradleBat)) {
    New-Item -ItemType Directory -Force -Path $localGradleRoot | Out-Null
    $zip = Join-Path $localGradleRoot "gradle-$GradleVersion-bin.zip"
    if (-not (Test-Path $zip)) {
        $url = "https://services.gradle.org/distributions/gradle-$GradleVersion-bin.zip"
        Invoke-WebRequest -Uri $url -OutFile $zip
    }
    Expand-Archive -Force -Path $zip -DestinationPath $localGradleRoot
}

$env:ANDROID_HOME = if ($env:ANDROID_HOME) { $env:ANDROID_HOME } else { "E:\PROG\AndroidSDK" }
$env:ANDROID_SDK_ROOT = $env:ANDROID_HOME

Push-Location $smokeRoot
try {
    & $gradleBat --no-daemon :app:assembleDebug
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}
finally {
    Pop-Location
}
