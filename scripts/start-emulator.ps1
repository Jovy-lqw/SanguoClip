param(
    [int]$Index = 0,
    [switch]$OpenGame,
    [string]$PackageName
)

$ErrorActionPreference = "Stop"

. (Join-Path $PSScriptRoot "ldplayer.ps1")

$LdConsole = Find-LdConsole
if (-not $LdConsole) {
    throw "LDPlayer is missing. Run scripts/setup.ps1 -WithEmulator first."
}

& $LdConsole launch --index $Index
if ($LASTEXITCODE -ne 0) {
    throw "LDPlayer failed to start."
}

$Serial = Connect-LdPlayerAdb -Index $Index
if ($OpenGame) {
    if (-not $PackageName) {
        $Packages = @(
            & adb -s $Serial shell "cmd package query-activities --brief -a android.intent.action.VIEW -d 'yokasgs://?payload=sanguoclip-probe'" 2>$null |
                ForEach-Object {
                    if ($_ -match "^\s*([A-Za-z0-9_.]+)/") {
                        $Matches[1]
                    }
                } |
                Sort-Object -Unique
        )
        if ($Packages.Count -ne 1) {
            throw "Expected one installed Sanguosha client; found $($Packages.Count). Pass -PackageName explicitly."
        }
        $PackageName = $Packages[0]
    }
    & $LdConsole runapp --index $Index --packagename $PackageName
    if ($LASTEXITCODE -ne 0) {
        throw "LDPlayer failed to open $PackageName."
    }
}

Write-Host "LDPlayer is ready: $Serial"
& adb devices -l
