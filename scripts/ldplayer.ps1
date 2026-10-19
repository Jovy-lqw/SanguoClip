$script:LdPlayerGamePage = "https://www.ldmnq.com/app/4697/"
$script:LdPlayerGameInstallerUrl = "https://lddl01.ldmnq.com/downloader/ldplayerinst14.exe?n=ldplayer14_ld_114_4697_2_ld.exe"

function Get-LdPlayerGameInstallerUrl {
    $Headers = @{
        "User-Agent" = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/140 Safari/537.36"
    }
    try {
        $Page = Invoke-WebRequest -UseBasicParsing -Uri $script:LdPlayerGamePage -Headers $Headers
        $Pattern = "https://lddl01\.ldmnq\.com/downloader/ldplayerinst14\.exe\?n=ldplayer14_[^`"'<\s]*4697[^`"'<\s]*\.exe"
        $Match = [regex]::Match($Page.Content, $Pattern)
        if ($Match.Success) {
            return $Match.Value
        }
    } catch {
        Write-Warning "Could not resolve the latest Sanguosha LDPlayer installer; using the known official URL."
    }
    return $script:LdPlayerGameInstallerUrl
}

function Find-LdConsole {
    $Candidates = @(
        (Join-Path $env:ProgramFiles "LDPlayer\LDPlayer14\ldconsole.exe"),
        (Join-Path $env:ProgramFiles "leidian\LDPlayer14\ldconsole.exe"),
        (Join-Path $env:ProgramFiles "LDPlayer\LDPlayer9\ldconsole.exe"),
        (Join-Path $env:ProgramFiles "leidian\LDPlayer9\ldconsole.exe"),
        (Join-Path ${env:ProgramFiles(x86)} "LDPlayer\LDPlayer14\ldconsole.exe"),
        (Join-Path ${env:ProgramFiles(x86)} "leidian\LDPlayer14\ldconsole.exe"),
        (Join-Path ${env:ProgramFiles(x86)} "LDPlayer\LDPlayer9\ldconsole.exe"),
        (Join-Path ${env:ProgramFiles(x86)} "leidian\LDPlayer9\ldconsole.exe"),
        "C:\leidian\LDPlayer14\ldconsole.exe",
        "D:\leidian\LDPlayer14\ldconsole.exe",
        "E:\leidian\LDPlayer14\ldconsole.exe",
        "C:\leidian\LDPlayer9\ldconsole.exe",
        "D:\leidian\LDPlayer9\ldconsole.exe",
        "E:\leidian\LDPlayer9\ldconsole.exe",
        "C:\LDPlayer\LDPlayer14\ldconsole.exe",
        "D:\LDPlayer\LDPlayer14\ldconsole.exe"
    )

    foreach ($Candidate in $Candidates) {
        if ($Candidate -and (Test-Path $Candidate)) {
            return (Resolve-Path $Candidate).Path
        }
    }

    $RegistryRoots = @(
        "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall",
        "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall",
        "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall"
    )
    foreach ($Root in $RegistryRoots) {
        if (-not (Test-Path $Root)) {
            continue
        }
        $Entries = Get-ChildItem $Root -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
            Where-Object { $_.DisplayName -match "LDPlayer|雷电模拟器" }
        foreach ($Entry in $Entries) {
            if ($Entry.InstallLocation) {
                $Console = Join-Path $Entry.InstallLocation "ldconsole.exe"
                if (Test-Path $Console) {
                    return (Resolve-Path $Console).Path
                }
            }
        }
    }

    return $null
}

function Get-LdPlayerSerial {
    param([int]$Index = 0)
    return "127.0.0.1:$((5555 + $Index * 2))"
}

function Connect-LdPlayerAdb {
    param(
        [int]$Index = 0,
        [int]$TimeoutSeconds = 180
    )

    if (-not (Get-Command adb -ErrorAction SilentlyContinue)) {
        throw "ADB is missing. Run scripts/setup.ps1 first."
    }

    $Serial = Get-LdPlayerSerial -Index $Index
    $Deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        & adb connect $Serial | Out-Null
        $State = & adb -s $Serial get-state 2>$null
        if ($LASTEXITCODE -eq 0 -and $State -eq "device") {
            return $Serial
        }
        Start-Sleep -Seconds 2
    } while ((Get-Date) -lt $Deadline)

    throw "LDPlayer did not become available through ADB at $Serial."
}
