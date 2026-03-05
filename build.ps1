param(
    [switch]$NoTests
)

$ErrorActionPreference = "Stop"
$ProjectDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$BuildDir = Join-Path $ProjectDir "build"

Write-Host "=== ScoutUTM Build ===" -ForegroundColor Cyan
Write-Host ""

# 1. Build frontend.
Write-Host "[1/5] Building frontend..."
Push-Location (Join-Path $ProjectDir "web")
npm run build --silent
if ($LASTEXITCODE -ne 0) { throw "frontend build failed" }
Pop-Location
Write-Host "  OK: scoutd\web\dist\" -ForegroundColor Green

# 2. Build scoutd binary (Go is in WSL, not Windows).
Write-Host "[2/5] Building scoutd via WSL..."
wsl bash -c "cd scoutd && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags '-X main.version=0.1.0-dev' -o ../build/scoutd ./cmd/scoutd/"
if ($LASTEXITCODE -ne 0) { throw "scoutd build failed" }
Write-Host "  OK: build\scoutd" -ForegroundColor Green

# 3. Copy templates into build directory.
Write-Host "[3/5] Copying templates..."
$TemplatesDest = Join-Path $BuildDir "templates"
if (-not (Test-Path $TemplatesDest)) { New-Item -ItemType Directory -Path $TemplatesDest | Out-Null }
Copy-Item (Join-Path $ProjectDir "scoutd\templates\*.tmpl") -Destination $TemplatesDest -Force
$count = (Get-ChildItem (Join-Path $TemplatesDest "*.tmpl")).Count
Write-Host "  OK: build\templates\ ($count files)" -ForegroundColor Green

# 4. Destroy existing VM.
Write-Host "[4/5] Destroying existing VM..."
Push-Location $ProjectDir
vagrant destroy -f 2>$null
Pop-Location

# 5. Create and provision VM.
Push-Location $ProjectDir
if ($NoTests) {
    Write-Host "[5/5] Starting VM (-NoTests: skipping integration tests)..."
    $env:SCOUT_NO_TESTS = "1"
    vagrant up
    Remove-Item Env:\SCOUT_NO_TESTS
} else {
    Write-Host "[5/5] Starting VM (provision + tests)..."
    vagrant up
}
Pop-Location

Write-Host ""
Write-Host "=== Done ===" -ForegroundColor Cyan
Write-Host "SSH:     vagrant ssh utm      (UTM appliance)"
Write-Host "SSH:     vagrant ssh client   (LAN client)"
Write-Host "Web UI:  https://localhost:8443"
Write-Host "Login:   admin / admin"
