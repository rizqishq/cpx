$ErrorActionPreference = "Stop"

$repo = "rizqishq/cpx"
$Version = if ($env:CPX_VERSION) { [string]$env:CPX_VERSION } else { "latest" }
$InstallDir = if ($env:CPX_INSTALL_DIR) {
    [string]$env:CPX_INSTALL_DIR
} elseif ($env:LOCALAPPDATA) {
    Join-Path $env:LOCALAPPDATA "Programs\cpx"
} else {
    Join-Path $HOME "AppData\Local\Programs\cpx"
}

function Resolve-Tag {
    param([string]$RequestedVersion)

    if ($RequestedVersion -eq "latest") {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
        if (-not $release.tag_name) {
            throw "Could not determine the latest release tag."
        }
        return [string]$release.tag_name
    }

    if ($RequestedVersion.StartsWith("v")) {
        return $RequestedVersion
    }

    return "v$RequestedVersion"
}

function Resolve-Arch {
    switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
        "X64" { return "amd64" }
        "Arm64" { return "arm64" }
        default { throw "Unsupported architecture: $([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture)" }
    }
}

$tag = Resolve-Tag -RequestedVersion $Version
$assetVersion = $tag.TrimStart("v")
$arch = Resolve-Arch
$asset = "cpx_${assetVersion}_windows_${arch}.zip"
$downloadUrl = "https://github.com/$repo/releases/download/$tag/$asset"
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("cpx-install-" + [System.Guid]::NewGuid().ToString("N"))
$zipPath = Join-Path $tempDir $asset

New-Item -ItemType Directory -Path $tempDir | Out-Null

try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath
    Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path (Join-Path $tempDir "cpx.exe") -Destination (Join-Path $InstallDir "cpx.exe") -Force

    Write-Host "Installed cpx $tag to $InstallDir\cpx.exe"
    Write-Host "Make sure $InstallDir is in your PATH."
}
finally {
    if (Test-Path $tempDir) {
        Remove-Item -Path $tempDir -Recurse -Force
    }
}
