param(
    [string]$Output = ".\bin\sanguoclip.exe"
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go was not found in PATH. Install Go 1.26 or newer first."
}

$outputDirectory = Split-Path -Parent $Output
if ($outputDirectory) {
    New-Item -ItemType Directory -Force -Path $outputDirectory | Out-Null
}

go build -o $Output ./cmd/sanguoclip
Write-Host "Built $Output"
