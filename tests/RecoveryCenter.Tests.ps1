$ErrorActionPreference='Stop'
Import-Module (Join-Path $PSScriptRoot '..\src\PackageCache.psm1')
Import-Module (Join-Path $PSScriptRoot '..\src\RecoveryCenter.psm1')
$script:pass=0;$script:fail=0
function It($testName,[scriptblock]$body){try{&$body;$script:pass++;Write-Host "[PASS] $testName"}catch{$script:fail++;Write-Host "[FAIL] $testName : $_"}}
function Assert($x){if(!$x){throw 'Assertion failed'}}
function Refused([scriptblock]$body,[string]$code){$caught=$false;try{&$body}catch{$caught=$true;if($_.Exception.Message -notmatch $code){throw $_}};Assert $caught}
function Package($root,$version){
 $pkg=Join-Path $root 'node_modules\@wonderwhy-er\desktop-commander'
 New-Item -ItemType Directory -Path (Join-Path $pkg 'dist') -Force|Out-Null
 Set-Content (Join-Path $pkg 'dist\index.js') '// inert fixture'
 [pscustomobject]@{name='@wonderwhy-er/desktop-commander';version=$version}|ConvertTo-Json|Set-Content (Join-Path $pkg 'package.json')
 [pscustomobject]@{dependencies=@{'@wonderwhy-er/desktop-commander'=$version}}|ConvertTo-Json|Set-Content (Join-Path $root 'package.json')
 Set-Content (Join-Path $root 'package-lock.json') '{}'
}
$tmp=Join-Path ([IO.Path]::GetTempPath()) ('dcr-t7-'+[guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp|Out-Null
try{
 $root=Join-Path $tmp 'root';$state=Join-Path $tmp 'state';$cache=Join-Path $tmp 'cache'
 $name='package-backup-0123456789abcdef0123456789abcdef';$backup=Join-Path $state $name
 Package $root '0.3.4';Package $backup '0.3.3'
 New-Item -ItemType Directory -Path (Join-Path $root 'Remote'),(Join-Path $cache '_npx')|Out-Null
 Set-Content (Join-Path $root 'Remote\keep') 'fixture'
 $guard={param($path)}
 function Item($n){Get-DcrRecoveryItem $state $cache $n ([datetime]::UtcNow.AddSeconds(20))}
 It 'Backup validation reports version, logical size and retention' {
  (Get-Item $backup).LastWriteTime=(Get-Date).AddDays(-35)
  $r=Item $name;Assert ($r.status -eq 'ready');Assert ($r.version -eq '0.3.3');Assert ($r.sizeKnown -and $r.bytes -gt 0);Assert $r.review;Assert ($r.token -match '^[a-f0-9]{64}$')
 }
 It 'Stale backup token refused' {
  Refused {Invoke-DcrRecoveryRestore $root $state $cache $name ('0'*64) $guard} 'STALE'
  Assert ((Get-DcrT6Version $root) -eq '0.3.4')
 }
 It 'Active guard refuses package restore' {
  $r=Item $name
  Refused {Invoke-DcrRecoveryRestore $root $state $cache $name $r.token {throw 'ACTIVE'}} 'ACTIVE'
 }
 It 'Restore preserves source, previous package and Remote credentials' {
  $r=Item $name;Invoke-DcrRecoveryRestore $root $state $cache $name $r.token $guard
  Assert ((Get-DcrT6Version $root) -eq '0.3.3');Assert ((Get-DcrT6Version $backup) -eq '0.3.3')
  Assert ((Get-Content (Join-Path $root 'Remote\keep')) -eq 'fixture')
  $rows=@(Get-DcrRecoveryInventory $state $cache)
  Assert (@($rows|Where-Object {$_.kind -eq 'package' -and $_.version -eq '0.3.4' -and $_.status -eq 'ready'}).Count -eq 1)
  Assert (@($rows|Where-Object {$_.kind -eq 'stage' -and $_.status -eq 'completed'}).Count -eq 1)
 }
 It 'Incomplete and interrupted stages are not restorable' {
  $n='stage-abcdef0123456789abcdef0123456789';$d=Join-Path $state $n
  New-Item -ItemType Directory -Path $d|Out-Null
  Set-Content (Join-Path $d 'transaction.json') '{"schema":1,"state":"switching"}'
  $r=Item $n;Assert ($r.status -eq 'switching');Assert (!$r.token)
  Refused {Invoke-DcrRecoveryRestore $root $state $cache $n ('0'*64) $guard} 'STALE'
 }
 $cn='cache-backup-abcdef0123456789abcdef0123456789';$cb=Join-Path $state $cn;$target=Join-Path $cache '_npx\abcdef0123456789'
 Package $cb '0.3.3'
 [pscustomobject]@{originalPath=$target;backupPath=$cb}|ConvertTo-Json|Set-Content ($cb+'.json')
 It 'Cache restore refuses occupied destination' {
  New-Item -ItemType Directory -Path $target|Out-Null
  $r=Item $cn;Assert ($r.status -eq 'destination-blocked')
  Refused {Invoke-DcrRecoveryRestore $root $state $cache $cn ('0'*64) $guard} 'STALE'
  Assert (Test-Path $cb);Remove-Item -LiteralPath $target
 }
 It 'Cache restore uses same-volume move and preserves metadata' {
  $r=Item $cn;Assert ($r.status -eq 'ready')
  Invoke-DcrRecoveryRestore $root $state $cache $cn $r.token $guard
  Assert (Test-Path $target);Assert (!(Test-Path $cb));Assert (Test-Path ($cb+'.json'))
 }
 It 'Tracked rollback failure records incomplete state and preserves recovery artifacts' {
  $r=Join-Path $tmp 'fault-root';$st=Join-Path $state ('stage-'+('f'*32));$bk=Join-Path $state ('package-backup-'+('f'*32))
  Package $r '0.3.4';Package $st '0.3.5'
  $move={param($from,$to)
   if($from -eq (Join-Path $st 'package.json') -or ($from -eq (Join-Path $r 'node_modules') -and $to -eq (Join-Path $st 'node_modules'))){throw 'EBUSY simulated'}
   Move-Item -LiteralPath $from -Destination $to -ErrorAction Stop
  }.GetNewClosure()
  Refused {Switch-DcrTrackedPackage $r $st $bk '0.3.5' $move} 'rollback incomplete'
  Assert ((Item ([IO.Path]::GetFileName($st))).status -eq 'rollback-incomplete')
  Assert (Test-Path (Join-Path $bk 'node_modules'))
 }
 It 'Backup manifest change invalidates its previously observed token' {
  $r=Item $name
  Add-Content -LiteralPath (Join-Path $backup 'package.json') ' '
  Refused {Invoke-DcrRecoveryRestore $root $state $cache $name $r.token $guard} 'STALE'
 }
 It 'Path traversal and mixed backup refused' {
  Refused {Invoke-DcrRecoveryRestore $root $state $cache '..\other' ('0'*64) $guard} 'UNSAFE'
  Set-Content (Join-Path $backup 'package.json') '{"dependencies":{"@wonderwhy-er/desktop-commander":"0.3.3","other":"1"}}'
  Assert ((Item $name).status -eq 'unsafe')
 }
}finally{Remove-Item -LiteralPath $tmp -Recurse -Force}
Write-Host "$script:pass passed / $script:fail failed"
if($script:fail){exit 1}
