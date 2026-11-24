param(
    [Parameter(Mandatory)][ValidateSet('start','stop','restart')][string]$Action,
    [Parameter(Mandatory)][ValidatePattern('^(none|[1-9][0-9]*(,[1-9][0-9]*)*)$')][string]$Expected
)
$ErrorActionPreference='Stop'
$WarningPreference='SilentlyContinue'
$ProgressPreference='SilentlyContinue'
Import-Module (Join-Path $PSScriptRoot 'Operations.psm1') -Force
Import-Module (Join-Path $PSScriptRoot 'Core.psm1') -Force
function Read-Remote {
    Select-DcrRemote @(Get-CimInstance Win32_Process -Filter "Name='node.exe'" -ErrorAction Stop)
}
function Get-Identity($list) {
    $ids=@($list | ForEach-Object ProcessId | Sort-Object)
    if(!$ids.Count){return 'none'}
    return ($ids -join ',')
}
$mutex=New-Object System.Threading.Mutex($false,'Local\DCRManager')
$owned=$false
try{
    try{$owned=$mutex.WaitOne(0)}catch [System.Threading.AbandonedMutexException]{$owned=$true}
    if(!$owned){throw 'Another maintenance manager is open.'}
    $targets=@(Read-Remote)
    if((Get-Identity $targets) -ne $Expected){throw 'Remote process set changed; refresh before acting.'}
    if($Action -eq 'start' -and $targets.Count){throw 'Remote already active.'}
    if($Action -ne 'start' -and !$targets.Count){throw 'No Remote process to stop.'}
    if($Action -ne 'stop'){
        $entry=Join-Path $env:LOCALAPPDATA 'DesktopCommander\node_modules\@wonderwhy-er\desktop-commander\dist\index.js'
        if(!(Test-Path -LiteralPath $entry -PathType Leaf)){throw 'Dedicated install missing.'}
        [void](Get-Command node.exe -ErrorAction Stop)
    }
    if($Action -ne 'start'){
        # Capture and revalidate only the previously confirmed DCR identities.
        foreach($p in $targets | Sort-Object CreationDate){
            $current=Get-CimInstance Win32_Process -Filter ("ProcessId="+$p.ProcessId) -ErrorAction SilentlyContinue
            if(!$current){continue}
            if($current.CreationDate -ne $p.CreationDate -or $current.CommandLine -ne $p.CommandLine){throw 'Process identity changed.'}
            Stop-Process -Id $p.ProcessId -ErrorAction Stop
        }
        Start-Sleep -Milliseconds 800
        if(@(Read-Remote).Count){throw 'Remote still active; restart refused.'}
    }
    if($Action -ne 'stop'){
        if(@(Read-Remote).Count){throw 'Remote started elsewhere.'}
        Start-DcrRemote | Out-Null
        $deadline=(Get-Date).AddSeconds(8)
        do {
            Start-Sleep -Milliseconds 400
            $active=@(Read-Remote)
        }while(!$active.Count -and (Get-Date) -lt $deadline)
        if(!$active.Count){throw 'Remote launch not observed.'}
    }
    Write-Output 'OK'
}finally{
    if($owned){$mutex.ReleaseMutex()}
    $mutex.Dispose()
}
