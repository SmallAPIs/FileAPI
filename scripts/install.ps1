<#
.SYNOPSIS
  Install, upgrade, or uninstall FileAPI on Windows.

.DESCRIPTION
  Downloads a release binary and adds FileAPI to your user PATH.

  Public repository:

    irm https://raw.githubusercontent.com/SmallAPIs/FileAPI/main/scripts/install.ps1 | iex

  Private repository (GitHub CLI):

    gh api repos/SmallAPIs/FileAPI/contents/scripts/install.ps1?ref=main -H "Accept: application/vnd.github.raw" | Invoke-Expression

  Private repository (personal access token):

    $env:FILEAPI_GITHUB_TOKEN = "ghp_..."
    $install = Invoke-RestMethod `
      -Uri "https://api.github.com/repos/SmallAPIs/FileAPI/contents/scripts/install.ps1?ref=main" `
      -Headers @{ Authorization = "Bearer $env:FILEAPI_GITHUB_TOKEN"; Accept = "application/vnd.github+json" }
    [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String(($install.content -replace "`n", ""))) | Invoke-Expression

  Environment variables:

    FILEAPI_VERSION         Target release tag (default: latest)
    FILEAPI_INSTALL_DIR     Custom install directory
    FILEAPI_GITHUB_REPO     GitHub repo (default: SmallAPIs/FileAPI)
    FILEAPI_GITHUB_TOKEN    GitHub token with repo read access (required for private repos)
    GITHUB_TOKEN            Fallback token variable
    FILEAPI_UNINSTALL       Set to 1 to uninstall FileAPI
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

function Get-GitHubToken {
    if ($env:FILEAPI_GITHUB_TOKEN) { return $env:FILEAPI_GITHUB_TOKEN }
    if ($env:GITHUB_TOKEN) { return $env:GITHUB_TOKEN }
    return $null
}

function Get-DirectDownloadUrl {
    $artifact = "fileapi-windows-amd64.exe"
    if ($Version) {
        return "https://github.com/$GitHubRepo/releases/download/$Version/$artifact"
    }
    return "https://github.com/$GitHubRepo/releases/latest/download/$artifact"
}

function Get-PrivateRepoHelp {
    return @"
Download failed. The $GitHubRepo repository appears to be private.

Make the repository public in GitHub Settings, or install with one of these options:

  GitHub CLI (after: gh auth login):
    gh api repos/$GitHubRepo/contents/scripts/install.ps1?ref=main -H "Accept: application/vnd.github.raw" | Invoke-Expression

  Personal access token:
    `$env:FILEAPI_GITHUB_TOKEN = "ghp_..."`
    Then re-run this installer.
"@
}

function Invoke-DownloadReleaseAsset {
    param(
        [string]$OutFile
    )

    $artifact = "fileapi-windows-amd64.exe"
    $token = Get-GitHubToken
    $directUrl = Get-DirectDownloadUrl

    $directHeaders = @{}
    if ($token) {
        $directHeaders["Authorization"] = "Bearer $token"
    }

    try {
        Invoke-WebRequest -Uri $directUrl -OutFile $OutFile -Headers $directHeaders -UseBasicParsing | Out-Null
        return
    } catch {
        if (-not $token) {
            throw (Get-PrivateRepoHelp)
        }
    }

    $apiHeaders = @{
        Authorization = "Bearer $token"
        Accept = "application/vnd.github+json"
        "X-GitHub-Api-Version" = "2022-11-28"
        "User-Agent" = "FileAPI-Install"
    }

    if ($Version) {
        $releaseUri = "https://api.github.com/repos/$GitHubRepo/releases/tags/$Version"
    } else {
        $releaseUri = "https://api.github.com/repos/$GitHubRepo/releases/latest"
    }

    $release = Invoke-RestMethod -Uri $releaseUri -Headers $apiHeaders
    $asset = $release.assets | Where-Object { $_.name -eq $artifact } | Select-Object -First 1
    if (-not $asset) {
        throw "Release asset '$artifact' was not found for $GitHubRepo."
    }

    $downloadHeaders = @{
        Authorization = "Bearer $token"
        Accept = "application/octet-stream"
        "User-Agent" = "FileAPI-Install"
    }
    Invoke-WebRequest -Uri $asset.url -OutFile $OutFile -Headers $downloadHeaders -UseBasicParsing | Out-Null
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
    $tempFile = Join-Path $env:TEMP "fileapi-windows-amd64.exe"

    Write-Step "Downloading FileAPI for Windows..."
    Invoke-DownloadReleaseAsset -OutFile $tempFile

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
