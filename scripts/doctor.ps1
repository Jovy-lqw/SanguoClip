$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
$SdkRoot = Join-Path $RootDir ".tools\android-sdk"
. (Join-Path $PSScriptRoot "ldplayer.ps1")

if (Test-Path $SdkRoot) {
    $env:ANDROID_HOME = $SdkRoot
    $env:ANDROID_SDK_ROOT = $SdkRoot
    $env:ANDROID_AVD_HOME = Join-Path $RootDir ".tools\avd"
    $env:Path = "$(Join-Path $SdkRoot 'platform-tools');$(Join-Path $SdkRoot 'emulator');$env:Path"
}

function Show-Dependency {
    param([string]$Name)

    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($command) {
        Write-Host ("{0,-9} {1}" -f $Name, $command.Source)
    } else {
        Write-Host ("{0,-9} missing" -f $Name)
    }
}

Show-Dependency "go"
Show-Dependency "adb"
Show-Dependency "ffmpeg"
Show-Dependency "ffprobe"

$LdConsole = Find-LdConsole
if ($LdConsole) {
    Write-Host ("{0,-9} {1}" -f "ldconsole", $LdConsole)
    & $LdConsole list2
} else {
    Write-Host ("{0,-9} missing" -f "ldconsole")
}

if (Get-Command adb -ErrorAction SilentlyContinue) {
    Write-Host ""
    $DeviceLines = @(& adb devices -l)
    $DeviceLines | ForEach-Object { Write-Host $_ }

    function Get-PackageStatus {
        param(
            [string]$Serial,
            [string]$PackageName
        )

        $Packages = @(& adb -s $Serial shell pm list packages $PackageName 2>$null)
        if ($Packages -contains "package:$PackageName") {
            return "installed"
        }
        return "missing"
    }

    foreach ($Line in $DeviceLines) {
        if ($Line -notmatch "^(\S+)\s+device(?:\s|$)") {
            continue
        }

        $Serial = $Matches[1]
        $GamePackage = "com.bf.sgs.hdexp.huawei"
        $Handlers = @(
            & adb -s $Serial shell `
                "cmd package query-activities --brief -a android.intent.action.VIEW -d 'yokasgs://?payload=sanguoclip-probe'" `
                2>$null
        )
        $DeepLinkStatus = if ($Handlers -match "^$([regex]::Escape($GamePackage))/") {
            "ready"
        } else {
            "missing"
        }

        Write-Host ""
        Write-Host "Huawei channel ($Serial)"
        Write-Host ("  {0,-11} {1}" -f "AppGallery", (Get-PackageStatus -Serial $Serial -PackageName "com.huawei.appmarket"))
        Write-Host ("  {0,-11} {1}" -f "HMS Core", (Get-PackageStatus -Serial $Serial -PackageName "com.huawei.hwid"))
        Write-Host ("  {0,-11} {1}" -f "Sanguosha", (Get-PackageStatus -Serial $Serial -PackageName $GamePackage))
        Write-Host ("  {0,-11} {1}" -f "deep link", $DeepLinkStatus)
    }
}
