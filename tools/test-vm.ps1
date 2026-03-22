#Requires -Version 5.1
<#
.SYNOPSIS
    Boot the ScoutOS disk image in a VirtualBox UEFI VM for testing.

.DESCRIPTION
    Creates a VirtualBox VM from the built ScoutOS disk image and boots it.
    The VM joins the "scoutlan" internal network so existing Vagrant client
    VMs can test against it.

    Prerequisites:
      - VirtualBox installed (VBoxManage in PATH)
      - Disk image built: build\output\scoutos-*.img or .img.zst

.PARAMETER Version
    Firmware version string matching the built image (default: 0.1.0).

.PARAMETER DiskSize
    Size to expand the virtual disk to, in MB (default: 8192 = 8GB).
    Simulates a real NVMe so the first-boot data partition resize runs.

.PARAMETER Memory
    VM RAM in MB (default: 4096).

.PARAMETER CPUs
    VM CPU count (default: 2).

.PARAMETER Headless
    Start the VM headless (no GUI window). View via VBoxManage or serial log.

.PARAMETER Destroy
    Delete an existing test VM before creating a new one.

.EXAMPLE
    .\tools\test-vm.ps1
    .\tools\test-vm.ps1 -Destroy
    .\tools\test-vm.ps1 -Headless
    .\tools\test-vm.ps1 -Version 1.0.0
#>
param(
    [string]$Version = "0.1.0",
    [int]$DiskSize = 8192,
    [int]$Memory = 4096,
    [int]$CPUs = 2,
    [switch]$Headless,
    [switch]$Destroy
)

$ErrorActionPreference = "Stop"
$VMName = "ScoutOS-Test"
$ProjectDir = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$OutputDir = Join-Path $ProjectDir "build\output"
$ImgPath = Join-Path $OutputDir "scoutos-$Version.img"
$ImgZst = "$ImgPath.zst"
$VDIPath = Join-Path $OutputDir "scoutos-test.vdi"
$SerialLog = Join-Path $OutputDir "serial-console.log"

# --- Helpers ---

function Write-Step($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Write-Detail($msg) { Write-Host "    $msg" }

function Test-VBoxManage {
    try { $null = & VBoxManage --version 2>&1; return $true }
    catch { return $false }
}

function Get-VMExists($name) {
    $vms = & VBoxManage list vms 2>&1
    return ($vms -match [regex]::Escape("`"$name`""))
}

function Remove-TestVM {
    if (Get-VMExists $VMName) {
        Write-Step "Removing existing VM: $VMName"
        try { & VBoxManage controlvm $VMName poweroff 2>&1 | Out-Null } catch {}
        Start-Sleep -Seconds 2
        & VBoxManage unregistervm $VMName --delete 2>&1 | Out-Null
    }
    if (Test-Path $VDIPath) {
        Remove-Item $VDIPath -Force
    }
}

# --- Preflight ---

Write-Host ""
Write-Host "  ScoutOS VM Test Runner" -ForegroundColor White
Write-Host ""

if (-not (Test-VBoxManage)) {
    Write-Host "ERROR: VBoxManage not found. Install VirtualBox or add it to PATH." -ForegroundColor Red
    exit 1
}

# --- Handle -Destroy ---

if ($Destroy) {
    Remove-TestVM
    if ((-not (Test-Path $ImgPath)) -and (-not (Test-Path $ImgZst))) {
        Write-Host "VM destroyed. No image found to rebuild - done."
        exit 0
    }
}

# --- If VM already exists, just start it ---

if (Get-VMExists $VMName) {
    Write-Step "VM '$VMName' already exists - starting it"
    if ($Headless) { $startType = "headless" } else { $startType = "gui" }
    & VBoxManage startvm $VMName --type $startType
    Write-Host ""
    Write-Detail "Serial console log: $SerialLog"
    Write-Detail "Web UI:             https://192.168.1.1 (from client VM)"
    Write-Detail "Stop VM:            VBoxManage controlvm $VMName poweroff"
    Write-Host ""
    exit 0
}

# --- Locate and decompress image ---

Write-Step "Locating disk image..."

if (-not (Test-Path $ImgPath)) {
    if (Test-Path $ImgZst) {
        Write-Detail "Decompressing $ImgZst..."
        & zstd -d $ImgZst -o $ImgPath --force
        if ($LASTEXITCODE -ne 0) {
            Write-Host "ERROR: zstd decompression failed" -ForegroundColor Red
            exit 1
        }
    }
    else {
        Write-Host "ERROR: No image found at $ImgPath or $ImgZst" -ForegroundColor Red
        Write-Host "       Run: wsl ./tools/build-image.sh" -ForegroundColor Yellow
        exit 1
    }
}

$imgSizeMB = [math]::Round((Get-Item $ImgPath).Length / 1MB)
Write-Detail "Image: $ImgPath ($imgSizeMB MB)"

# --- Convert raw image to VDI ---

Write-Step "Converting raw image to VDI..."

if (Test-Path $VDIPath) {
    Remove-Item $VDIPath -Force
}
& VBoxManage convertfromraw $ImgPath $VDIPath --format VDI
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: VBoxManage convertfromraw failed" -ForegroundColor Red
    exit 1
}

# Resize VDI to simulate real NVMe (triggers first-boot partition resize)
Write-Step "Resizing VDI to ${DiskSize}MB..."
& VBoxManage modifymedium disk $VDIPath --resize $DiskSize

# --- Create VM ---

Write-Step "Creating VM: $VMName"

& VBoxManage createvm --name $VMName --ostype "Linux_64" --register
& VBoxManage modifyvm $VMName --firmware efi --cpus $CPUs --memory $Memory --vram 16 --graphicscontroller vmsvga --audio-enabled off

# NIC1 = WAN (NAT), NIC2 = LAN (internal "scoutlan" - matches Vagrant client VMs)
Write-Step "Configuring network..."
& VBoxManage modifyvm $VMName --nic1 nat --nictype1 virtio --nic2 intnet --intnet2 "scoutlan" --nictype2 virtio

# Storage
Write-Step "Attaching storage..."
& VBoxManage storagectl $VMName --name "SATA" --add sata --controller IntelAhci --portcount 1
& VBoxManage storageattach $VMName --storagectl "SATA" --port 0 --device 0 --type hdd --medium $VDIPath

# Serial console (captures boot messages to log file)
Write-Step "Setting up serial console..."
& VBoxManage modifyvm $VMName --uart1 0x3F8 4 --uart-mode1 file $SerialLog

# --- Boot ---

Write-Step "Starting VM..."
if ($Headless) { $startType = "headless" } else { $startType = "gui" }
& VBoxManage startvm $VMName --type $startType

Write-Host ""
Write-Host "  VM started!" -ForegroundColor Green
Write-Host ""
Write-Detail "Serial console log: $SerialLog"
Write-Host ""
Write-Detail "Once booted, test from Vagrant client VM:"
Write-Detail "  vagrant up client"
Write-Detail "  vagrant ssh client -- curl -sk https://192.168.1.1/api/v1/system"
Write-Host ""
Write-Detail "VM management:"
Write-Detail "  Stop:     VBoxManage controlvm $VMName poweroff"
Write-Detail "  Restart:  VBoxManage controlvm $VMName reset"
Write-Detail "  Delete:   .\tools\test-vm.ps1 -Destroy"
Write-Host ""
