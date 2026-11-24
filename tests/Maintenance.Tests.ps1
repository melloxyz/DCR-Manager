$ErrorActionPreference='Stop'
Import-Module (Join-Path $PSScriptRoot '..\src\Maintenance.psm1') -Force
$script:passed=0;$script:failed=0
function It($name,[scriptblock]$body){try{&$body;$script:passed++;Write-Host "[PASS] $name"}catch{$script:failed++;Write-Host "[FAIL] $name : $_"}}
function Assert($condition){if(!$condition){throw 'Assertion failed'}}
$tmp=Join-Path ([IO.Path]::GetTempPath()) ('dcr-maint-'+[guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    It 'Recognizes exact package argument, not similar names' {
        Assert (Test-DcrArgument '@wonderwhy-er/desktop-commander@latest')
        Assert (!(Test-DcrArgument '@wonderwhy-er/desktop-commander-fake'))
        Assert (Test-DcrArgument 'C:\test\node_modules\@wonderwhy-er\desktop-commander\dist\index.js')
    }
    It 'JSON audit excludes secret values and unrelated MCPs' {
        $j='{"mcpServers":{"a":{"command":"npx","args":["-y","@wonderwhy-er/desktop-commander@latest"],"env":{"TOKEN":"secret"}},"b":{"command":"npx","args":["other"]}}}'
        $r=@(Get-DcrJsonConfigSummary $j)
        Assert ($r.Count -eq 1);Assert ($r[0].mode -eq 'npx');Assert (($r | ConvertTo-Json) -notmatch 'secret|TOKEN')
    }
    It 'Recognizes direct entry and avoids reporting server names' {
        $j='{"servers":{"private-name":{"command":"node","args":["C:\\\\x\\\\node_modules\\\\@wonderwhy-er\\\\desktop-commander\\\\dist\\\\index.js"]}}}'
        $r=@(Get-DcrJsonConfigSummary $j)
        Assert ($r.Count -eq 1);Assert ($r[0].mode -eq 'direct');Assert (($r | ConvertTo-Json) -notmatch 'private-name')
    }
    It 'Malformed JSON rejected without leaking input' {
        $caught=$false;try{Get-DcrJsonConfigSummary '{"secret":' | Out-Null}catch{$caught=$true;Assert ($_.Exception.Message -notmatch 'secret')}
        Assert $caught
    }
    It 'Version compatibility distinguishes unsupported and unknown' {
        Assert ((Get-DcrNodeAssessment 'v24.15.0' '>=18') -eq 'meets-minimum')
        Assert ((Get-DcrNodeAssessment 'v16.0.0' '>=18') -eq 'below-minimum')
        Assert ((Get-DcrNodeAssessment 'v24.0.0' '^18 || ^20') -eq 'unknown')
    }
    $backup=Join-Path $tmp 'package-backup-test'
    $pkg=Join-Path $backup 'node_modules\@wonderwhy-er\desktop-commander'
    New-Item -ItemType Directory -Path (Join-Path $pkg 'dist') -Force | Out-Null
    Set-Content (Join-Path $pkg 'package.json') '{"name":"@wonderwhy-er/desktop-commander","version":"0.3.4"}'
    Set-Content (Join-Path $pkg 'dist\index.js') '// fixture'
    Set-Content (Join-Path $backup 'package.json') '{"dependencies":{"@wonderwhy-er/desktop-commander":"0.3.4"}}'
    Set-Content (Join-Path $backup 'package-lock.json') '{}'
    It 'Complete package backup recognized' { Assert ((Get-DcrBackupVersion $backup) -eq '0.3.4') }
    It 'Incomplete or wrong package rejected' {
        Remove-Item (Join-Path $pkg 'dist\index.js');Assert (!(Get-DcrBackupVersion $backup))
        Set-Content (Join-Path $pkg 'dist\index.js') '// fixture'
        Set-Content (Join-Path $pkg 'package.json') '{"name":"other","version":"1.0.0"}';Assert (!(Get-DcrBackupVersion $backup))
        Set-Content (Join-Path $pkg 'package.json') '{"name":"@wonderwhy-er/desktop-commander","version":"0.3.4"}'
    }
    It 'Restore preserves backup and Remote credentials' {
        $root=Join-Path $tmp 'install';$state=Join-Path $tmp 'state'
        New-Item -ItemType Directory -Path (Join-Path $root 'Remote'),$state -Force | Out-Null
        Set-Content (Join-Path $root 'Remote\keep.txt') 'keep'
        Restore-DcrBackupFiles $root $backup $state | Out-Null
        Assert ((Get-DcrBackupVersion $root) -eq '0.3.4')
        Assert ((Get-DcrBackupVersion $backup) -eq '0.3.4')
        Assert ((Get-Content (Join-Path $root 'Remote\keep.txt')) -eq 'keep')
    }
}finally{Remove-Item -LiteralPath $tmp -Recurse -Force}
Write-Host "$script:passed passed / $script:failed failed"
if($script:failed){exit 1}
