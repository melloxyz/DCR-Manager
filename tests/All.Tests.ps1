$ErrorActionPreference='Stop'
foreach($name in @('Run-Tests.ps1','Maintenance.Tests.ps1','Reliability.Tests.ps1','PackageCache.Tests.ps1')){
    & (Get-Process -Id $PID).Path -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot $name)
    if($LASTEXITCODE -ne 0){exit $LASTEXITCODE}
}
