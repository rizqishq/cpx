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

    Write-Host "Installed cpx $ResolvedVersion to $InstallPath"
    Write-Host "Make sure $InstallDir is in your PATH."
}
finally {
    if (Test-Path $TempDir) {
        Remove-Item -Path $TempDir -Recurse -Force
    }
}
