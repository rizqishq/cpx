param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"

$Owner = "rizqishq"
$Repo = "cpx"
$InstallDir = if ($env:CPX_INSTALL_DIR) { $env:CPX_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "cpx\bin" }

function Resolve-Version {
    param([string]$RequestedVersion)

    if ($RequestedVersion) {
        return $RequestedVersion
    }

    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Owner/$Repo/releases/latest"
    if (-not $release.tag_name) {
        throw "Failed to resolve latest release tag."
    }

    return [string]$release.tag_name
}

function Resolve-Arch {
    switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
        "X64" { return "amd64" }
        "Arm64" { return "arm64" }
        default { throw "Unsupported architecture: $([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture)" }
    }
}

function Add-ToPath {
    param([string]$PathToAdd)

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $entries = @()
    if ($userPath) {
        $entries = $userPath.Split(';', [System.StringSplitOptions]::RemoveEmptyEntries)
    }

    $alreadyPresent = $entries | Where-Object { $_.TrimEnd('\\') -ieq $PathToAdd.TrimEnd('\\') }
    $newEntries = @($entries)
    if (-not $alreadyPresent) {
        $newEntries = @($entries + $PathToAdd)
        [Environment]::SetEnvironmentVariable("Path", ($newEntries -join ';'), "User")
    }

    $userPathForSession = if ($newEntries.Count -gt 0) { $newEntries -join ';' } else { $PathToAdd }
    $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    if ($machinePath) {
        $env:Path = "$userPathForSession;$machinePath"
    }
    else {
        $env:Path = $userPathForSession
    }
}

$ResolvedVersion = Resolve-Version -RequestedVersion $Version
$AssetVersion = if ($ResolvedVersion.StartsWith("v")) { $ResolvedVersion.Substring(1) } else { $ResolvedVersion }
$Arch = Resolve-Arch
$Archive = "cpx_{0}_windows_{1}.zip" -f $AssetVersion, $Arch
$Url = "https://github.com/$Owner/$Repo/releases/download/$ResolvedVersion/$Archive"

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
$ArchivePath = Join-Path $TempDir $Archive

New-Item -ItemType Directory -Path $TempDir | Out-Null

try {
    Write-Host "Downloading $Url"
    Invoke-WebRequest -Uri $Url -OutFile $ArchivePath

    Expand-Archive -Path $ArchivePath -DestinationPath $TempDir -Force

    $BinaryPath = Join-Path $TempDir "cpx.exe"
    if (-not (Test-Path $BinaryPath)) {
        throw "cpx.exe was not found in the downloaded archive."
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $InstallPath = Join-Path $InstallDir "cpx.exe"
    Move-Item -Path $BinaryPath -Destination $InstallPath -Force
    Add-ToPath -PathToAdd $InstallDir

    Write-Host "Installed cpx $ResolvedVersion to $InstallPath"
    Write-Host "Added $InstallDir to your user PATH."
    Write-Host "You can run cpx immediately in this PowerShell session."
}
finally {
    if (Test-Path $TempDir) {
        Remove-Item -Path $TempDir -Recurse -Force
    }
}
