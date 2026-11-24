Import-Module (Join-Path $PSScriptRoot 'Core.psm1') -Force
$script:pkg='@wonderwhy-er/desktop-commander'
function Assert-DcrPlainPath([string]$Path){
    if(![IO.Path]::IsPathRooted($Path)){throw 'UNSAFE'}
    $current=[IO.Path]::GetFullPath($Path)
    while($current){
        if(Test-Path -LiteralPath $current){
            if((Get-Item -LiteralPath $current -Force -ErrorAction Stop).Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'UNSAFE'}
        }
        $parent=[IO.Path]::GetDirectoryName($current)
        if(!$parent -or $parent -eq $current){break}
        $current=$parent
    }
}
function Get-DcrT6Version([string]$Root){
    $f=Join-Path $Root 'node_modules\@wonderwhy-er\desktop-commander\package.json'
    Assert-DcrPlainPath $f
    if(!(Test-Path -LiteralPath $f -PathType Leaf)){return 'absent'}
    try{
        $j=Get-Content -LiteralPath $f -Raw -ErrorAction Stop|ConvertFrom-Json -ErrorAction Stop
        if($j.name -ne $script:pkg -or $j.version -notmatch '^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$'){return 'unknown'}
        return [string]$j.version
    }catch{return 'unknown'}
}
function Assert-DcrPackageOwnership([string]$Root){
    Assert-DcrPlainPath $Root
    foreach($n in @('node_modules','package.json','package-lock.json')){Assert-DcrPlainPath (Join-Path $Root $n)}
    $f=Join-Path $Root 'package.json'
    if(!(Test-Path -LiteralPath $f)){
        if((Test-Path -LiteralPath (Join-Path $Root 'node_modules')) -or (Test-Path -LiteralPath (Join-Path $Root 'package-lock.json'))){throw 'UNSAFE'}
        return
    }
    try{$j=Get-Content -LiteralPath $f -Raw -ErrorAction Stop|ConvertFrom-Json -ErrorAction Stop}catch{throw 'UNSAFE'}
    $deps=@($j.dependencies.PSObject.Properties.Name|Where-Object {$_})
    if($deps.Count -ne 1 -or $deps[0] -ne $script:pkg -or $j.devDependencies -or $j.optionalDependencies -or $j.workspaces){throw 'UNSAFE'}
}
function Get-DcrT6Caches([string]$CacheRoot){
    if(!$CacheRoot){return}
    Assert-DcrPlainPath $CacheRoot
    $npx=Join-Path $CacheRoot '_npx'
    Assert-DcrPlainPath $npx
    foreach($d in Find-DcrCache $CacheRoot){
        if($d.Name -notmatch '^[a-f0-9]{8,64}$'){continue}
        try{
            Assert-DcrPlainPath $d.FullName
            Assert-DcrPlainPath (Join-Path $d.FullName 'package.json')
            Assert-DcrPlainPath (Join-Path $d.FullName 'node_modules\@wonderwhy-er\desktop-commander\package.json')
            Assert-DcrPackageOwnership $d.FullName
            $v=Get-DcrT6Version $d.FullName
            if($v -in @('absent','unknown')){continue}
            $hash=Get-FileHash -LiteralPath (Join-Path $d.FullName 'package.json') -Algorithm SHA256
            $identity=$d.FullName.ToLowerInvariant()+'|'+$d.CreationTimeUtc.Ticks+'|'+$hash.Hash+'|'+$v
            $sha=[Security.Cryptography.SHA256]::Create()
            try{$token=([BitConverter]::ToString($sha.ComputeHash([Text.Encoding]::UTF8.GetBytes($identity)))).Replace('-','').ToLowerInvariant()}finally{$sha.Dispose()}
            [pscustomobject]@{id=$d.Name;version=$v;token=$token;path=$d.FullName}
        }catch{continue}
    }
}
function Move-DcrT6Cache([string]$CacheRoot,[string]$State,[string]$Id,[string]$Expected,[scriptblock]$Guard){
    Assert-DcrPlainPath $State
    $c=@(Get-DcrT6Caches $CacheRoot|Where-Object {$_.id -eq $Id -and $_.token -eq $Expected})
    if($c.Count -ne 1){throw 'STALE'}
    & $Guard $c[0].path
    $again=@(Get-DcrT6Caches $CacheRoot|Where-Object {$_.id -eq $Id -and $_.token -eq $Expected})
    if($again.Count -ne 1){throw 'STALE'}
    if([IO.Path]::GetPathRoot($again[0].path) -ne [IO.Path]::GetPathRoot($State)){throw 'UNSAFE'}
    New-Item -ItemType Directory -Path $State -Force|Out-Null
    $dest=Join-Path $State ('cache-backup-'+[guid]::NewGuid().ToString('N'))
    [pscustomobject]@{originalPath=$again[0].path;backupPath=$dest}|ConvertTo-Json|Set-Content -LiteralPath ($dest+'.json') -Encoding UTF8 -ErrorAction Stop
    # Directory.Move is a same-volume rename, not a recursive copy/delete.
    [IO.Directory]::Move($again[0].path,$dest)
}
function Invoke-DcrT6Package([string]$Root,[string]$State,[string]$Action,[string]$Expected,[scriptblock]$Guard,[scriptblock]$StageRunner){
    Assert-DcrPlainPath $State
    Assert-DcrPackageOwnership $Root
    $current=Get-DcrT6Version $Root
    if($current -ne $Expected){throw 'STALE'}
    if($Action -notin @('repair','update')){throw 'UNSAFE'}
    $spec=Get-DcrPackageSpec $current -Update:($Action -eq 'update' -or $current -eq 'absent')
    & $Guard $Root
    if([IO.Path]::GetPathRoot($Root) -ne [IO.Path]::GetPathRoot($State)){throw 'UNSAFE'}
    $stage=Join-Path $State ('stage-'+[guid]::NewGuid().ToString('N'))
    $backup=Join-Path $State ('package-backup-'+[guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $stage -Force|Out-Null
    & $StageRunner $stage $spec
    Assert-DcrPackageOwnership $stage
    Assert-DcrPlainPath (Join-Path $stage 'node_modules\@wonderwhy-er\desktop-commander\dist\index.js')
    $v=Get-DcrT6Version $stage
    if($v -in @('absent','unknown') -or !(Test-Path -LiteralPath (Join-Path $stage 'node_modules\@wonderwhy-er\desktop-commander\dist\index.js') -PathType Leaf)){throw 'NPM'}
    if($Action -eq 'repair' -and $current -ne 'absent' -and $v -ne $current){throw 'NPM'}
    $top=Get-Content -LiteralPath (Join-Path $stage 'package.json') -Raw|ConvertFrom-Json
    if($top.dependencies.PSObject.Properties[$script:pkg].Value -ne $v){throw 'NPM'}
    & $Guard $Root
    Assert-DcrPackageOwnership $Root
    if((Get-DcrT6Version $Root) -ne $Expected){throw 'STALE'}
    Switch-DcrTrackedPackage $Root $stage $backup $v
}
function Get-DcrT6ErrorCode($Record){
    $msg=$Record.Exception.Message
    if($msg -match 'rollback incomplete|ROLLBACK'){return 'ROLLBACK'}
    if($msg -match '^(ACTIVE|STALE|LOCKED|UNSAFE|NPM|EBUSY|EPERM|EACCES)$'){return $msg}
    $e=$Record.Exception
    while($e){
        if(($e.HResult -band 0xffff) -in @(32,33)){return 'EBUSY'}
        if($e -is [UnauthorizedAccessException] -or ($e.HResult -band 0xffff) -eq 5){return 'EACCES'}
        $e=$e.InnerException
    }
    return 'FAILED'
}
function Switch-DcrTrackedPackage([string]$Root,[string]$Stage,[string]$Backup,[string]$Version,[scriptblock]$MoveAction={param($from,$to) Move-Item -LiteralPath $from -Destination $to -ErrorAction Stop}){
    $journal=Join-Path $Stage 'transaction.json'
    function Write-State([string]$Status){
        $tmp=$journal+'.tmp'
        [pscustomobject]@{schema=1;state=$Status;backup=[IO.Path]::GetFileName($Backup);version=$Version}|ConvertTo-Json -Compress|Set-Content -LiteralPath $tmp -Encoding UTF8 -ErrorAction Stop
        if([IO.File]::Exists($journal)){[IO.File]::Replace($tmp,$journal,($journal+'.previous'))}else{[IO.File]::Move($tmp,$journal)}
    }
    Write-State 'switching'
    try{
        Switch-DcrPackageFiles $Root $Stage $Backup -MoveAction $MoveAction
        if((Get-DcrT6Version $Root) -ne $Version){throw 'ROLLBACK'}
    }catch{
        $original=$_
        $status=if((Get-DcrT6ErrorCode $original) -eq 'ROLLBACK'){'rollback-incomplete'}else{'switch-failed'}
        try{Write-State $status}catch{}
        throw $original
    }
    Write-State 'completed'
}
Export-ModuleMember -Function Get-DcrT6Caches,Move-DcrT6Cache,Invoke-DcrT6Package,Get-DcrT6Version,Get-DcrT6ErrorCode,Assert-DcrPlainPath,Assert-DcrPackageOwnership,Switch-DcrTrackedPackage
