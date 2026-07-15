[CmdletBinding()]
param(
    [string]$TaskName = "Tonkic API Auto Update",
    [string]$DailyAt = "04:00",
    [switch]$RebuildDocker
)

$ErrorActionPreference = "Stop"
$updateScript = Join-Path $PSScriptRoot "auto-update.ps1"
if (-not (Test-Path -LiteralPath $updateScript)) {
    throw "Update script not found: $updateScript"
}

$time = [datetime]::ParseExact($DailyAt, "HH:mm", [Globalization.CultureInfo]::InvariantCulture)
$arguments = "-NoProfile -ExecutionPolicy Bypass -File `"$updateScript`""
if ($RebuildDocker) {
    $arguments += " -RebuildDocker"
}

$action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument $arguments
$trigger = New-ScheduledTaskTrigger -Daily -At $time
$settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -MultipleInstances IgnoreNew

Register-ScheduledTask `
    -TaskName $TaskName `
    -Action $action `
    -Trigger $trigger `
    -Settings $settings `
    -Description "Update Tonkic API and re-merge QuantumNous/new-api PR #5062." `
    -Force | Out-Null

Write-Host "Scheduled task '$TaskName' installed. It runs daily at $DailyAt."
if (-not $RebuildDocker) {
    Write-Host "It updates source code only. Re-run with -RebuildDocker to build and restart the new-api container."
}
