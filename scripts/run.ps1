param(
    [string]$Listen = "127.0.0.1:8787",
    [string]$DataDir
)

$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
$SdkRoot = Join-Path $RootDir ".tools\android-sdk"

if (Test-Path $SdkRoot) {
    $env:ANDROID_HOME = $SdkRoot
    $env:ANDROID_SDK_ROOT = $SdkRoot
    $env:ANDROID_AVD_HOME = Join-Path $RootDir ".tools\avd"
    $env:Path = "$(Join-Path $SdkRoot 'platform-tools');$(Join-Path $SdkRoot 'emulator');$env:Path"
}

$EnvFile = Join-Path $RootDir ".env"
if (Test-Path $EnvFile) {
    Get-Content $EnvFile | ForEach-Object {
        $Line = $_.Trim()
        if ($Line -and -not $Line.StartsWith("#") -and $Line.Contains("=")) {
            $Name, $Value = $Line.Split("=", 2)
            [Environment]::SetEnvironmentVariable($Name.Trim(), $Value.Trim(), "Process")
        }
    }
}

if (-not $DataDir) {
    $DataDir = Join-Path $RootDir "data"
}

$Executable = Join-Path $RootDir "bin\sanguoclip.exe"
if (Test-Path $Executable) {
    & $Executable -addr $Listen -data-dir $DataDir
    exit $LASTEXITCODE
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "SanguoClip is not built and Go is missing. Run scripts/setup.ps1 first."
}

Push-Location $RootDir
try {
    & go run ./cmd/sanguoclip -addr $Listen -data-dir $DataDir
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
