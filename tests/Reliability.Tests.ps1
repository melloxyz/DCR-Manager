$ErrorActionPreference='Stop'
Import-Module (Join-Path $PSScriptRoot '..\src\Core.psm1') -Force
Import-Module (Join-Path $PSScriptRoot '..\src\Recovery.psm1') -Force
$script:pass=0;$script:fail=0
function It($name,[scriptblock]$body){try{&$body;$script:pass++;Write-Host "[PASS] $name"}catch{$script:fail++;Write-Host "[FAIL] $name : $_"}}
function Assert($condition){if(!$condition){throw 'Assertion failed'}}
$tmp=Join-Path ([IO.Path]::GetTempPath()) ('dcr-reliability-'+[guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp | Out-Null
function Fixture($tag){
    $r=Join-Path $tmp "$tag-root";$s=Join-Path $tmp "$tag-stage";$b=Join-Path $tmp "$tag-backup"
    foreach($d in @($r,$s)){New-Item -ItemType Directory -Path (Join-Path $d 'node_modules') -Force | Out-Null;Set-Content (Join-Path $d 'package.json') $d;Set-Content (Join-Path $d 'package-lock.json') $d}
    New-Item -ItemType Directory -Path (Join-Path $r 'Remote') | Out-Null
    Set-Content (Join-Path $r 'Remote\keep') 'credential fixture'
    @{root=$r;stage=$s;backup=$b}
}
try{
    It 'Injected EBUSY during switch rolls all earlier moves back' {
        $f=Fixture 'busy';$caught=$false
        $move={param($from,$to) if($from -eq (Join-Path $f.stage 'package.json')){throw 'EBUSY simulated'};Move-Item -LiteralPath $from -Destination $to -ErrorAction Stop}.GetNewClosure()
        try{Switch-DcrPackageFiles $f.root $f.stage $f.backup -MoveAction $move}catch{$caught=$true;Assert ($_.Exception.Message -match 'EBUSY')}
        Assert $caught;Assert ((Get-Content (Join-Path $f.root 'package.json')) -eq $f.root)
        Assert ((Get-Content (Join-Path $f.stage 'package.json')) -eq $f.stage)
        Assert (Test-Path (Join-Path $f.root 'node_modules'));Assert (Test-Path (Join-Path $f.stage 'node_modules'))
        Assert ((Get-Content (Join-Path $f.root 'Remote\keep')) -eq 'credential fixture')
    }
    It 'Rollback attempts remaining items and reports incomplete recovery' {
        $f=Fixture 'rollback';$caught=$false
        $move={param($from,$to) if($from -eq (Join-Path $f.stage 'package.json') -or ($from -eq (Join-Path $f.root 'node_modules') -and $to -eq (Join-Path $f.stage 'node_modules'))){throw 'EBUSY simulated'};Move-Item -LiteralPath $from -Destination $to -ErrorAction Stop}.GetNewClosure()
        try{Switch-DcrPackageFiles $f.root $f.stage $f.backup -MoveAction $move}catch{$caught=$true;Assert ($_.Exception.Message -match 'rollback incomplete')}
        Assert $caught;Assert ((Get-Content (Join-Path $f.root 'package.json')) -eq $f.root)
        Assert (Test-Path (Join-Path $f.backup 'node_modules'))
    }
    It 'Real Windows locked file fails safely and preserves old package' {
        $f=Fixture 'lock';$lock=[IO.File]::Open((Join-Path $f.root 'package.json'),'Open','Read','None')
        $caught=$false
        try{try{Switch-DcrPackageFiles $f.root $f.stage $f.backup}catch{$caught=$true};Assert $caught}finally{$lock.Dispose()}
        Assert ((Get-Content (Join-Path $f.root 'package.json')) -eq $f.root)
        Assert (Test-Path (Join-Path $f.root 'node_modules'))
    }
    $state=Join-Path $tmp 'state';$cache=Join-Path $tmp 'cache';$backupName='cache-backup-0123456789abcdef0123456789abcdef'
    $source=Join-Path $state $backupName;$target=Join-Path $cache '_npx\4b4c857f6efdfb61'
    New-Item -ItemType Directory -Path (Join-Path $source 'node_modules\@wonderwhy-er\desktop-commander'),(Join-Path $cache '_npx') -Force | Out-Null
    Set-Content (Join-Path $source 'package.json') '{"dependencies":{"@wonderwhy-er/desktop-commander":"0.3.3"}}'
    function Meta($path){[pscustomobject]@{originalPath=$path;backupPath=$source}|ConvertTo-Json|Set-Content ($source+'.json')}
    Meta $target
    It 'Cache restore rejects destinations outside configured npm cache' {
        Meta (Join-Path $tmp '4b4c857f6efdfb61')
        $caught=$false;try{Get-DcrCacheRestoreTarget $state $cache $backupName}catch{$caught=$true}
        Assert $caught;Meta $target
    }
    It 'Cache restore does not overwrite existing cache' {
        New-Item -ItemType Directory -Path $target | Out-Null
        $caught=$false;try{Restore-DcrCacheBackup $state $cache $backupName}catch{$caught=$true}
        Assert $caught;Assert (Test-Path $source);Remove-Item -LiteralPath $target
    }
    It 'Cache restore moves exclusive backup to original target' {
        Assert ((Restore-DcrCacheBackup $state $cache $backupName) -eq $target)
        Assert (Test-Path $target);Assert (!(Test-Path $source))
    }
    It 'Process assessment identifies missing and reused parents' {
        $now=Get-Date
        $a=[pscustomobject]@{ProcessId=1;ParentProcessId=9;CreationDate=$now}
        $b=[pscustomobject]@{ProcessId=2;ParentProcessId=3;CreationDate=$now}
        $c=[pscustomobject]@{ProcessId=3;ParentProcessId=0;CreationDate=$now.AddSeconds(1)}
        $r=@(Get-DcrParentAssessment @($a,$b,$c) @($a,$b))
        Assert ($r[0].assessment -eq 'parent-missing');Assert ($r[1].assessment -eq 'parent-pid-reused')
    }
    It 'Retention only recommends review; does not delete backups' {
        $now=Get-Date;$b=[pscustomobject]@{Name='old';LastWriteTime=$now.AddDays(-31)}
        Assert ((Get-DcrRetentionAssessment @($b) $now).review)
        Assert (Test-Path $target)
    }
    $tomlSupported=$false; if(Get-Command py.exe -ErrorAction SilentlyContinue){& py.exe -3 -c "import tomllib" 2>$null; $tomlSupported=($LASTEXITCODE -eq 0)}
    if($tomlSupported){
        It 'TOML audit detects DCR without exposing secrets' {
            Import-Module (Join-Path $PSScriptRoot '..\src\Maintenance.psm1') -Force
            $toml=Join-Path $tmp 'client.toml'
            @"
[mcp_servers.private_name]
command = "npx"
args = ["-y", "@wonderwhy-er/desktop-commander@latest"]
[mcp_servers.private_name.env]
TOKEN = "do-not-export"
[mcp_servers.other]
command = "node"
args = ["other"]
"@ | Set-Content -LiteralPath $toml
            $result=Get-DcrCustomAudit $toml
            Assert ($result.status -eq 'checked');Assert ($result.entries -eq 1);Assert ($result.modes -eq 'npx')
            Assert (($result | ConvertTo-Json) -notmatch 'private_name|TOKEN|do-not-export')
        }
        It 'TOML audit counts empty config as zero and multiple entries correctly' {
            $toml=Join-Path $tmp 'counts.toml'
            Set-Content -LiteralPath $toml '[mcp_servers.other]'
            Assert ((Get-DcrCustomAudit $toml).entries -eq 0)
            @"
[mcp_servers.a]
command = "npx"
args = ["@wonderwhy-er/desktop-commander@latest"]
[mcp_servers.b]
command = "npx"
args = ["@wonderwhy-er/desktop-commander@0.3.4"]
"@ | Set-Content -LiteralPath $toml
            $result=Get-DcrCustomAudit $toml
            Assert ($result.entries -eq 2);Assert ($result.floating -eq 1)
        }
    }else{Write-Host '[SKIP] Optional Python TOML fixture'}
}finally{Remove-Item -LiteralPath $tmp -Recurse -Force}
Write-Host "$script:pass passed / $script:fail failed"
if($script:fail){exit 1}
