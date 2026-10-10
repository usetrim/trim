#Requires -Version 5.1
# Trim Windows installer (mirrors scripts/install.sh). Host at https://use-trim.com/install.ps1
# Usage: irm https://use-trim.com/install.ps1 | iex
$ErrorActionPreference = "Stop"

function Get-TrimArch {
  $a = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
  switch -Regex ($a) {
    "x64|amd64" { return "amd64" }
    "arm64" { return "arm64" }
    default { throw "Unsupported architecture: $a" }
  }
}

function Invoke-TrimInstallBeacon {
  param([string]$PathQuery = "/install.ps1")
  if ($env:DO_NOT_TRACK -eq "1" -or $env:TRIM_TELEMETRY_DISABLED -eq "1") {
    return
  }
  $api = if ($env:TRIM_API_BASE_URL) { $env:TRIM_API_BASE_URL.TrimEnd("/") } else { "https://api.use-trim.com" }
  $uri = "$api/api/v1/public/install-hit?path=$([uri]::EscapeDataString($PathQuery))"
  try {
    Invoke-WebRequest -Uri $uri -Method POST -TimeoutSec 3 -UseBasicParsing | Out-Null
  } catch {
    # Fail soft: install must not fail if beacon is unreachable.
  }
}

$repo = if ($env:TRIM_GITHUB_REPO) { $env:TRIM_GITHUB_REPO } else { "usetrim/trim" }
$arch = Get-TrimArch
$os = "windows"

$releaseJson = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -Headers @{
  "User-Agent" = "trim-install.ps1"
  "Accept"     = "application/vnd.github+json"
}
$tag = [string]$releaseJson.tag_name
if (-not $tag) {
  throw "Could not resolve latest release tag from $repo"
}

$withTs = ("$env:TRIM_WITH_TREESITTER").ToLowerInvariant()
$withZig = ("$env:TRIM_WITH_ZIG_TREESITTER").ToLowerInvariant()
$ext = "zip"
$binaryName = "trim.exe"
$asset = "trim_${os}_${arch}.${ext}"

if ($withZig -in @("1", "true", "yes")) {
  if ($arch -ne "amd64") {
    throw "No treesitter+Zig release asset for windows/$arch (amd64 only)."
  }
  $asset = "trim_windows_amd64_treesitter_zig.zip"
  $withTs = "1"
} elseif ($withTs -in @("1", "true", "yes")) {
  if ($arch -ne "amd64") {
    throw "No treesitter release asset for windows/$arch (amd64 only)."
  }
  $asset = "trim_windows_amd64_treesitter.zip"
}

$base = "https://github.com/$repo/releases/download/$tag"
$url = "$base/$asset"
$checksumsUrl = "$base/checksums.txt"

Write-Host "Installing Trim $tag for $os/$arch (asset: $asset)..."

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("trim-install-" + [guid]::NewGuid().ToString("n"))
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  $archive = Join-Path $tmp "trim_archive.zip"
  Invoke-WebRequest -Uri $url -OutFile $archive -UseBasicParsing

  $checksumsPath = Join-Path $tmp "checksums.txt"
  Invoke-WebRequest -Uri $checksumsUrl -OutFile $checksumsPath -UseBasicParsing
  $expectedLine = Get-Content $checksumsPath | Where-Object { $_ -match ("\s" + [regex]::Escape($asset) + "\s*$") } | Select-Object -First 1
  if (-not $expectedLine) {
    throw "No checksum entry for $asset in checksums.txt"
  }
  $expected = ($expectedLine -split "\s+")[0].Trim().ToLowerInvariant()
  $actual = (Get-FileHash -Path $archive -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($actual -ne $expected) {
    throw "Checksum mismatch for $asset`n  expected: $expected`n  actual:   $actual"
  }
  Write-Host "Checksum OK ($actual)"

  Expand-Archive -Path $archive -DestinationPath $tmp -Force
  $bin = Get-ChildItem -Path $tmp -Recurse -Filter $binaryName -File | Select-Object -First 1
  if (-not $bin) {
    throw "Archive extracted but $binaryName was not found"
  }

  $installDir = if ($env:TRIM_INSTALL_DIR) { $env:TRIM_INSTALL_DIR } else {
    Join-Path $env:LOCALAPPDATA "Programs\trim"
  }
  New-Item -ItemType Directory -Path $installDir -Force | Out-Null
  $dest = Join-Path $installDir "trim.exe"
  Copy-Item -Path $bin.FullName -Destination $dest -Force

  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if (-not $userPath) { $userPath = "" }
  $parts = $userPath -split ";" | Where-Object { $_ -and $_.Trim() -ne "" }
  if ($parts -notcontains $installDir) {
    [Environment]::SetEnvironmentVariable("Path", (($parts + $installDir) -join ";"), "User")
    $env:Path = "$installDir;$env:Path"
  }

  # Register Apps & features from public CLI chrome (fail-closed: skip if chrome missing).
  $apiBase = if ($env:TRIM_API_BASE_URL) { $env:TRIM_API_BASE_URL.TrimEnd("/") } else { "https://api.use-trim.com" }
  $arpDisplay = $null
  $arpPublisher = $null
  $arpRegKey = $null
  try {
    $prov = Invoke-RestMethod -Uri "$apiBase/api/v1/public/auth-providers" -Headers @{ "User-Agent" = "trim-install.ps1" } -TimeoutSec 8
    $cli = $prov.local.cli
    if ($cli) {
      $arpDisplay = [string]$cli.uninstall_arp_display_name
      $arpPublisher = [string]$cli.uninstall_arp_publisher
      $arpRegKey = [string]$cli.uninstall_arp_reg_key
    }
  } catch {
    # Fail-closed: no invent ARP names when API chrome is unreachable.
  }
  if ($arpDisplay -and $arpPublisher -and $arpRegKey -and ($arpRegKey -notmatch '[\\/]')) {
    $arpKey = Join-Path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall" $arpRegKey
    New-Item -Path $arpKey -Force | Out-Null
    Set-ItemProperty -Path $arpKey -Name "DisplayName" -Value $arpDisplay
    Set-ItemProperty -Path $arpKey -Name "Publisher" -Value $arpPublisher
    Set-ItemProperty -Path $arpKey -Name "InstallLocation" -Value $installDir
    $displayVersion = if ($tag -match '^v?(.+)$') { $Matches[1] } else { $tag }
    Set-ItemProperty -Path $arpKey -Name "DisplayVersion" -Value $displayVersion
    Set-ItemProperty -Path $arpKey -Name "UninstallString" -Value "`"$dest`" uninstall"
    Set-ItemProperty -Path $arpKey -Name "QuietUninstallString" -Value "`"$dest`" uninstall"
    Set-ItemProperty -Path $arpKey -Name "NoModify" -Value 1 -Type DWord
    Set-ItemProperty -Path $arpKey -Name "NoRepair" -Value 1 -Type DWord
  }

  Invoke-TrimInstallBeacon -PathQuery "/install.ps1"

  Write-Host "Trim installed to $dest"
  Write-Host "Run: trim start"
  Write-Host "Then: trim login"
  Write-Host "Uninstall via Settings → Apps, or: trim uninstall"
  Write-Host "Open a new terminal if 'trim' is not on PATH yet."
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
