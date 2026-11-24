$ErrorActionPreference='Stop'
$WarningPreference='SilentlyContinue'
$ProgressPreference='SilentlyContinue'
[Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false)
Import-Module (Join-Path $PSScriptRoot 'Operations.psm1') -Force
Import-Module (Join-Path $PSScriptRoot 'Maintenance.psm1') -Force
Import-Module (Join-Path $PSScriptRoot 'PackageCache.psm1') -Force
$caches=@()
try{ $npm=Get-Command npm.cmd -ErrorAction Stop; $cache=@(& $npm.Source config get cache 2>$null); if($LASTEXITCODE -eq 0 -and $cache.Count -eq 1){$caches=@(Get-DcrT6Caches $cache[0].Trim() | Select-Object id,version,token)} }catch{}
[pscustomobject]@{
    caches=$caches
    status=Get-DcrStatus
    runtime=Get-DcrRuntime
    installations=@(Get-DcrInstallInventory)
    processes=@(Get-DcrInventory)
    health=@(Get-DcrProcessHealth)
    clients=@(Get-DcrClientAudit)
    logs=@(Get-DcrLogSummary)
    backups=@(Get-DcrBackupReview)
} | ConvertTo-Json -Depth 8 -Compress
