Import-Module (Join-Path $PSScriptRoot 'Core.psm1') -Force
function Test-DcrArgument([string]$Value){
    return ($Value -match '^@wonderwhy-er/desktop-commander(?:@[^@\s]+)?$' -or $Value -match '(?i)[\\/]+@wonderwhy-er[\\/]+desktop-commander[\\/]+')
}
function Get-DcrJsonConfigSummary([string]$Json){
    try{$cfg=ConvertFrom-Json -InputObject $Json -ErrorAction Stop}catch{throw 'Invalid or unsupported JSON configuration.'}
    foreach($key in @('mcpServers','servers')){
        $servers=$cfg.$key
        if(!$servers){continue}
        foreach($item in $servers.PSObject.Properties){
            $v=$item.Value
            if(!$v){continue}
            $argsList=@($v.args | Where-Object {$_ -is [string]})
            if(!(@($argsList | Where-Object {Test-DcrArgument $_}).Count -or (Test-DcrArgument $v.command))){continue}
            $mode=if($v.command -match '(?i)(^|[\\/])npx(?:\.cmd|\.exe)?$'){'npx'}elseif($v.command -match '(?i)(^|[\\/])node(?:\.exe)?$'){'direct'}else{'other'}
            [pscustomobject]@{mode=$mode;floatingVersion=[bool](@($argsList | Where-Object {$_ -match '^@wonderwhy-er/desktop-commander(?:@latest)?$'}).Count)}
        }
    }
}
function Get-DcrNodeAssessment([string]$Version,[string]$Range){
    if($Version -notmatch '^v?(\d+)\.(\d+)\.(\d+)$'){return 'unknown'}
    $actual=[version]($Version.TrimStart('v'))
    if($Range -notmatch '^>=\s*(\d+)(?:\.(\d+))?(?:\.(\d+))?$'){return 'unknown'}
    $min=[version]::new([int]$Matches[1],[int]$Matches[2],[int]$Matches[3])
    if($actual -lt $min){'below-minimum'}else{'meets-minimum'}
}
function Get-DcrBackupVersion([string]$Path){
    try {
        $pkg=Join-Path $Path 'node_modules\@wonderwhy-er\desktop-commander'
        foreach($f in @('package.json','package-lock.json')){if(!(Test-Path -LiteralPath (Join-Path $Path $f) -PathType Leaf)){return}}
        if(!(Test-Path -LiteralPath (Join-Path $pkg 'dist\index.js') -PathType Leaf)){return}
        $j=Get-Content -LiteralPath (Join-Path $pkg 'package.json') -Raw | ConvertFrom-Json -ErrorAction Stop
        $top=Get-Content -LiteralPath (Join-Path $Path 'package.json') -Raw | ConvertFrom-Json -ErrorAction Stop
        if($j.name -eq '@wonderwhy-er/desktop-commander' -and $j.version -match '^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$' -and $top.dependencies.'@wonderwhy-er/desktop-commander' -eq $j.version){$j.version}
    }catch{return}
}
function Restore-DcrBackupFiles([string]$Root,[string]$Backup,[string]$State,[scriptblock]$BeforeSwitch={}){
    if(!(Get-DcrBackupVersion $Backup)){throw 'Invalid or incomplete package backup.'}
    $stage=Join-Path $State ('restore-stage-'+[guid]::NewGuid().ToString('N'))
    $previous=Join-Path $State ('package-backup-'+[guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $stage -Force | Out-Null
    foreach($name in @('node_modules','package.json','package-lock.json')){
        $src=Join-Path $Backup $name
        if((Get-Item -LiteralPath $src).Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'Backup contains a reparse point.'}
        if(@(Get-ChildItem -LiteralPath $src -Recurse -Force -ErrorAction Stop | Where-Object {$_.Attributes -band [IO.FileAttributes]::ReparsePoint}).Count){throw 'Backup contains reparse points; manual recovery required.'}
        Copy-Item -LiteralPath $src -Destination $stage -Recurse -ErrorAction Stop
    }
    if(!(Get-DcrBackupVersion $stage)){throw 'Invalid restored stage.'}
    & $BeforeSwitch
    Switch-DcrPackageFiles $Root $stage $previous
    $previous
}
function Get-DcrInstallInventory {
    $roots=@([pscustomobject]@{kind='dedicated';path=(Join-Path $env:LOCALAPPDATA 'DesktopCommander\node_modules')})
    if(Get-Command npm.cmd -ErrorAction SilentlyContinue){
        $out=@(& npm.cmd root -g 2>$null)
        if($LASTEXITCODE -eq 0 -and $out.Count -eq 1 -and [IO.Path]::IsPathRooted($out[0])){$roots+=[pscustomobject]@{kind='global';path=$out[0]}}
        $out=@(& npm.cmd config get cache 2>$null)
        if($LASTEXITCODE -eq 0 -and $out.Count -eq 1 -and [IO.Path]::IsPathRooted($out[0])){
            foreach($dir in Find-DcrCache $out[0]){$roots+=[pscustomobject]@{kind='npx';path=(Join-Path $dir.FullName 'node_modules')}}
        }
    }
    foreach($r in $roots){
        $pkg=Join-Path $r.path '@wonderwhy-er\desktop-commander'
        $manifest=Join-Path $pkg 'package.json'
        if(!(Test-Path -LiteralPath $manifest)){continue}
        try{$j=Get-Content -LiteralPath $manifest -Raw | ConvertFrom-Json -ErrorAction Stop
            if($j.name -ne '@wonderwhy-er/desktop-commander'){continue}
            [pscustomobject]@{kind=$r.kind;version=$j.version;entryExists=(Test-Path -LiteralPath (Join-Path $pkg 'dist\index.js'));nodeRange=$j.engines.node}
        }catch{[pscustomobject]@{kind=$r.kind;version='unreadable';entryExists=$false;nodeRange=$null}}
    }
}
function Get-DcrCustomAudit([string]$Path){
    $extension=[IO.Path]::GetExtension($Path)
    $result=@()
    $status='checked'
    try{
        $item=Get-Item -LiteralPath $Path -ErrorAction Stop
        if($item.PSIsContainer -or $item.Length -gt 5MB){throw 'Invalid config file'}
        if($extension -eq '.json'){
            $result=@(Get-DcrJsonConfigSummary (Get-Content -LiteralPath $item.FullName -Raw -ErrorAction Stop))
        }elseif($extension -eq '.toml'){
            $py=Get-Command py.exe -ErrorAction SilentlyContinue
            if(!$py){$status='optional-python-unavailable'}
            else{
                $out=@(& $py.Source -3 (Join-Path $PSScriptRoot 'audit_toml.py') $item.FullName 2>$null)
                if($LASTEXITCODE -ne 0){$status='unreadable-or-unsupported'}
                else{$parsed=($out -join [Environment]::NewLine)|ConvertFrom-Json;$result=@($parsed | Where-Object {$null -ne $_ -and $_.mode -in @('npx','direct','other')})}
            }
        }else{$status='format-not-audited'}
    }catch{$status='unreadable-or-unsupported';$result=@()}
    [pscustomobject]@{status=$status;entries=$result.Count;modes=(@($result.mode|Select-Object -Unique)-join ',');floating=@($result|Where-Object floatingVersion).Count}
}
function Get-DcrClientAudit([switch]$IncludeMissing,[switch]$OnlyToml) {
    $candidates=@(
        @{client='Claude Desktop';path=(Join-Path $env:APPDATA 'Claude\claude_desktop_config.json');type='json'},
        @{client='Claude Code';path=(Join-Path $env:USERPROFILE '.claude.json');type='json'},
        @{client='Cursor';path=(Join-Path $env:USERPROFILE '.cursor\mcp.json');type='json'},
        @{client='Windsurf';path=(Join-Path $env:USERPROFILE '.codeium\windsurf\mcp_config.json');type='json'},
        @{client='Antigravity';path=(Join-Path $env:USERPROFILE '.gemini\antigravity\mcp_config.json');type='json'},
        @{client='VS Code';path=(Join-Path $env:APPDATA 'Code\User\mcp.json');type='json'},
        @{client='Codex';path=(Join-Path $env:USERPROFILE '.codex\config.toml');type='toml'}
    )
    foreach($c in $candidates){
        if($OnlyToml -and $c.type -ne 'toml'){continue}
        try{
            $item=Get-Item -LiteralPath $c.path -Force -ErrorAction Stop
            if($item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)){
                if($IncludeMissing){[pscustomobject]@{client=$c.client;status='unreadable-or-unsupported';entries=0;modes='';floating=0}}
                continue
            }
        }catch{
            if($IncludeMissing){
                $status=if($_.CategoryInfo.Category -eq [System.Management.Automation.ErrorCategory]::ObjectNotFound){'not-found'}else{'unreadable-or-unsupported'}
                [pscustomobject]@{client=$c.client;status=$status;entries=0;modes='';floating=0}
            }
            continue
        }
        $r=Get-DcrCustomAudit $c.path
        [pscustomobject]@{client=$c.client;status=$r.status;entries=$r.entries;modes=$r.modes;floating=$r.floating}
    }
}
function Get-DcrRuntime {
    $node=if(Get-Command node.exe -ErrorAction SilentlyContinue){& node.exe --version 2>$null}else{'missing'}
    $npm=if(Get-Command npm.cmd -ErrorAction SilentlyContinue){& npm.cmd --version 2>$null}else{'missing'}
    $dedicated=Get-DcrInstallInventory | Where-Object kind -eq 'dedicated' | Select-Object -First 1
    [pscustomobject]@{node=$node;npm=$npm;declaredNodeRange=$dedicated.nodeRange;assessment=(Get-DcrNodeAssessment $node $dedicated.nodeRange)}
}
function Restore-DcrManagedBackup([string]$Name){
    if($Name -notmatch '^package-backup-[0-9a-f]{32}$'){throw 'Invalid backup name.'}
    $running=@(Get-CimInstance Win32_Process -Filter "Name='node.exe'" | Where-Object {Test-DcrCommand $_.CommandLine})
    if($running.Count){throw 'Close DCR Remote and local DCR clients before restore.'}
    $state=Join-Path $env:LOCALAPPDATA 'DCRManager-state'
    $backup=Join-Path $state $Name
    if(!(Test-Path -LiteralPath $backup -PathType Container)){throw 'Backup missing.'}
    if((Get-Item -LiteralPath $backup).Attributes -band [IO.FileAttributes]::ReparsePoint){throw 'Backup is a reparse point.'}
    Restore-DcrBackupFiles (Join-Path $env:LOCALAPPDATA 'DesktopCommander') $backup $state { if(@(Get-CimInstance Win32_Process -Filter "Name='node.exe'" | Where-Object {Test-DcrCommand $_.CommandLine}).Count){throw 'DCR started during restore. Original untouched.'} }
}
Export-ModuleMember -Function Test-DcrArgument,Get-DcrJsonConfigSummary,Get-DcrNodeAssessment,Get-DcrBackupVersion,Restore-DcrBackupFiles,Get-DcrInstallInventory,Get-DcrClientAudit,Get-DcrRuntime,Restore-DcrManagedBackup,Get-DcrCustomAudit
