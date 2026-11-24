$ErrorActionPreference='Stop'
$WarningPreference='SilentlyContinue'
$ProgressPreference='SilentlyContinue'
[Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false)
Import-Module (Join-Path $PSScriptRoot 'Operations.psm1') -Force
Import-Module (Join-Path $PSScriptRoot 'Maintenance.psm1') -Force
[pscustomobject]@{
    status=Get-DcrStatus
    runtime=Get-DcrRuntime
    installations=@(Get-DcrInstallInventory)
    processes=@(Get-DcrInventory)
    health=@(Get-DcrProcessHealth)
    clients=@(Get-DcrClientAudit)
    logs=@(Get-DcrLogSummary)
    backups=@(Get-DcrBackupReview)
} | ConvertTo-Json -Depth 8 -Compress
