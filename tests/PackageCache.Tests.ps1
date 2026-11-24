$ErrorActionPreference='Stop'
Import-Module (Join-Path $PSScriptRoot '..\src\PackageCache.psm1') -Force
$script:passed=0;$script:failed=0
function It($name,[scriptblock]$body){try{&$body;$script:passed++;Write-Host "[PASS] $name"}catch{$script:failed++;Write-Host "[FAIL] $name : $_"}}
function Assert($condition){if(!$condition){throw 'Assertion failed'}}
function Throws([scriptblock]$body,[string]$code){
    $caught=$false
    try{&$body}catch{$caught=$true;if($code -and $_.Exception.Message -notmatch $code){throw $_}}
    Assert $caught
}
function Package($root,$version){
    $p=Join-Path $root 'node_modules\@wonderwhy-er\desktop-commander'
    New-Item -ItemType Directory -Path (Join-Path $p 'dist') -Force|Out-Null
    Set-Content -LiteralPath (Join-Path $p 'dist\index.js') '// inert fixture'
    [pscustomobject]@{name='@wonderwhy-er/desktop-commander';version=$version}|ConvertTo-Json|Set-Content (Join-Path $p 'package.json')
    [pscustomobject]@{dependencies=@{'@wonderwhy-er/desktop-commander'=$version}}|ConvertTo-Json|Set-Content (Join-Path $root 'package.json')
    Set-Content (Join-Path $root 'package-lock.json') '{}'
}
$tmp=Join-Path ([IO.Path]::GetTempPath()) ('dcr-t6-'+[guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp|Out-Null
try{
    $root=Join-Path $tmp 'install';$state=Join-Path $tmp 'state'
    Package $root '0.3.4'
    New-Item -ItemType Directory -Path (Join-Path $root 'Remote')|Out-Null
    Set-Content (Join-Path $root 'Remote\keep') 'credential fixture'
    $guard={param($path)}
    It 'Repair stages exact version and preserves credentials and backup' {
        Invoke-DcrT6Package $root $state 'repair' '0.3.4' $guard {param($stage,$spec) Assert ($spec -eq '@wonderwhy-er/desktop-commander@0.3.4');Package $stage '0.3.4'}
        Assert ((Get-DcrT6Version $root) -eq '0.3.4')
        Assert ((Get-Content (Join-Path $root 'Remote\keep')) -eq 'credential fixture')
        Assert (@(Get-ChildItem $state -Directory -Filter 'package-backup-*').Count -eq 1)
    }
    It 'Update stages latest and keeps previous version' {
        Invoke-DcrT6Package $root $state 'update' '0.3.4' $guard {param($stage,$spec) Assert ($spec -eq '@wonderwhy-er/desktop-commander@latest');Package $stage '0.3.5'}
        Assert ((Get-DcrT6Version $root) -eq '0.3.5')
    }
    It 'Fresh install works without existing directory' {
        $fresh=Join-Path $tmp 'fresh'
        Invoke-DcrT6Package $fresh $state 'repair' 'absent' $guard {param($stage,$spec) Assert ($spec -eq '@wonderwhy-er/desktop-commander@latest');Package $stage '0.3.5'}
        Assert ((Get-DcrT6Version $fresh) -eq '0.3.5')
    }
    It 'Stage failure keeps original installation' {
        Throws {Invoke-DcrT6Package $root $state 'repair' '0.3.5' $guard {throw 'EBUSY'}} 'EBUSY'
        Assert ((Get-DcrT6Version $root) -eq '0.3.5')
    }
    It 'Wrong staged repair version is rejected' {
        Throws {Invoke-DcrT6Package $root $state 'repair' '0.3.5' $guard {param($stage,$spec) Package $stage '0.3.6'}} 'NPM'
        Assert ((Get-DcrT6Version $root) -eq '0.3.5')
    }
    It 'Stale version blocks staging' {
        Throws {Invoke-DcrT6Package $root $state 'repair' '0.3.4' $guard {throw 'must not run'}} 'STALE'
    }
    It 'Active process guard refuses before staging' {
        Throws {Invoke-DcrT6Package $root $state 'update' '0.3.5' {throw 'ACTIVE'} {throw 'must not run'}} 'ACTIVE'
    }
    It 'Mixed package dependencies refuse replacement' {
        $mixed=Join-Path $tmp 'mixed';Package $mixed '0.3.5'
        Set-Content (Join-Path $mixed 'package.json') '{"dependencies":{"@wonderwhy-er/desktop-commander":"0.3.5","other":"1"}}'
        Throws {Invoke-DcrT6Package $mixed $state 'update' '0.3.5' $guard {throw 'must not run'}} 'UNSAFE'
    }
    It 'Real Windows sharing violation is reported as EBUSY' {
        $lockFile=Join-Path $tmp 'locked.txt'
        Set-Content -LiteralPath $lockFile 'fixture'
        $h=[IO.File]::Open($lockFile,[IO.FileMode]::Open,[IO.FileAccess]::ReadWrite,[IO.FileShare]::None)
        try{
            $code=''
            try{[IO.File]::Open($lockFile,[IO.FileMode]::Open,[IO.FileAccess]::Read).Dispose()}catch{$code=Get-DcrT6ErrorCode $_}
            Assert ($code -eq 'EBUSY')
        }finally{$h.Dispose()}
    }
    It 'Late active-process guard blocks package switch' {
        $script:checks=0
        Throws {Invoke-DcrT6Package $root $state 'update' '0.3.5' {param($path) $script:checks++;if($script:checks -gt 1){throw 'ACTIVE'}} {param($stage,$spec) Package $stage '0.3.6'}} 'ACTIVE'
        Assert ((Get-DcrT6Version $root) -eq '0.3.5')
    }
    $cache=Join-Path $tmp 'cache'
    $one=Join-Path $cache '_npx\0123456789abcdef';$two=Join-Path $cache '_npx\abcdef0123456789'
    Package $one '0.3.3';Package $two '0.3.4'
    $mixed=Join-Path $cache '_npx\1234567890abcdef';Package $mixed '0.3.4'
    Set-Content (Join-Path $mixed 'package.json') '{"dependencies":{"@wonderwhy-er/desktop-commander":"0.3.4","other":"1"}}'
    It 'Cache discovery excludes mixed cache' {
        $r=@(Get-DcrT6Caches $cache)
        Assert ($r.Count -eq 2);Assert (@($r|Where-Object {$_.token -notmatch '^[a-f0-9]{64}$'}).Count -eq 0)
    }
    It 'Stale cache token prevents quarantine' {
        $c=@(Get-DcrT6Caches $cache|Where-Object id -eq '0123456789abcdef')[0]
        Set-Content (Join-Path $one 'package.json') '{"dependencies":{"@wonderwhy-er/desktop-commander":"latest"}}'
        Throws {Move-DcrT6Cache $cache $state $c.id $c.token $guard} 'STALE'
        Assert (Test-Path $one)
    }
    It 'Active cache guard refuses quarantine' {
        $c=@(Get-DcrT6Caches $cache|Where-Object id -eq '0123456789abcdef')[0]
        Throws {Move-DcrT6Cache $cache $state $c.id $c.token {throw 'ACTIVE'}} 'ACTIVE'
        Assert (Test-Path $one)
    }
    It 'Quarantine moves only selected cache and records restore metadata' {
        $c=@(Get-DcrT6Caches $cache|Where-Object id -eq '0123456789abcdef')[0]
        Move-DcrT6Cache $cache $state $c.id $c.token $guard
        Assert (!(Test-Path $one));Assert (Test-Path $two);Assert (Test-Path $mixed)
        $backup=@(Get-ChildItem $state -Directory -Filter 'cache-backup-*')[0]
        $meta=Get-Content ($backup.FullName+'.json') -Raw|ConvertFrom-Json
        Assert ($meta.originalPath -eq $one);Assert ($meta.backupPath -eq $backup.FullName)
    }
}finally{Remove-Item -LiteralPath $tmp -Recurse -Force}
Write-Host "$script:passed passed / $script:failed failed"
if($script:failed){exit 1}
