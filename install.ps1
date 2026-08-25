# Install the ollygarden CLI on Windows.
#
# Quick install:
#   irm https://raw.githubusercontent.com/ollygarden/ollygarden-cli/main/install.ps1 | iex
#
# Customize with OLLYGARDEN_VERSION, OLLYGARDEN_INSTALL_DIR, or GITHUB_TOKEN.

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Repository = "ollygarden/ollygarden-cli"
$BinaryName = "ollygarden.exe"

if ([string]::IsNullOrWhiteSpace($env:OLLYGARDEN_INSTALL_DIR)) {
    if ([string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) {
        throw "LOCALAPPDATA is not set; set OLLYGARDEN_INSTALL_DIR to an installation directory"
    }
    $InstallDirectory = Join-Path $env:LOCALAPPDATA "OllyGarden"
} else {
    $InstallDirectory = [Environment]::ExpandEnvironmentVariables($env:OLLYGARDEN_INSTALL_DIR)
}
$InstallDirectory = [IO.Path]::GetFullPath($InstallDirectory)

function Get-OllyGardenArchitecture {
    try {
        $architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    } catch {
        if (-not [string]::IsNullOrWhiteSpace($env:PROCESSOR_ARCHITEW6432)) {
            $architecture = $env:PROCESSOR_ARCHITEW6432
        } else {
            $architecture = $env:PROCESSOR_ARCHITECTURE
        }
    }

    switch ($architecture.ToUpperInvariant()) {
        "AMD64" { return "amd64" }
        "X64" { return "amd64" }
        "ARM64" { return "arm64" }
        default { throw "unsupported Windows architecture: $architecture" }
    }
}

function Get-GitHubHeaders {
    $headers = @{ "User-Agent" = "ollygarden-installer" }
    $token = $env:GITHUB_TOKEN

    if ([string]::IsNullOrWhiteSpace($token) -and $null -ne (Get-Command gh -ErrorAction SilentlyContinue)) {
        $token = (& gh auth token 2>$null | Select-Object -First 1)
        if ($LASTEXITCODE -ne 0) {
            $token = ""
        }
    }

    if (-not [string]::IsNullOrWhiteSpace($token)) {
        $headers["Authorization"] = "Bearer $token"
    }
    return $headers
}

$architecture = Get-OllyGardenArchitecture
$version = $env:OLLYGARDEN_VERSION

if ([string]::IsNullOrWhiteSpace($version)) {
    Write-Host "Resolving latest release..."
    $release = Invoke-RestMethod `
        -Uri "https://api.github.com/repos/$Repository/releases/latest" `
        -Headers (Get-GitHubHeaders)
    $version = [string]$release.tag_name
}

$version = $version.Trim()
if (-not $version.StartsWith("v") -or $version.Length -eq 1) {
    throw "OLLYGARDEN_VERSION must be a release tag such as v0.2.2"
}

$versionNumber = $version.Substring(1)
$archiveName = "ollygarden_${versionNumber}_windows_${architecture}.zip"
$downloadBase = "https://github.com/$Repository/releases/download/$version"
$temporaryDirectory = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetRandomFileName())

New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null
try {
    $archivePath = Join-Path $temporaryDirectory $archiveName
    $checksumsPath = Join-Path $temporaryDirectory "checksums.txt"

    Write-Host "Downloading $archiveName..."
    Invoke-WebRequest -Uri "$downloadBase/$archiveName" -OutFile $archivePath
    Invoke-WebRequest -Uri "$downloadBase/checksums.txt" -OutFile $checksumsPath

    Write-Host "Verifying checksum..."
    $archivePattern = [Regex]::Escape($archiveName)
    $checksumMatches = @(Get-Content $checksumsPath | Where-Object {
        $_ -match "^[0-9a-fA-F]{64}\s+\*?$archivePattern$"
    })
    if ($checksumMatches.Count -eq 0) {
        throw "no checksum entry for $archiveName"
    }

    $expectedChecksum = ($checksumMatches[0] -split "\s+", 2)[0].ToLowerInvariant()
    $actualChecksum = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()
    if ($expectedChecksum -ne $actualChecksum) {
        throw "checksum mismatch: expected $expectedChecksum, got $actualChecksum"
    }

    $extractedDirectory = Join-Path $temporaryDirectory "extracted"
    Expand-Archive -LiteralPath $archivePath -DestinationPath $extractedDirectory
    $extractedBinary = Join-Path $extractedDirectory $BinaryName
    if (-not (Test-Path -LiteralPath $extractedBinary -PathType Leaf)) {
        throw "$archiveName does not contain $BinaryName"
    }

    New-Item -ItemType Directory -Force -Path $InstallDirectory | Out-Null
    $installedBinary = Join-Path $InstallDirectory $BinaryName
    Copy-Item -LiteralPath $extractedBinary -Destination $installedBinary -Force
} finally {
    Remove-Item -LiteralPath $temporaryDirectory -Recurse -Force -ErrorAction SilentlyContinue
}

$normalizedInstallDirectory = $InstallDirectory.TrimEnd("\")
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$userPathEntries = @($userPath -split ";" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
$normalizedUserPathEntries = @($userPathEntries | ForEach-Object { $_.TrimEnd("\") })

if ($normalizedUserPathEntries -notcontains $normalizedInstallDirectory) {
    $updatedUserPath = if ([string]::IsNullOrWhiteSpace($userPath)) {
        $InstallDirectory
    } else {
        "$userPath;$InstallDirectory"
    }
    [Environment]::SetEnvironmentVariable("Path", $updatedUserPath, "User")
    Write-Host "Added $InstallDirectory to the user Path."
}

if (@($env:Path -split ";" | ForEach-Object { $_.TrimEnd("\") }) -notcontains $normalizedInstallDirectory) {
    $env:Path = "$InstallDirectory;$env:Path"
}

Write-Host "Installed ollygarden $version to $installedBinary"
& $installedBinary version
if ($LASTEXITCODE -ne 0) {
    throw "installed ollygarden exited with code $LASTEXITCODE"
}
