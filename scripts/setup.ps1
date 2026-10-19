param(
    [switch]$WithEmulator,
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
$ToolsDir = Join-Path $RootDir ".tools"
. (Join-Path $PSScriptRoot "ldplayer.ps1")

function Update-ProcessPath {
    $MachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $env:Path = "$MachinePath;$UserPath;$env:Path"
}

function Install-WingetPackage {
    param(
        [string]$Id,
        [string]$Command
    )

    if ($Command -and (Get-Command $Command -ErrorAction SilentlyContinue)) {
        Write-Host "[ok] $Id"
        return
    }

    $Installed = & winget list --exact --id $Id --accept-source-agreements 2>$null
    if ($LASTEXITCODE -eq 0 -and $Installed) {
        Write-Host "[ok] $Id"
        Update-ProcessPath
        return
    }

    & winget install --exact --id $Id --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -ne 0) {
        throw "winget failed to install $Id"
    }
    Update-ProcessPath
}

function Install-LdPlayer {
    $LdConsole = Find-LdConsole
    if ($LdConsole) {
        Write-Host "[ok] LDPlayer: $LdConsole"
        return $LdConsole
    }

    $DownloadsDir = Join-Path $ToolsDir "downloads"
    $Installer = Join-Path $DownloadsDir "ldplayer-sanguosha.exe"
    New-Item -ItemType Directory -Force -Path $DownloadsDir | Out-Null

    Write-Host "Downloading the official Sanguosha LDPlayer installer..."
    $Headers = @{
        "Referer" = $script:LdPlayerGamePage
        "User-Agent" = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/140 Safari/537.36"
    }
    $InstallerUrl = Get-LdPlayerGameInstallerUrl
    Invoke-WebRequest -UseBasicParsing -Uri $InstallerUrl -Headers $Headers -OutFile $Installer

    $Signature = Get-AuthenticodeSignature $Installer
    if ($Signature.Status -ne "Valid") {
        throw "LDPlayer installer signature is not valid: $($Signature.Status)"
    }

    Write-Host "Complete the official LDPlayer installer. Setup will continue afterwards."
    Start-Process -FilePath $Installer -Wait
    Update-ProcessPath

    $Deadline = (Get-Date).AddMinutes(5)
    do {
        $LdConsole = Find-LdConsole
        if ($LdConsole) {
            return $LdConsole
        }
        Start-Sleep -Seconds 2
    } while ((Get-Date) -lt $Deadline)

    throw "LDPlayer installation was not detected. Finish the installer and rerun setup.ps1 -WithEmulator."
}

if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    throw "winget is required. Install or update App Installer from Microsoft Store."
}

New-Item -ItemType Directory -Force -Path $ToolsDir | Out-Null

Install-WingetPackage -Id "GoLang.Go" -Command "go"
Install-WingetPackage -Id "Gyan.FFmpeg" -Command "ffmpeg"
Install-WingetPackage -Id "Google.PlatformTools" -Command "adb"

$MissingCommands = @(
    "go",
    "ffmpeg",
    "ffprobe",
    "adb"
) | Where-Object { -not (Get-Command $_ -ErrorAction SilentlyContinue) }
if ($MissingCommands.Count -gt 0) {
    throw "Installed dependencies are not available in PATH: $($MissingCommands -join ', '). Close PowerShell, open it again, and rerun setup.ps1."
}

if ($WithEmulator) {
    if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq [System.Runtime.InteropServices.Architecture]::Arm64) {
        throw "LDPlayer supports Windows on Intel/AMD processors only. Use a physical Android device on Windows ARM64."
    }
    $LdConsole = Install-LdPlayer

    & $LdConsole rename --index 0 --title "SanguoClip"
    & $LdConsole modify `
        --index 0 `
        --resolution "1920,1080,280" `
        --cpu 4 `
        --memory 4096 `
        --autorotate 1 `
        --root 0
    if ($LASTEXITCODE -ne 0) {
        throw "LDPlayer configuration failed."
    }
    Write-Host "[ok] LDPlayer instance 0 configured for SanguoClip"
}

$EnvFile = Join-Path $RootDir ".env"
if (-not (Test-Path $EnvFile)) {
    Copy-Item (Join-Path $RootDir ".env.example") $EnvFile
    Write-Host "Created .env from .env.example"
}

if (-not $SkipBuild) {
    $Output = Join-Path $RootDir "bin\sanguoclip.exe"
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Output) | Out-Null
    Push-Location $RootDir
    try {
        & go build -o $Output ./cmd/sanguoclip
        if ($LASTEXITCODE -ne 0) {
            throw "Go build failed."
        }
    } finally {
        Pop-Location
    }
}

& (Join-Path $PSScriptRoot "doctor.ps1")

Write-Host ""
Write-Host "Setup complete."
Write-Host "Start SanguoClip: powershell -ExecutionPolicy Bypass -File .\scripts\run.ps1"
if ($WithEmulator) {
    Write-Host "Start LDPlayer: powershell -ExecutionPolicy Bypass -File .\scripts\start-emulator.ps1"
}
