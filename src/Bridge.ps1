param([switch]$NativeDiagnostics,[switch]$NativeRuntime,[switch]$NativeInventory)
if($NativeInventory){$NativeRuntime=$true}
if($NativeRuntime){$NativeDiagnostics=$true}
$ErrorActionPreference='Stop'
$WarningPreference='SilentlyContinue'
$ProgressPreference='SilentlyContinue'
[Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false)
Import-Module (Join-Path $PSScriptRoot 'Operations.psm1') -Force
Import-Module (Join-Path $PSScriptRoot 'Maintenance.psm1') -Force
Import-Module (Join-Path $PSScriptRoot 'PackageCache.psm1') -Force
Import-Module (Join-Path $PSScriptRoot 'RecoveryCenter.psm1')
Import-Module (Join-Path $PSScriptRoot 'Diagnostics.psm1')
$cacheRoot="";$caches=@()
try{ $npm=Get-Command npm.cmd -ErrorAction Stop; $cache=@(& $npm.Source config get cache 2>$null); if($LASTEXITCODE -eq 0 -and $cache.Count -eq 1){$cacheRoot=$cache[0].Trim();$caches=@(Get-DcrT6Caches $cacheRoot | Select-Object id,version,token)} }catch{}
$recoveryKnown=$false;$backups=@()
try{$backups=@(Get-DcrRecoveryInventory (Join-Path $env:LOCALAPPDATA 'DCRManager-state') $cacheRoot);$recoveryKnown=$true}catch{}
$logScan=if($NativeDiagnostics){[pscustomobject]@{known=$false;logs=@()}}else{Get-DcrBoundedLogs $cacheRoot}
$result=[ordered]@{
    logsKnown=$logScan.known
    recoveryKnown=$recoveryKnown
    caches=$caches
    status=Get-DcrStatus
    runtime=$(if(!$NativeRuntime){Get-DcrRuntime})
    installations=@(if(!$NativeInventory){Get-DcrInstallInventory})
    processes=@(Get-DcrInventory)
    health=@(Get-DcrProcessHealth)
    clients=@(if(!$NativeRuntime){Get-DcrClientAudit -IncludeMissing -OnlyToml:$NativeDiagnostics})
    logs=@($logScan.logs)
    backups=$backups
}
if($NativeDiagnostics){$result.nativeDiagnosticsSchema=$(if($NativeInventory){3}elseif($NativeRuntime){2}else{1});$result.internalCacheRoot=$cacheRoot}
[pscustomobject]$result | ConvertTo-Json -Depth 8 -Compress
