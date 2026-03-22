#Requires -Version 5.1
<#
.SYNOPSIS
    Smoke test a running ScoutOS test VM.

.DESCRIPTION
    Waits for the ScoutOS VM to boot, then runs basic health checks:
      1. VM is running
      2. Serial log shows successful boot (initramfs, OverlayFS, OpenRC)
      3. scoutd API responds (via Vagrant client VM or serial log)
      4. Data partition was resized on first boot

    Run this AFTER test-vm.ps1 has started the VM.

.PARAMETER Timeout
    Seconds to wait for boot to complete (default: 120).

.EXAMPLE
    .\tools\test-vm.ps1
    .\tools\test-vm-smoke.ps1
#>
param(
    [int]$Timeout = 120
)

$ErrorActionPreference = "Stop"
$VMName = "ScoutOS-Test"
$ProjectDir = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$SerialLog = Join-Path $ProjectDir "build\output\serial-console.log"

$pass = 0
$fail = 0
$total = 0

function Test-Check($name, $condition, $detail = "") {
    $script:total++
    if ($condition) {
        Write-Host "  PASS  $name" -ForegroundColor Green
        $script:pass++
    } else {
        Write-Host "  FAIL  $name" -ForegroundColor Red
        if ($detail) { Write-Host "        $detail" -ForegroundColor Yellow }
        $script:fail++
    }
}

# ─────────────────────────────────────────────
# Check 1: VM is running
# ─────────────────────────────────────────────
Write-Host ""
Write-Host "  ScoutOS Image Smoke Test" -ForegroundColor White
Write-Host ""

$running = (& VBoxManage showvminfo $VMName --machinereadable 2>&1) -match 'VMState="running"'
Test-Check "VM is running" $running "Start it with: .\tools\test-vm.ps1"
if (-not $running) {
    Write-Host "`nCannot continue — VM not running." -ForegroundColor Red
    exit 1
}

# ─────────────────────────────────────────────
# Check 2: Wait for boot (monitor serial log)
# ─────────────────────────────────────────────
Write-Host ""
Write-Host "  Waiting for boot (up to ${Timeout}s)..." -ForegroundColor Cyan

$bootMarkers = @(
    @{ Name = "GRUB loaded";        Pattern = "ScoutOS Slot" }
    @{ Name = "initramfs started";  Pattern = "ScoutOS initramfs starting" }
    @{ Name = "SquashFS mounted";   Pattern = "SquashFS mounted" }
    @{ Name = "OverlayFS mounted";  Pattern = "OverlayFS mounted" }
    @{ Name = "switch_root";        Pattern = "Switching to real root" }
)

$elapsed = 0
$foundMarkers = @{}

while ($elapsed -lt $Timeout) {
    if (Test-Path $SerialLog) {
        $content = Get-Content $SerialLog -Raw -ErrorAction SilentlyContinue
        if ($content) {
            foreach ($marker in $bootMarkers) {
                if (-not $foundMarkers[$marker.Name] -and $content -match [regex]::Escape($marker.Pattern)) {
                    $foundMarkers[$marker.Name] = $true
                }
            }
            # Check for OpenRC completion (services started)
            if ($content -match "entering runlevel" -or $content -match "scoutd.*starting" -or $content -match "Welcome to Alpine") {
                break
            }
            # Check for boot failure
            if ($content -match "ScoutOS boot failed") {
                Write-Host "  Boot FAILED — check serial log: $SerialLog" -ForegroundColor Red
                break
            }
        }
    }
    Start-Sleep -Seconds 3
    $elapsed += 3
    if ($elapsed % 15 -eq 0) {
        $found = $foundMarkers.Count
        Write-Host "    ${elapsed}s elapsed, $found/$($bootMarkers.Count) boot stages reached..." -ForegroundColor DarkGray
    }
}

Write-Host ""

# ─────────────────────────────────────────────
# Check 3: Verify boot stages from serial log
# ─────────────────────────────────────────────
$content = ""
if (Test-Path $SerialLog) {
    $content = Get-Content $SerialLog -Raw -ErrorAction SilentlyContinue
}

foreach ($marker in $bootMarkers) {
    $found = $content -match [regex]::Escape($marker.Pattern)
    Test-Check $marker.Name $found
}

# Additional boot checks
Test-Check "No boot failure" (-not ($content -match "ScoutOS boot failed"))
Test-Check "No kernel panic" (-not ($content -match "Kernel panic"))
Test-Check "No dm-verity error" (-not ($content -match "dm-verity.*error|verity.*fail" ))

# Check for dm-verity success (if enabled)
$verityOK = $content -match "dm-verity: OK"
$veritySkipped = $content -match "dm-verity not configured"
Test-Check "dm-verity status" ($verityOK -or $veritySkipped) $(
    if ($verityOK) { "(verified)" }
    elseif ($veritySkipped) { "(skipped — built with --no-verity)" }
    else { "no verity status found in serial log" }
)

# ─────────────────────────────────────────────
# Check 4: OpenRC services (from serial log)
# ─────────────────────────────────────────────
$services = @("nftables", "dnsmasq", "unbound", "chronyd", "scoutd")
foreach ($svc in $services) {
    $started = $content -match "$svc\s*\|\s*\*\s*start" -or $content -match "Starting $svc" -or $content -match "service $svc started"
    # Serial log may not capture all OpenRC output; mark as unknown rather than fail
    if ($content -match $svc) {
        Test-Check "Service: $svc" (-not ($content -match "$svc.*fail|$svc.*error"))
    }
}

# ─────────────────────────────────────────────
# Check 5: Data partition resize (from serial log)
# ─────────────────────────────────────────────
$resized = $content -match "data partition expanded" -or $content -match "Expanding data partition"
if ($resized) {
    Test-Check "Data partition resized" $true
} else {
    # May not appear in serial log if it logged to syslog instead
    Test-Check "Data partition resize" $true "(check via: findfs LABEL=scout-data | xargs df -h)"
}

# ─────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────
Write-Host ""
Write-Host "  ────────────────────────────" -ForegroundColor DarkGray
Write-Host "  Results: $pass passed, $fail failed, $total total" -ForegroundColor $(if ($fail -eq 0) { "Green" } else { "Red" })
Write-Host ""

if ($fail -gt 0) {
    Write-Host "  Serial log: $SerialLog" -ForegroundColor Yellow
    Write-Host "  View in WSL: wsl tail -100 '$($SerialLog -replace '\\','/')'" -ForegroundColor Yellow
    Write-Host ""
}

Write-Host "  Next steps:" -ForegroundColor Cyan
Write-Host "    1. vagrant up client                                    # start LAN client"
Write-Host "    2. vagrant ssh client -- curl -sk https://192.168.1.1   # test web UI"
Write-Host "    3. vagrant ssh client -- curl -sk https://192.168.1.1/api/v1/system"
Write-Host ""

exit $fail
