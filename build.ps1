$ErrorActionPreference = "Stop"
$ProjectDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$BuildDir = Join-Path $ProjectDir "build"

Write-Host "=== ScoutUTM Build ===" -ForegroundColor Cyan
Write-Host ""

# 1. Build scoutd binary (Go is in WSL, not Windows).
Write-Host "[1/4] Building scoutd via WSL..."
wsl bash -c "cd scoutd && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags '-X main.version=0.1.0-dev' -o ../build/scoutd ./cmd/scoutd/"
if ($LASTEXITCODE -ne 0) { throw "scoutd build failed" }
Write-Host "  OK: build\scoutd" -ForegroundColor Green

# 2. Copy templates into build directory.
Write-Host "[2/4] Copying templates..."
$TemplatesDest = Join-Path $BuildDir "templates"
if (-not (Test-Path $TemplatesDest)) { New-Item -ItemType Directory -Path $TemplatesDest | Out-Null }
Copy-Item (Join-Path $ProjectDir "scoutd\templates\*.tmpl") -Destination $TemplatesDest -Force
$count = (Get-ChildItem (Join-Path $TemplatesDest "*.tmpl")).Count
Write-Host "  OK: build\templates\ ($count files)" -ForegroundColor Green

# 3. Destroy existing VM.
Write-Host "[3/4] Destroying existing VM..."
Push-Location $ProjectDir
vagrant destroy -f 2>$null
Pop-Location

# 4. Create and provision VM (integration tests run at the end).
Write-Host "[4/4] Starting VM (provision + tests)..."
Push-Location $ProjectDir
vagrant up
Pop-Location

Write-Host ""
Write-Host "=== Done ===" -ForegroundColor Cyan
Write-Host "SSH:     vagrant ssh"
Write-Host "Web UI:  https://localhost:8443"
Write-Host "Login:   admin / admin"
