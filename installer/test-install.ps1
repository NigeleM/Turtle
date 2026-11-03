# Installs Turtle with the Windows installer, quietly, and checks what a
# user gets: the build of turtle.exe the PC runs natively (x64 or Arm64),
# Turtle on the PATH, .turtle files opening with Turtle, and a program
# running. Then it uninstalls and checks all of that is gone. CI runs it on
# an x64 PC and on an Arm PC.
#
#   installer/test-install.ps1 <turtle-windows-setup.exe> <x64 | arm64>

param(
	[Parameter(Mandatory = $true)] [string]$Setup,
	[Parameter(Mandatory = $true)] [ValidateSet("x64", "arm64")] [string]$Arch
)
$ErrorActionPreference = "Stop"

$app = Join-Path $env:ProgramFiles "Turtle"
$exe = Join-Path $app "turtle.exe"
$log = Join-Path ([IO.Path]::GetTempPath()) "turtle-install.log"

function OnPath { ([Environment]::GetEnvironmentVariable("Path", "Machine") -split ";") -contains $app }
function OpensTurtle { (Get-ItemProperty "HKLM:\Software\Classes\.turtle" -ErrorAction SilentlyContinue).'(default)' -eq "Turtle.Script" }

$p = Start-Process $Setup -ArgumentList "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", "/SP-", "/LOG=`"$log`"" -Wait -PassThru
if ($p.ExitCode -ne 0) {
	Get-Content $log -Tail 40
	throw "the installer failed (exit code $($p.ExitCode))"
}

# The PE header says which processor a program is for.
$bytes = [IO.File]::ReadAllBytes($exe)
$machine = [BitConverter]::ToUInt16($bytes, [BitConverter]::ToInt32($bytes, 0x3C) + 4)
$want = @{ x64 = 0x8664; arm64 = 0xAA64 }[$Arch]
if ($machine -ne $want) { throw ("installed turtle.exe is for machine 0x{0:X4}, not {1} (0x{2:X4})" -f $machine, $Arch, $want) }
"installed turtle.exe is the $Arch build"

& $exe version
if ($LASTEXITCODE -ne 0) { throw "turtle version failed" }
& $exe testdata/programs/multiline.turtle
if ($LASTEXITCODE -ne 0) { throw "turtle failed to run testdata/programs/multiline.turtle" }
if (-not (OnPath)) { throw "$app isn't on the system PATH" }
if (-not (OpensTurtle)) { throw ".turtle files don't open with Turtle" }
"PATH and the .turtle file type are set"

# The uninstaller hands over to a copy of itself and returns at once, so
# wait until it has removed itself, which it does last.
$uninstaller = Join-Path $app "unins000.exe"
Start-Process $uninstaller -ArgumentList "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART" -Wait
for ($i = 0; $i -lt 120 -and (Test-Path $uninstaller); $i++) { Start-Sleep -Milliseconds 500 }
if (Test-Path $exe) { throw "uninstalling left $exe" }
if (OnPath) { throw "uninstalling left $app on the PATH" }
if (OpensTurtle) { throw "uninstalling left .turtle opening with Turtle" }
"uninstalled cleanly"
