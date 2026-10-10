# Secret hygiene gate for Trim (run before any public push).
# Usage (from trim/):  powershell -File scripts/check-secrets-hygiene.ps1
# Exit 0 = OK to proceed with Phase 0 reporting; non-zero = stop.

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$fail = $false

function Fail([string]$msg) {
  Write-Host "FAIL  $msg" -ForegroundColor Red
  $script:fail = $true
}
function Pass([string]$msg) {
  Write-Host "PASS  $msg" -ForegroundColor Green
}

Write-Host "=== Trim secrets hygiene ==="
Write-Host "cwd=$root"
Write-Host "git_root=$(git rev-parse --show-toplevel 2>$null)"
Write-Host "prefix=$(git rev-parse --show-prefix 2>$null)"

# 1) Real env files must not be tracked
$trackedEnv = @()
foreach ($path in (git ls-files)) {
  $name = Split-Path -Leaf $path
  if ($name -eq ".env" -or $name -eq ".env.local") {
    $trackedEnv += $path
    continue
  }
  if ($name -like ".env.*" -and $name -ne ".env.example") {
    $trackedEnv += $path
  }
}
if ($trackedEnv.Count -gt 0) {
  Fail ("Tracked env-like paths:`n" + ($trackedEnv -join "`n"))
} else {
  Pass "No real .env / .env.local tracked"
}

# 2) Examples must remain un-ignored (templates)
$examples = @(
  "server/.env.example",
  "cli/.env.example",
  "apps/web/.env.example",
  "apps/admin/.env.example"
)
foreach ($ex in $examples) {
  if (-not (Test-Path $ex)) {
    Fail "Missing template: $ex"
    continue
  }
  $ignored = git check-ignore -q $ex 2>$null
  # check-ignore exit 0 => ignored (bad for examples); 1 => not ignored
  if ($LASTEXITCODE -eq 0) {
    Fail "Template is gitignored (should be trackable): $ex"
  } else {
    Pass "Template trackable: $ex"
  }
}

# 3) Local secret files should exist only as ignored locals (optional presence)
$locals = @("server/.env", "cli/.env", "apps/web/.env.local", "apps/admin/.env.local")
foreach ($f in $locals) {
  if (Test-Path $f) {
    git check-ignore -q $f 2>$null
    if ($LASTEXITCODE -eq 0) {
      Pass "Local secret ignored: $f"
    } else {
      Fail "Local secret NOT ignored by git: $f"
    }
  } else {
    Write-Host "INFO  absent (ok): $f"
  }
}

# 4) Warn if git root is parent monorepo (publish risk)
$gt = (git rev-parse --show-toplevel 2>$null)
$prefix = (git rev-parse --show-prefix 2>$null)
if ($prefix -and $prefix -ne "") {
  Write-Host "WARN  trim/ is nested under git root: $gt" -ForegroundColor Yellow
  Write-Host "WARN  Do NOT push this parent monorepo as github.com/usetrim/trim." -ForegroundColor Yellow
  Write-Host "WARN  Phase 3 must be a CLEAN dedicated repo containing only Trim." -ForegroundColor Yellow
}

# 5) Remotes sanity
$remotes = git remote -v 2>$null
if ($remotes -match 'ihatemender|back-y') {
  Write-Host "WARN  remotes still point at legacy private names (ihatemender/back-y)." -ForegroundColor Yellow
  Write-Host "WARN  Expected later: https://github.com/usetrim/trim.git" -ForegroundColor Yellow
}

if ($fail) {
  Write-Host "`nRESULT: FAIL - fix before any public push." -ForegroundColor Red
  exit 1
}
Write-Host "`nRESULT: PASS - env templates OK; real envs ignored." -ForegroundColor Green
Write-Host "Still required from YOU: rotate credentials that lived in local .env files,"
Write-Host "then reply: env files tracked / history / credentials rotated."
exit 0
