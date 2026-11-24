param(
 [Parameter(Mandatory)][ValidateSet('repair','update','quarantine','restore')][string]$Action,
 [Parameter(Mandatory)][ValidatePattern('^(absent|unknown|[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?|[a-f0-9]{64})$')][string]$Expected,
 [ValidatePattern('^([a-f0-9]{8,64}|(?:package|cache)-backup-[a-f0-9]{32})?$')][string]$CacheID='',
 [ValidatePattern('^(absent|unknown|[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?)?$')][string]$CurrentVersion=''
)
$ErrorActionPreference='Stop'
$WarningPreference='SilentlyContinue'
$ProgressPreference='SilentlyContinue'
[Console]::OutputEncoding=New-Object Text.UTF8Encoding($false)
Import-Module (Join-Path $PSScriptRoot 'PackageCache.psm1') -Force
$root=Join-Path $env:LOCALAPPDATA 'DesktopCommander'
$state=Join-Path $env:LOCALAPPDATA 'DCRManager-state'
$mutex=New-Object System.Threading.Mutex($false,'Local\DCRManager')
$owned=$false
$code=''
try{
 try{$owned=$mutex.WaitOne(0)}catch [System.Threading.AbandonedMutexException]{$owned=$true}
 if(!$owned){throw 'LOCKED'}
 if($Action -eq 'restore'){
    Import-Module (Join-Path $PSScriptRoot 'RecoveryCenter.psm1')
    $cacheRoot=''
    if($CacheID.StartsWith('cache-backup-')){
        $npm=(Get-Command npm.cmd -ErrorAction Stop).Source
        $out=@(& $npm config get cache 2>$null)
        if($LASTEXITCODE -ne 0 -or $out.Count -ne 1 -or ![IO.Path]::IsPathRooted($out[0])){throw 'UNSAFE'}
        $cacheRoot=$out[0].Trim()
    }elseif((Get-DcrT6Version $root) -ne $CurrentVersion){throw 'STALE'}
    Invoke-DcrRecoveryRestore $root $state $cacheRoot $CacheID $Expected {
        param($path)
        foreach($p in Get-CimInstance Win32_Process -ErrorAction Stop){
            if($p.ProcessId -eq $PID){continue}
            if($p.Name -eq 'node.exe' -and (!$p.CommandLine -or $p.CommandLine -match '(?i)@wonderwhy-er[\\/]desktop-commander[\\/]|node_modules[\\/]\.bin[\\/]desktop-commander(?:\s|"|$)')){throw 'ACTIVE'}
            if($p.CommandLine -and $p.CommandLine.IndexOf($path,[StringComparison]::OrdinalIgnoreCase) -ge 0){throw 'ACTIVE'}
        }
        if($CacheID.StartsWith('package-backup-') -and (Get-DcrT6Version $root) -ne $CurrentVersion){throw 'STALE'}
    }
 }elseif($Action -eq 'quarantine'){
    $npm=(Get-Command npm.cmd -ErrorAction Stop).Source
    $out=@(& $npm config get cache 2>$null)
    if($LASTEXITCODE -ne 0 -or $out.Count -ne 1 -or ![IO.Path]::IsPathRooted($out[0])){throw 'UNSAFE'}
    Move-DcrT6Cache $out[0].Trim() $state $CacheID $Expected {
        param($path)
        $processes=@(Get-CimInstance Win32_Process -ErrorAction Stop)
        foreach($p in $processes){
            if($p.ProcessId -eq $PID){continue}
            if($p.Name -eq 'node.exe' -and !$p.CommandLine){throw 'ACTIVE'}
            if($p.CommandLine -and $p.CommandLine.IndexOf($path,[StringComparison]::OrdinalIgnoreCase) -ge 0){throw 'ACTIVE'}
        }
    }
 }else{
    $npm=(Get-Command npm.cmd -ErrorAction Stop).Source
    Invoke-DcrT6Package $root $state $Action $Expected {
        param($path)
        foreach($p in Get-CimInstance Win32_Process -Filter "Name='node.exe'" -ErrorAction Stop){
            if(!$p.CommandLine -or $p.CommandLine -match '(?i)@wonderwhy-er[\\/]desktop-commander[\\/]|node_modules[\\/]\.bin[\\/]desktop-commander(?:\s|"|$)' -or $p.CommandLine.IndexOf($path,[StringComparison]::OrdinalIgnoreCase) -ge 0){throw 'ACTIVE'}
        }
    } {
        param($stage,$spec)
        $cache=Join-Path $stage 'npm-cache'
        $log=Join-Path $stage 'npm-install.log'
        $old=$ErrorActionPreference
        try{
            $ErrorActionPreference='Continue'
            & $npm install --prefix $stage $spec --cache $cache --save-exact --no-audit --no-fund --fetch-retries 1 --fetch-timeout 30000 *> $log
            $exit=$LASTEXITCODE
        }finally{$ErrorActionPreference=$old}
        if($exit -ne 0){
            $match=Select-String -LiteralPath $log -Pattern '\b(EBUSY|EPERM|EACCES)\b'|Select-Object -First 1
            if($match){throw $match.Matches[0].Value}
            throw 'NPM'
        }
    }
 }
 [pscustomobject]@{ok=$true;code='OK'}|ConvertTo-Json -Compress
}catch{
 $code=Get-DcrT6ErrorCode $_
 [pscustomobject]@{ok=$false;code=$code}|ConvertTo-Json -Compress
 exit 1
}finally{if($owned){$mutex.ReleaseMutex()};$mutex.Dispose()}
