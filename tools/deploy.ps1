# ================================================================
# tools/deploy.ps1 — Build, transfer, and re-deploy ScoutUTM
# ================================================================
#
# Uses Windows SSH/SCP for all operations. First run uses sshpass
# (WSL) to push the Windows SSH key, then changes root password.
#
# Usage:
#   .\tools\deploy.ps1                        # build + deploy
#   .\tools\deploy.ps1 -SkipBuild             # deploy only
#   .\tools\deploy.ps1 -DeviceIP 10.0.0.1     # different IP
#   .\tools\deploy.ps1 -FullSetup             # first-time full setup.sh
#
param(
    [string]$DeviceIP = "172.16.0.1",
    [string]$User = "root",
    [switch]$SkipBuild,
    [switch]$FullSetup
)

$ErrorActionPreference = "Stop"
$Root = (Resolve-Path "$PSScriptRoot\..").Path
$USB  = "$Root\build\usb"
$Target = "$User@$DeviceIP"
$SSHKey = "$env:USERPROFILE\.ssh\id_ed25519"
$SSHOpts = @("-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=5",
             "-o", "UserKnownHostsFile=NUL", "-o", "LogLevel=ERROR")

# ── SSH key setup (first-run only) ──

# Generate key pair if missing.
if (-not (Test-Path $SSHKey)) {
    Write-Host "==> Generating SSH key..." -ForegroundColor Cyan
    ssh-keygen.exe -t ed25519 -f $SSHKey -N '""' -q
}

# Test if key auth works (using Windows ssh).
$ErrorActionPreference = "Continue"
$keyTest = Start-Process -FilePath "ssh.exe" -ArgumentList "-o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes $Target true" -NoNewWindow -Wait -PassThru
$keyWorks = ($keyTest.ExitCode -eq 0)
$ErrorActionPreference = "Stop"

if (-not $keyWorks) {
    Write-Host "==> Setting up SSH key auth (first-time)..." -ForegroundColor Cyan

    # Ensure sshpass is in WSL.
    wsl bash -c "command -v sshpass >/dev/null 2>&1 || (sudo apt-get update -qq && sudo apt-get install -y -qq sshpass 2>/dev/null) || (sudo apk add sshpass 2>/dev/null)"

    # Convert Windows key path to WSL path for reading the pubkey.
    $KeyDrive = "$SSHKey.pub".Substring(0,1).ToLower()
    $WslPubKey = "/mnt/$KeyDrive" + ("$SSHKey.pub".Substring(2) -replace '\\', '/')
    $InitPass = "changeme"

    # Push key: pipe pubkey file into remote cat >> authorized_keys.
    wsl bash -c "sshpass -p $InitPass ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o PreferredAuthentications=password $Target 'mkdir -p ~/.ssh; cat >> ~/.ssh/authorized_keys; chmod 700 ~/.ssh; chmod 600 ~/.ssh/authorized_keys' < '$WslPubKey'"
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Failed to push SSH key. Is $DeviceIP reachable with password '$InitPass'?" -ForegroundColor Red
        exit 1
    }
    Write-Host "  SSH key authorized" -ForegroundColor Green

    # Change root password to a random string.
    $Chars = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
    $NewPass = "changeme"

    ssh.exe @SSHOpts $Target "echo $User`:$NewPass | chpasswd"
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  Root password changed to: $NewPass" -ForegroundColor Yellow
    } else {
        Write-Host "  WARNING: password change failed (SSH key still works)" -ForegroundColor Yellow
    }
}

# ── Build ──

if (-not $SkipBuild) {
    Write-Host "==> Building..." -ForegroundColor Cyan
    & "$Root\build\build.ps1" -Target usb
    if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: Build failed" -ForegroundColor Red; exit 1 }
} else {
    Write-Host "==> Skipping build (using existing build/usb/)" -ForegroundColor Cyan
}

if (-not (Test-Path "$USB\scoutd")) {
    Write-Host "ERROR: build/usb/scoutd not found - run without -SkipBuild" -ForegroundColor Red
    exit 1
}

# ── Transfer ──

Write-Host "==> Uploading to $Target..." -ForegroundColor Cyan

scp.exe @SSHOpts "$USB\scoutd" "${Target}:/tmp/scoutd"
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: SCP scoutd failed" -ForegroundColor Red; exit 1 }

scp.exe @SSHOpts -r "$USB\templates" "${Target}:/tmp/scout-templates"
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: SCP templates failed" -ForegroundColor Red; exit 1 }

scp.exe @SSHOpts "$USB\defaults\config.json" "${Target}:/tmp/scout-config.json"
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: SCP config failed" -ForegroundColor Red; exit 1 }

if ($FullSetup) {
    scp.exe @SSHOpts "$USB\setup.sh" "${Target}:/tmp/setup.sh"
    if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: SCP setup.sh failed" -ForegroundColor Red; exit 1 }
}

# ── Deploy ──

if ($FullSetup) {
    Write-Host "==> Running full setup..." -ForegroundColor Cyan
    $script = @"
mkdir -p /tmp/scout-usb/defaults /tmp/scout-usb/templates && mv /tmp/scoutd /tmp/scout-usb/scoutd && mv /tmp/scout-config.json /tmp/scout-usb/defaults/config.json && mv /tmp/scout-templates/*.tmpl /tmp/scout-usb/templates/ && mv /tmp/setup.sh /tmp/scout-usb/setup.sh && sh /tmp/scout-usb/setup.sh && rm -rf /tmp/scout-usb /tmp/scout-templates
"@
    ssh.exe @SSHOpts $Target $script
} else {
    Write-Host "==> Deploying update..." -ForegroundColor Cyan
    $script = @"
rc-service scoutd stop 2>/dev/null; sleep 1; install -m 755 /tmp/scoutd /usr/sbin/scoutd && cp /tmp/scout-templates/*.tmpl /usr/share/scoutd/templates/ && rc-service scoutd start && sleep 3 && (nft -f /etc/nftables.conf 2>/dev/null || true) && (rc-service dnsmasq restart 2>/dev/null || true) && (rc-service unbound restart 2>/dev/null || true) && echo '=== Deploy complete ===' && (rc-service scoutd status 2>/dev/null || echo 'scoutd: not running') && rm -f /tmp/scoutd /tmp/scout-config.json && rm -rf /tmp/scout-templates
"@
    ssh.exe @SSHOpts $Target $script
}

if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: Deploy failed" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "=== Done ===" -ForegroundColor Green
Write-Host "  https://$DeviceIP"
Write-Host ""
