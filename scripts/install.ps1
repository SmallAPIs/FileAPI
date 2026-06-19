<#
.SYNOPSIS
  Install, upgrade, or uninstall FileAPI on Windows.

.DESCRIPTION
  Downloads a release binary and adds FileAPI to your user PATH.

  Quick install:

    irm https://raw.githubusercontent.com/SmallAPIs/FileAPI/main/scripts/install.ps1 | iex

  Specific version:

    $env:FILEAPI_VERSION="v1.0.0"; irm https://raw.githubusercontent.com/SmallAPIs/FileAPI/main/scripts/install.ps1 | iex

  Uninstall:

    $env:FILEAPI_UNINSTALL=1; irm https://raw.githubusercontent.com/SmallAPIs/FileAPI/main/scripts/install.ps1 | iex

  Environment variables:

    FILEAPI_VERSION       Target release tag (default: latest)
    FILEAPI_INSTALL_DIR   Custom install directory
    FILEAPI_GITHUB_REPO   GitHub repo (default: SmallAPIs/FileAPI)
    FILEAPI_UNINSTALL     Set to 1 to uninstall FileAPI
#>

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$Version = if ($env:FILEAPI_VERSION) { $env:FILEAPI_VERSION } else { "" }
$GitHubRepo = if ($env:FILEAPI_GITHUB_REPO) { $env:FILEAPI_GITHUB_REPO } else { "SmallAPIs/FileAPI" }
$Uninstall = $env:FILEAPI_UNINSTALL -eq "1"
$InstallDir = if ($env:FILEAPI_INSTALL_DIR) {
    $env:FILEAPI_INSTALL_DIR
} else {
    Join-Path $env:LOCALAPPDATA "Programs\FileAPI"
}

function Write-Step {
    param([string]$Message)
    Write-Host ">>> $Message"
}

function Get-DownloadUrl {
    $artifact = "fileapi-windows-amd64.exe"
    if ($Version) {
        return "https://github.com/$GitHubRepo/releases/download/$Version/$artifact"
    }
    return "https://github.com/$GitHubRepo/releases/latest/download/$artifact"
}

function Update-UserPath {
    param([string]$Directory)

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ([string]::IsNullOrWhiteSpace($userPath)) {
        $segments = @()
    } else {
        $segments = $userPath -split ';' | Where-Object { $_ -and $_ -ne $Directory }
    }

    $newPath = @($Directory) + $segments
    $joined = ($newPath -join ';').TrimEnd(';')
    [Environment]::SetEnvironmentVariable("Path", $joined, "User")

    if ($env:PATH -notlike "*$Directory*") {
        $env:PATH = "$Directory;$env:PATH"
    }
}

function Remove-UserPathEntry {
    param([string]$Directory)

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ([string]::IsNullOrWhiteSpace($userPath)) {
        return
    }

    $segments = $userPath -split ';' | Where-Object { $_ -and ($_ -ne $Directory) }
    [Environment]::SetEnvironmentVariable("Path", ($segments -join ';'), "User")
}

function Invoke-Uninstall {
    Write-Step "Uninstalling FileAPI"

    $binary = Join-Path $InstallDir "fileapi.exe"
    if (Test-Path $binary) {
        Remove-Item $binary -Force
    }

    if (Test-Path $InstallDir) {
        $remaining = Get-ChildItem -Path $InstallDir -Force -ErrorAction SilentlyContinue
        if (-not $remaining) {
            Remove-Item $InstallDir -Force -ErrorAction SilentlyContinue
        }
    }

    Remove-UserPathEntry -Directory $InstallDir

    Write-Step "FileAPI has been uninstalled."
}

function Invoke-Install {
    $downloadUrl = Get-DownloadUrl
    $tempFile = Join-Path $env:TEMP "fileapi-windows-amd64.exe"

    Write-Step "Downloading FileAPI for Windows..."
    Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -UseBasicParsing

    Write-Step "Installing FileAPI to $InstallDir..."
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $target = Join-Path $InstallDir "fileapi.exe"
    Copy-Item -Path $tempFile -Destination $target -Force
    Remove-Item $tempFile -Force -ErrorAction SilentlyContinue

    Write-Step "Adding FileAPI to your user PATH..."
    Update-UserPath -Directory $InstallDir

    Write-Step "Install complete. Run 'fileapi' from the command line."
    Write-Step "Start the agent with: fileapi serve"
}

if ($Uninstall) {
    Invoke-Uninstall
} else {
    Invoke-Install
}
