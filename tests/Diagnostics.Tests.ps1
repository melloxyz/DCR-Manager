$ErrorActionPreference='Stop'
Import-Module (Join-Path $PSScriptRoot '..\src\Diagnostics.psm1')
$script:pass=0;$script:fail=0
function It($testName,[scriptblock]$body){try{&$body;$script:pass++;Write-Host "[PASS] $testName"}catch{$script:fail++;Write-Host "[FAIL] $testName : $_"}}
function Assert($x){if(!$x){throw 'Assertion failed'}}
$tmp=Join-Path ([IO.Path]::GetTempPath()) ('dcr-t8-'+[guid]::NewGuid().ToString('N'))
$dir=Join-Path $tmp '_logs'
New-Item -ItemType Directory -Path $dir -Force|Out-Null
try{
 It 'Log scan returns only recognized DCR codes and safe metadata' {
  Set-Content (Join-Path $dir '2026-11-24T11_43_31_950Z-debug-0.log') @('npm @wonderwhy-er/desktop-commander','error EBUSY EBUSY EPERM','secret token=fixture-private')
  Set-Content (Join-Path $dir '2026-11-24T11_44_31_950Z-debug-0.log') 'other package ENOTFOUND'
  $r=Get-DcrBoundedLogs $tmp
  Assert $r.known;Assert ($r.logs.Count -eq 1)
  Assert ($r.logs[0].codes.Count -eq 2)
  Assert (($r.logs[0].codes -join ',') -eq 'EBUSY,EPERM')
  Assert (($r|ConvertTo-Json -Depth 5) -notmatch 'fixture-private|token=|ENOTFOUND')
 }
 It 'Unknown cache is distinct from empty scan' {
  Assert (!(Get-DcrBoundedLogs '').known)
  $empty=Join-Path $tmp 'empty';New-Item -ItemType Directory -Path $empty|Out-Null
  $r=Get-DcrBoundedLogs $empty;Assert $r.known;Assert ($r.logs.Count -eq 0)
 }
 It 'Oversized log marks scan partial without reading contents' {
  $p=Join-Path $dir '2026-11-24T11_45_31_950Z-debug-0.log'
  $h=[IO.File]::Create($p);try{$h.SetLength(2097153)}finally{$h.Dispose()}
  $r=Get-DcrBoundedLogs $tmp;Assert (!$r.known);Assert ($r.logs.Count -eq 1)
 }
 It 'Untrusted filenames are not returned' {
  Set-Content (Join-Path $dir 'fixture-private-debug.log') '@wonderwhy-er/desktop-commander EACCES'
  $r=Get-DcrBoundedLogs $tmp
  Assert (($r|ConvertTo-Json -Depth 5) -notmatch 'fixture-private')
 }
}finally{Remove-Item -LiteralPath $tmp -Recurse -Force}
Write-Host "$script:pass passed / $script:fail failed"
if($script:fail){exit 1}
