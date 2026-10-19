param(
    [string]$ApkPath,
    [string]$Serial,
    [string]$PackageName,
    [int]$Index = 0
)

$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
$SdkRoot = Join-Path $RootDir ".tools\android-sdk"
$OfficialUrl = "https://m.sanguosha.com/"
. (Join-Path $PSScriptRoot "ldplayer.ps1")

if (Test-Path $SdkRoot) {
    $env:Path = "$(Join-Path $SdkRoot 'platform-tools');$env:Path"
}

if (-not $ApkPath) {
    Start-Process $OfficialUrl
    Write-Host "Download the official Android APK, then run:"
    Write-Host "powershell -ExecutionPolicy Bypass -File .\scripts\install-client.ps1 -ApkPath C:\path\to\official.apk"
    Write-Host "Channel clients should be downloaded from their channel's official app store."
    exit 2
}

$ResolvedApk = (Resolve-Path $ApkPath -ErrorAction Stop).Path
if (-not (Get-Command adb -ErrorAction SilentlyContinue)) {
    throw "ADB is missing. Run scripts/setup.ps1 first."
}

$Header = New-Object byte[] 2
$Stream = [System.IO.File]::OpenRead($ResolvedApk)
try {
    if ($Stream.Read($Header, 0, 2) -ne 2 -or $Header[0] -ne 0x50 -or $Header[1] -ne 0x4B) {
        throw "The selected file is not a valid APK archive: $ResolvedApk"
    }
} finally {
    $Stream.Dispose()
}

function Get-GamePackages {
    param([string]$DeviceSerial)

    return @(
        & adb -s $DeviceSerial shell "cmd package query-activities --brief -a android.intent.action.VIEW -d 'yokasgs://?payload=sanguoclip-probe'" 2>$null |
            ForEach-Object {
                if ($_ -match "^\s*([A-Za-z0-9_.]+)/") {
                    $Matches[1]
                }
            } |
            Sort-Object -Unique
    )
}

function Get-ThirdPartyPackages {
    param([string]$DeviceSerial)

    return @(
        & adb -s $DeviceSerial shell pm list packages -3 2>$null |
            ForEach-Object { $_ -replace "^package:", "" } |
            Sort-Object -Unique
    )
}

$LdConsole = Find-LdConsole
if ($LdConsole) {
    $Serial = Connect-LdPlayerAdb -Index $Index
} elseif (-not $Serial) {
    $Devices = @(
        & adb devices |
            Select-Object -Skip 1 |
            ForEach-Object {
                if ($_ -match "^(\S+)\s+device$") {
                    $Matches[1]
                }
            }
    )
    if ($Devices.Count -ne 1) {
        & adb devices -l
        throw "Expected exactly one online Android device; found $($Devices.Count). Pass -Serial explicitly."
    }
    $Serial = $Devices[0]
}

$BeforeGamePackages = @(Get-GamePackages -DeviceSerial $Serial)
$BeforePackages = @(Get-ThirdPartyPackages -DeviceSerial $Serial)

if ($LdConsole) {
    & $LdConsole installapp --index $Index --filename $ResolvedApk
    if ($LASTEXITCODE -ne 0) {
        throw "LDPlayer failed to install the APK."
    }
} else {
    & adb -s $Serial install -r $ResolvedApk
    if ($LASTEXITCODE -ne 0) {
        throw "APK installation failed."
    }
}

$AfterGamePackages = @(Get-GamePackages -DeviceSerial $Serial)
$AfterPackages = @(Get-ThirdPartyPackages -DeviceSerial $Serial)

if (-not $PackageName) {
    $NewGamePackages = @($AfterGamePackages | Where-Object { $BeforeGamePackages -notcontains $_ })
    if ($NewGamePackages.Count -eq 1) {
        $PackageName = $NewGamePackages[0]
    } elseif ($AfterGamePackages.Count -eq 1) {
        $PackageName = $AfterGamePackages[0]
    } else {
        $NewPackages = @($AfterPackages | Where-Object { $BeforePackages -notcontains $_ })
        if ($NewPackages.Count -eq 1) {
            $PackageName = $NewPackages[0]
        }
    }
}

if (-not $PackageName) {
    Write-Host "Installed Sanguosha clients:"
    $AfterGamePackages | ForEach-Object { Write-Host "  $_" }
    throw "The APK was installed, but its package could not be selected automatically. Pass -PackageName explicitly."
}

$Component = & adb -s $Serial shell cmd package resolve-activity --brief `
    -a android.intent.action.MAIN `
    -c android.intent.category.LAUNCHER `
    $PackageName |
    ForEach-Object {
        if ($_ -match "^[A-Za-z0-9_.]+/") {
            $_
        }
    } |
    Select-Object -Last 1
if (-not $Component) {
    throw "Installed $PackageName, but no launcher activity was found."
}

& adb -s $Serial shell am start -W -n $Component

Write-Host ""
Write-Host "The client is installed and open: $PackageName"
Write-Host "Review its agreements and sign in directly in the emulator or Android device."
