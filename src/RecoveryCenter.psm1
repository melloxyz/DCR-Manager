Import-Module (Join-Path $PSScriptRoot 'Core.psm1')
Import-Module (Join-Path $PSScriptRoot 'PackageCache.psm1')
Import-Module (Join-Path $PSScriptRoot 'Recovery.psm1')
Import-Module (Join-Path $PSScriptRoot 'Maintenance.psm1')

function Measure-DcrRecoveryTree([string]$Path,[datetime]$Deadline){
    Assert-DcrPlainPath $Path
    $stack=New-Object 'System.Collections.Generic.Stack[string]'
    $stack.Push($Path);[long]$bytes=0;[long]$count=0
    while($stack.Count){
        $dir=$stack.Pop()
        foreach($item in Get-ChildItem -LiteralPath $dir -Force -ErrorAction Stop){
            $count++
            if($count -gt 100000 -or [datetime]::UtcNow -gt $Deadline){throw 'LIMIT'}
            if($item.Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'UNSAFE'}
            if($item.PSIsContainer){$stack.Push($item.FullName)}else{$bytes+=$item.Length}
        }
    }
    [pscustomobject]@{bytes=$bytes;count=$count}
}
function Get-DcrRecoveryToken($Item,$Measure,[string]$Version,[string]$Metadata){
    $parts=@($Item.FullName.ToLowerInvariant(),$Item.CreationTimeUtc.Ticks,$Item.LastWriteTimeUtc.Ticks,$Measure.bytes,$Measure.count,$Version)
    foreach($n in @('package.json','package-lock.json','node_modules\@wonderwhy-er\desktop-commander\package.json')){
        $p=Join-Path $Item.FullName $n
        if(Test-Path -LiteralPath $p -PathType Leaf){$parts+=(Get-FileHash -LiteralPath $p -Algorithm SHA256 -ErrorAction Stop).Hash}
    }
    if($Metadata){$parts+=(Get-FileHash -LiteralPath $Metadata -Algorithm SHA256 -ErrorAction Stop).Hash}
    $sha=[Security.Cryptography.SHA256]::Create()
    try{([BitConverter]::ToString($sha.ComputeHash([Text.Encoding]::UTF8.GetBytes(($parts -join '|'))))).Replace('-','').ToLowerInvariant()}finally{$sha.Dispose()}
}
function Get-DcrRecoveryItem([string]$State,[string]$CacheRoot,[string]$Name,[datetime]$Deadline){
    if($Name -notmatch '^(package-backup|cache-backup|stage|restore-stage)-[a-f0-9]{32}$'){throw 'UNSAFE'}
    $path=Join-Path $State $Name
    $kind=if($Name.StartsWith('package-backup-')){'package'}elseif($Name.StartsWith('cache-backup-')){'cache'}else{'stage'}
    $r=[ordered]@{name=$Name;kind=$kind;version='';status='incomplete';token='';bytes=0;sizeKnown=$false;ageDays=0;review=$false}
    try{
        Assert-DcrPlainPath $State;Assert-DcrPlainPath $path
        $item=Get-Item -LiteralPath $path -Force -ErrorAction Stop
        if(!$item.PSIsContainer){throw 'UNSAFE'}
        $r.ageDays=[int][math]::Max(0,[math]::Floor(((Get-Date)-$item.LastWriteTime).TotalDays))
        $r.review=$r.ageDays -ge 30
        try{$measure=Measure-DcrRecoveryTree $path $Deadline;$r.bytes=$measure.bytes;$r.sizeKnown=$true}catch{if($_.Exception.Message -eq 'LIMIT'){$r.status='not-assessed'}else{$r.status='unsafe'};return [pscustomobject]$r}
        if($kind -eq 'stage'){
            $r.status='unclassified-stage'
            $journal=Join-Path $path 'transaction.json'
            if(Test-Path -LiteralPath $journal -PathType Leaf){
                $j=Get-Content -LiteralPath $journal -Raw -ErrorAction Stop|ConvertFrom-Json -ErrorAction Stop
                if($j.state -in @('switching','completed','rollback-incomplete','switch-failed')){$r.status=$j.state}
            }
            return [pscustomobject]$r
        }
        Assert-DcrPackageOwnership $path
        $v=Get-DcrT6Version $path
        if($v -in @('absent','unknown')){return [pscustomobject]$r}
        $r.version=$v
        if(!(Test-Path -LiteralPath (Join-Path $path 'node_modules\@wonderwhy-er\desktop-commander\dist\index.js') -PathType Leaf)){return [pscustomobject]$r}
        if($kind -eq 'package'){
            if((Get-DcrBackupVersion $path) -ne $v){return [pscustomobject]$r}
            $meta=''
        }else{
            $meta=$path+'.json'
            Assert-DcrPlainPath $meta
            try{
                $target=Get-DcrCacheRestoreTarget $State $CacheRoot $Name
                Assert-DcrPlainPath $target
                if([IO.Path]::GetPathRoot($target) -ne [IO.Path]::GetPathRoot($path)){throw 'UNSAFE'}
            }catch{$r.status='destination-blocked';return [pscustomobject]$r}
        }
        $r.token=Get-DcrRecoveryToken $item $measure $v $meta
        $r.status='ready'
    }catch{$r.status='unsafe'}
    [pscustomobject]$r
}
function Get-DcrRecoveryInventory([string]$State,[string]$CacheRoot){
    Assert-DcrPlainPath $State
    if(!(Test-Path -LiteralPath $State)){return}
    $deadline=[datetime]::UtcNow.AddSeconds(8)
    foreach($d in Get-ChildItem -LiteralPath $State -Directory -Force -ErrorAction Stop|Sort-Object LastWriteTime -Descending){
        if($d.Name -match '^(package-backup|cache-backup|stage|restore-stage)-[a-f0-9]{32}$'){
            Get-DcrRecoveryItem $State $CacheRoot $d.Name $deadline
        }
    }
}
function Invoke-DcrRecoveryRestore([string]$Root,[string]$State,[string]$CacheRoot,[string]$Name,[string]$Expected,[scriptblock]$Guard){
    $item=Get-DcrRecoveryItem $State $CacheRoot $Name ([datetime]::UtcNow.AddSeconds(20))
    if($item.status -ne 'ready' -or !$Expected -or $item.token -ne $Expected){throw 'STALE'}
    $source=Join-Path $State $Name
    & $Guard $source
    if($item.kind -eq 'cache'){
        $target=Get-DcrCacheRestoreTarget $State $CacheRoot $Name
        & $Guard $target
        $again=Get-DcrRecoveryItem $State $CacheRoot $Name ([datetime]::UtcNow.AddSeconds(20))
        if($again.status -ne 'ready' -or $again.token -ne $Expected){throw 'STALE'}
        Assert-DcrPlainPath $target
        [IO.Directory]::Move($source,$target)
        return
    }
    Assert-DcrPackageOwnership $Root
    if([IO.Path]::GetPathRoot($Root) -ne [IO.Path]::GetPathRoot($State)){throw 'UNSAFE'}
    $current=Get-DcrT6Version $Root
    $stage=Join-Path $State ('restore-stage-'+[guid]::NewGuid().ToString('N'))
    $previous=Join-Path $State ('package-backup-'+[guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $stage -ErrorAction Stop|Out-Null
    foreach($n in @('node_modules','package.json','package-lock.json')){
        Copy-Item -LiteralPath (Join-Path $source $n) -Destination $stage -Recurse -ErrorAction Stop
    }
    if((Get-DcrBackupVersion $stage) -ne $item.version){throw 'UNSAFE'}
    $again=Get-DcrRecoveryItem $State $CacheRoot $Name ([datetime]::UtcNow.AddSeconds(20))
    if($again.status -ne 'ready' -or $again.token -ne $Expected){throw 'STALE'}
    & $Guard $Root
    Assert-DcrPackageOwnership $Root
    if((Get-DcrT6Version $Root) -ne $current){throw 'STALE'}
    Switch-DcrTrackedPackage $Root $stage $previous $item.version
}
Export-ModuleMember -Function Get-DcrRecoveryInventory,Get-DcrRecoveryItem,Invoke-DcrRecoveryRestore
