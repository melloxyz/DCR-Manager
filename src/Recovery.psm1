function Test-DcrCacheBackup([string]$Path){
    try{
        if(!(Test-Path -LiteralPath $Path -PathType Container)){return $false}
        if((Get-Item -LiteralPath $Path).Attributes -band [IO.FileAttributes]::ReparsePoint){return $false}
        $j=Get-Content -LiteralPath (Join-Path $Path 'package.json') -Raw -ErrorAction Stop | ConvertFrom-Json -ErrorAction Stop
        $deps=@($j.dependencies.PSObject.Properties.Name)
        return ($deps.Count -eq 1 -and $deps[0] -eq '@wonderwhy-er/desktop-commander' -and (Test-Path -LiteralPath (Join-Path $Path 'node_modules\@wonderwhy-er\desktop-commander') -PathType Container))
    }catch{return $false}
}
function Get-DcrCacheRestoreTarget([string]$State,[string]$CacheRoot,[string]$Name){
    if($Name -notmatch '^cache-backup-[a-f0-9]{32}$'){throw 'Invalid backup name.'}
    if(!$CacheRoot -or ![IO.Path]::IsPathRooted($CacheRoot)){throw 'npm cache unavailable.'}
    $backup=Join-Path $State $Name
    $metadata=$backup+'.json'
    if(!(Test-DcrCacheBackup $backup)){throw 'Backup is not an exclusive DCR cache.'}
    foreach($path in @($State,$CacheRoot,(Join-Path $CacheRoot '_npx'),$metadata)){
        if(!(Test-Path -LiteralPath $path)){throw 'Required path missing.'}
        if((Get-Item -LiteralPath $path).Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'Reparse point not supported.'}
    }
    try{$j=Get-Content -LiteralPath $metadata -Raw -ErrorAction Stop | ConvertFrom-Json -ErrorAction Stop}catch{throw 'Backup metadata unreadable.'}
    if($j.backupPath -ne $backup -or ![IO.Path]::IsPathRooted([string]$j.originalPath)){throw 'Backup metadata mismatch.'}
    $target=[IO.Path]::GetFullPath([string]$j.originalPath)
    $parent=[IO.Path]::GetFullPath((Join-Path $CacheRoot '_npx'))
    if([IO.Path]::GetDirectoryName($target) -ne $parent -or [IO.Path]::GetFileName($target) -notmatch '^[a-f0-9]{8,64}$'){throw 'Original cache path is outside current npm cache or invalid.'}
    if(Test-Path -LiteralPath $target){throw 'Original cache already exists. No overwrite allowed.'}
    return $target
}
function Restore-DcrCacheBackup([string]$State,[string]$CacheRoot,[string]$Name,[scriptblock]$BeforeMove={}){
    $target=Get-DcrCacheRestoreTarget $State $CacheRoot $Name
    $source=Join-Path $State $Name
    & $BeforeMove $source $target
    if(Test-Path -LiteralPath $target){throw 'Original cache appeared during restore.'}
    Move-Item -LiteralPath $source -Destination $target -ErrorAction Stop
    # Keep metadata as an audit record. No deletion of other backups.
    return $target
}
function Get-DcrParentAssessment($Processes,$DcrProcesses){
    $map=@{}
    foreach($p in $Processes){$map[[string]$p.ProcessId]=$p}
    foreach($p in $DcrProcesses){
        $parent=$map[[string]$p.ParentProcessId]
        $status=if(!$parent){'parent-missing'}elseif($parent.CreationDate -and $p.CreationDate -and $parent.CreationDate -gt $p.CreationDate){'parent-pid-reused'}else{'parent-present'}
        [pscustomobject]@{pid=$p.ProcessId;parent=$p.ParentProcessId;assessment=$status}
    }
}
function Get-DcrRetentionAssessment($Backups,[datetime]$Now=(Get-Date),[int]$Days=30){
    foreach($b in $Backups){
        $age=[math]::Max(0,[math]::Floor(($Now-$b.LastWriteTime).TotalDays))
        [pscustomobject]@{name=$b.Name;ageDays=$age;review=($age -ge $Days)}
    }
}
Export-ModuleMember -Function Test-DcrCacheBackup,Get-DcrCacheRestoreTarget,Restore-DcrCacheBackup,Get-DcrParentAssessment,Get-DcrRetentionAssessment
