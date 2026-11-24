Import-Module (Join-Path $PSScriptRoot 'Core.psm1') -Force
$script:root=Join-Path $env:LOCALAPPDATA 'DesktopCommander'
$script:state=Join-Path $env:LOCALAPPDATA 'DCRManager-state'
$script:entry=Join-Path $script:root 'node_modules\@wonderwhy-er\desktop-commander\dist\index.js'
$script:manifest=Join-Path $script:root 'node_modules\@wonderwhy-er\desktop-commander\package.json'
Import-Module (Join-Path $PSScriptRoot 'Recovery.psm1') -Force
function Get-DcrProcesses {
    Get-CimInstance Win32_Process -Filter "Name='node.exe'" | Where-Object { Test-DcrCommand $_.CommandLine }
}
function Get-DcrRemote { Select-DcrRemote @(Get-DcrProcesses) }
function Get-DcrVersion {
    if(Test-Path -LiteralPath $script:manifest){try{(Get-Content -LiteralPath $script:manifest -Raw | ConvertFrom-Json).version}catch{'unknown'}}else{'absent'}
}
function Get-NpmCache {
    if(!(Get-Command npm.cmd -ErrorAction SilentlyContinue)){return}
    $out=@(& npm.cmd config get cache 2>$null)
    if($LASTEXITCODE -eq 0 -and $out.Count -eq 1 -and [IO.Path]::IsPathRooted($out[0])){return $out[0].Trim()}
}
function Get-DcrStatus {
    $all=@(Get-DcrProcesses);$remote=@(Select-DcrRemote $all);$cache=Get-NpmCache
    [pscustomobject]@{
        version=Get-DcrVersion; installed=((Test-Path -LiteralPath $script:entry) -and (Test-Path -LiteralPath $script:manifest))
        nodeAvailable=[bool](Get-Command node.exe -ErrorAction SilentlyContinue); npmAvailable=[bool](Get-Command npm.cmd -ErrorAction SilentlyContinue)
        remoteProcesses=$remote.Count; remoteRoots=@($remote | Where-Object {$remote.ProcessId -notcontains $_.ParentProcessId}).Count
        otherDcrProcesses=$all.Count-$remote.Count; cacheKnown=[bool]$cache; cacheCandidates=@(Find-DcrCache $cache).Count
    }
}
function Get-DcrLogSummary {
    $cache=Get-NpmCache
    if(!$cache){return}
    $logs=Join-Path $cache '_logs'
    if(!(Test-Path -LiteralPath $logs)){return}
    Get-ChildItem -LiteralPath $logs -Filter '*debug*.log' -File | Sort-Object LastWriteTime -Descending | Select-Object -First 100 | ForEach-Object {
        try {
            if(Select-String -LiteralPath $_.FullName -SimpleMatch '@wonderwhy-er/desktop-commander' -Quiet){
                [pscustomobject]@{name=$_.Name; codes=@(Get-DcrErrorCodes (Get-Content -LiteralPath $_.FullName))}
            }
        } catch {}
    } | Select-Object -First 5
}
function Get-DcrInventory {
    $all=@(Get-DcrProcesses);$remote=@(Select-DcrRemote $all)
    $all | ForEach-Object {
        [pscustomobject]@{pid=$_.ProcessId;parent=$_.ParentProcessId;role=$(if($remote.ProcessId -contains $_.ProcessId){'Remote'}else{'MCP'});location=$(if($_.CommandLine.IndexOf($script:root,[StringComparison]::OrdinalIgnoreCase) -ge 0){'Dedicated'}elseif($_.CommandLine -match '(?i)[\\/]_npx[\\/]'){'npx'}else{'Other'})}
    }
}
function Start-DcrRemote {
    if(@(Get-DcrRemote).Count){return 'Remote already active. No duplicate started.'}
    if(!(Test-Path -LiteralPath $script:entry)){throw 'Package missing. Install first.'}
    $node=(Get-Command node.exe -ErrorAction Stop).Source
    Start-Process -FilePath $node -ArgumentList ('"{0}" remote' -f $script:entry) -WorkingDirectory $script:root
    'Launch requested; check connection in the official device portal.'
}
function Stop-DcrRemote {
    $remote=@(Get-DcrRemote)
    foreach($p in $remote | Sort-Object CreationDate){
        $current=Get-CimInstance Win32_Process -Filter ("ProcessId="+$p.ProcessId) -ErrorAction SilentlyContinue
        if($current -and $current.CreationDate -eq $p.CreationDate -and $current.CommandLine -eq $p.CommandLine){Stop-Process -Id $p.ProcessId -ErrorAction SilentlyContinue}
    }
    Start-Sleep -Milliseconds 800
    if(@(Get-DcrRemote).Count){throw 'Remote still running; no further action taken.'}
    'Remote stopped.'
}
function Get-DcrCacheCandidates { Find-DcrCache (Get-NpmCache) }
function Move-DcrCache {
    param([Parameter(Mandatory)][string]$Path)
    $matches=@(Get-DcrCacheCandidates | Where-Object {$_.FullName -eq $Path})
    if($matches.Count -ne 1){throw 'Cache is not an exclusively identified DCR cache.'}
    $busy=@(Get-CimInstance Win32_Process | Where-Object {$_.CommandLine -and $_.ProcessId -ne $PID -and $_.CommandLine.IndexOf($Path,[StringComparison]::OrdinalIgnoreCase) -ge 0})
    if($busy.Count){throw ('Cache in use by PID(s): '+($busy.ProcessId -join ', '))}
    $target=Join-Path $script:state ('cache-backup-'+[guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $script:state -Force | Out-Null
    [pscustomobject]@{originalPath=$Path;backupPath=$target} | ConvertTo-Json | Set-Content -LiteralPath ($target+'.json') -Encoding UTF8 -ErrorAction Stop
    Move-Item -LiteralPath $Path -Destination $target -ErrorAction Stop
    $target
}
function Install-DcrPackage([switch]$Update) {
    if(@(Get-DcrProcesses).Count){throw 'Close DCR Remote and DCR MCP clients before changing packages.'}
    $version=Get-DcrVersion
    $spec=Get-DcrPackageSpec $version -Update:($Update -or $version -eq 'absent')
    $stage=Join-Path $script:state ('stage-'+[guid]::NewGuid().ToString('N'))
    $backup=Join-Path $script:state ('package-backup-'+[guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $stage -Force | Out-Null
    & npm.cmd install --prefix $stage $spec --save-exact --no-audit --no-fund | Out-Host
    if($LASTEXITCODE -ne 0 -or !(Test-Path -LiteralPath (Join-Path $stage 'node_modules\@wonderwhy-er\desktop-commander\dist\index.js'))){throw "Staged install failed. Original untouched: $stage"}
    if(@(Get-DcrProcesses).Count){throw 'DCR started during staging; original untouched.'}
    Switch-DcrPackageFiles $script:root $stage $backup
    "Installed $(Get-DcrVersion). Previous files: $backup"
}
function Get-DcrBackups {
    if(Test-Path -LiteralPath $script:state){Get-ChildItem -LiteralPath $script:state -Directory | Where-Object {$_.Name -match '^(cache|package)-backup-'} | Select-Object Name,LastWriteTime}
}
function Export-DcrReport {
    $dir=Join-Path $script:state 'reports'
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    $path=Join-Path $dir ('diagnostic-'+(Get-Date -Format 'yyyyMMdd-HHmmss')+'.json')
    [pscustomobject]@{schema=1;managerVersion='1.2.1';status=Get-DcrStatus;processes=@(Get-DcrInventory);npmLogs=@(Get-DcrLogSummary)} | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $path -Encoding UTF8
    $path
}
function Restore-DcrManagedCache([string]$Name){
    Restore-DcrCacheBackup $script:state (Get-NpmCache) $Name {
        param($source,$target)
        $busy=@(Get-CimInstance Win32_Process | Where-Object {$_.CommandLine -and $_.ProcessId -ne $PID -and ($_.CommandLine.IndexOf($source,[StringComparison]::OrdinalIgnoreCase) -ge 0 -or $_.CommandLine.IndexOf($target,[StringComparison]::OrdinalIgnoreCase) -ge 0)})
        if($busy.Count){throw 'Cache referenced by active processes. Restore refused.'}
    }
}
function Get-DcrProcessHealth {
    $all=@(Get-CimInstance Win32_Process)
    $dcr=@($all|Where-Object {$_.Name -eq 'node.exe' -and (Test-DcrCommand $_.CommandLine)})
    Get-DcrParentAssessment $all $dcr
}
function Get-DcrBackupReview { Get-DcrRetentionAssessment @(Get-DcrBackups) }
Export-ModuleMember -Function Get-DcrStatus,Get-DcrLogSummary,Get-DcrInventory,Start-DcrRemote,Stop-DcrRemote,Get-DcrCacheCandidates,Move-DcrCache,Install-DcrPackage,Get-DcrBackups,Export-DcrReport,Restore-DcrManagedCache,Get-DcrProcessHealth,Get-DcrBackupReview
