param(
    [ValidateSet("vagrant", "usb")]
    [string]$Target = "vagrant",
    [switch]$NoTests
)

$ErrorActionPreference = "Stop"
$BuildDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectDir = Split-Path -Parent $BuildDir

Push-Location $ProjectDir
try {

if ($Target -eq "usb") {
    $OutDir = Join-Path $BuildDir "usb"
    $Steps = 4
    Write-Host "=== ScoutUTM Build (USB) ===" -ForegroundColor Cyan
} else {
    $OutDir = $BuildDir
    $Steps = 5
    Write-Host "=== ScoutUTM Build (Vagrant) ===" -ForegroundColor Cyan
}
Write-Host ""

# 1. Build frontend.
Write-Host "[1/$Steps] Building frontend..."
Push-Location "web"
npm ci --ignore-scripts
if ($LASTEXITCODE -ne 0) { throw "npm ci failed" }
npm run build --silent
if ($LASTEXITCODE -ne 0) { throw "frontend build failed" }
Pop-Location
Write-Host "  OK: scoutd\web\dist\" -ForegroundColor Green

# 2. Build scoutd binary (Go is in WSL, not Windows).
Write-Host "[2/$Steps] Building scoutd via WSL..."
wsl bash -c "cd scoutd && /usr/local/go/bin/go mod verify"
if ($LASTEXITCODE -ne 0) { throw "go mod verify failed - module checksums do not match go.sum" }
$drive = ($OutDir -split ':')[0].ToLower()
$rest = ($OutDir -split ':', 2)[1] -replace '\\', '/'
$WslOut = "/mnt/$drive$rest"
wsl bash -c "cd scoutd && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags '-X main.version=0.1.0-dev' -o '$WslOut/scoutd' ./cmd/scoutd/"
if ($LASTEXITCODE -ne 0) { throw "scoutd build failed" }
Write-Host "  OK: $($OutDir | Resolve-Path -Relative)\scoutd" -ForegroundColor Green

# 3. Copy templates.
Write-Host "[3/$Steps] Copying templates..."
$TemplatesDest = Join-Path $OutDir "templates"
New-Item -ItemType Directory -Force -Path $TemplatesDest | Out-Null
Copy-Item "scoutd\templates\*.tmpl" -Destination $TemplatesDest -Force
$tmplCount = (Get-ChildItem (Join-Path $TemplatesDest "*.tmpl")).Count
Write-Host "  OK: $tmplCount templates" -ForegroundColor Green

if ($Target -eq "usb") {
    # 4. Copy default config.
    Write-Host "[4/$Steps] Copying default config..."
    $DefaultsDest = Join-Path $OutDir "defaults"
    New-Item -ItemType Directory -Force -Path $DefaultsDest | Out-Null
    Copy-Item "build\defaults\config.json" -Destination "$DefaultsDest\config.json" -Force
    Write-Host "  OK: defaults\config.json" -ForegroundColor Green

    $binary = Get-Item (Join-Path $OutDir "scoutd")
    $size = "{0:N1} MB" -f ($binary.Length / 1MB)

    Write-Host ""
    Write-Host "=== USB build complete ===" -ForegroundColor Green
    Write-Host ""
    Write-Host "  build/usb/"
    Write-Host "    scoutd          ($size)"
    Write-Host "    templates/      ($tmplCount files)"
    Write-Host "    defaults/config.json"
    Write-Host "    setup.sh"
    Write-Host ""
    Write-Host "First install (USB stick):" -ForegroundColor Yellow
    Write-Host "  mount /dev/sdb1 /media/usb"
    Write-Host "  sh /media/usb/setup.sh"
    Write-Host ""
    Write-Host "Re-deploy (SSH already available):" -ForegroundColor Yellow
    Write-Host "  scp -i ~/.ssh/scoututm-usb -r ./usb/* root@172.16.0.1:/tmp/scout/"
    Write-Host "  ssh -i ~/.ssh/scoututm-usb root@172.16.0.1 'sh /tmp/scout/setup.sh'"
} else {
    # 4. Destroy existing VM.
    Write-Host "[4/$Steps] Destroying existing VM..."
    vagrant destroy -f 2>$null

    # 5. Create and provision VM.
    if ($NoTests) {
        Write-Host "[5/$Steps] Starting VM (-NoTests: skipping integration tests)..."
        $env:SCOUT_NO_TESTS = "1"
        vagrant up
        Remove-Item Env:\SCOUT_NO_TESTS
    } else {
        Write-Host "[5/$Steps] Starting VM (provision + tests)..."
        vagrant up
    }

    Write-Host ""
    Write-Host "=== Done ===" -ForegroundColor Cyan
    Write-Host "SSH:     vagrant ssh utm      (UTM appliance)"
    Write-Host "SSH:     vagrant ssh client   (LAN client)"
    Write-Host "Web UI:  https://localhost:8443"
    Write-Host "Login:   admin / admin"
}

} finally { Pop-Location }
