$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot '..\src\Core.psm1') -Force
$script:passed=0; $script:failed=0
function It($name,[scriptblock]$body) {
    try { & $body; $script:passed++; Write-Host "[PASS] $name" -ForegroundColor Green }
    catch { $script:failed++; Write-Host "[FAIL] $name : $_" -ForegroundColor Red }
}
function Assert($condition) { if (!$condition) { throw 'Assertion failed' } }
function Proc($id,$parent,$line) { [pscustomobject]@{ProcessId=$id;ParentProcessId=$parent;CommandLine=$line;Name='node.exe'} }
$tmp=Join-Path ([IO.Path]::GetTempPath()) ('dcr-tests-'+[guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    $p=@(
        (Proc 1 0 '"C:\node.exe" "C:\a\node_modules\@wonderwhy-er\desktop-commander\dist\index.js" remote')
        (Proc 2 1 '"C:\node.exe" "C:\a\node_modules\@wonderwhy-er\desktop-commander\dist\index.js"')
        (Proc 3 0 '"C:\node.exe" "C:\b\node_modules\@wonderwhy-er\desktop-commander\dist\index.js"')
        (Proc 4 1 'node.exe C:\project\server.js')
        (Proc 5 0 'node.exe C:\project\remote.js')
    )
    It 'Remote tree excludes unrelated Node and local MCP' { Assert ((@(Select-DcrRemote $p).ProcessId -join ',') -eq '1,2') }
    It 'DCR identification excludes similarly named packages' { Assert (!(Test-DcrCommand 'node C:\x\node_modules\@wonderwhy-er\desktop-commander-fake\dist\index.js')) }
    It 'Recognizes npm shim entry' { Assert (Test-DcrCommand 'node C:\a\node_modules\.bin\desktop-commander remote') }
    It 'Repair pins version' { Assert ((Get-DcrPackageSpec '0.3.4') -eq '@wonderwhy-er/desktop-commander@0.3.4') }
    It 'Update selects latest' { Assert ((Get-DcrPackageSpec '0.3.4' -Update) -eq '@wonderwhy-er/desktop-commander@latest') }
    It 'Invalid version rejected for repair' { $caught=$false; try { Get-DcrPackageSpec 'garbage' | Out-Null } catch {$caught=$true}; Assert $caught }
    $npx=Join-Path $tmp '_npx'
    foreach($name in @('abc','def','other')) {
        $dir=Join-Path $npx $name
        New-Item -ItemType Directory -Path (Join-Path $dir 'node_modules\@wonderwhy-er\desktop-commander') -Force | Out-Null
        $json = if ($name -eq 'abc') {'{"dependencies":{"@wonderwhy-er/desktop-commander":"0.3.4"}}'} elseif ($name -eq 'def') {'{"dependencies":{"@wonderwhy-er/desktop-commander":"0.3.4","other":"1"}}'} else {'{"dependencies":{"other":"1"}}'}
        Set-Content -LiteralPath (Join-Path $dir 'package.json') -Value $json
    }
    It 'Cache discovery finds only exclusively owned DCR cache' { Assert ((@(Find-DcrCache $tmp).Name -join ',') -eq 'abc') }
    It 'Missing cache is safe' { Assert (@(Find-DcrCache (Join-Path $tmp 'missing')).Count -eq 0) }
    It 'Error summaries do not expose full log lines' { Assert ((@(Get-DcrErrorCodes @('secret EBUSY','token EACCES','EBUSY','private')).Count) -eq 2); Assert ((Get-DcrErrorCodes @('secret EBUSY')) -eq 'EBUSY') }
    $root=Join-Path $tmp 'install'; $stage=Join-Path $tmp 'stage'; $backup=Join-Path $tmp 'backup'
    foreach($d in @($root,$stage)){New-Item -ItemType Directory -Path (Join-Path $d 'node_modules') -Force | Out-Null}
    New-Item -ItemType Directory -Path (Join-Path $root 'Remote') | Out-Null
    Set-Content (Join-Path $root 'Remote\keep.txt') 'keep'
    foreach($d in @($root,$stage)){Set-Content (Join-Path $d 'package.json') $d;Set-Content (Join-Path $d 'package-lock.json') $d}
    It 'Package swap preserves credentials and backs up previous version' {
        Switch-DcrPackageFiles $root $stage $backup
        Assert ((Get-Content (Join-Path $root 'Remote\keep.txt')) -eq 'keep')
        Assert ((Get-Content (Join-Path $root 'package.json')) -eq $stage)
        Assert ((Get-Content (Join-Path $backup 'package.json')) -eq $root)
    }
    It 'Failed staging validation leaves installation intact' {
        $caught=$false; try { Switch-DcrPackageFiles $root (Join-Path $tmp 'missing') (Join-Path $tmp 'unused') } catch {$caught=$true}
        Assert $caught; Assert ((Get-Content (Join-Path $root 'package.json')) -eq $stage)
    }
} finally { Remove-Item -LiteralPath $tmp -Recurse -Force }
Write-Host "$script:passed passed / $script:failed failed"
if($script:failed){exit 1}
