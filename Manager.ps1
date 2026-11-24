param([ValidateSet('tui','status','doctor','inventory','clients','runtime','health','backups','config')][string]$Mode='tui',[string]$ConfigPath)
$ErrorActionPreference='Stop'
Import-Module (Join-Path $PSScriptRoot 'src\Operations.psm1') -Force
Import-Module (Join-Path $PSScriptRoot 'src\Maintenance.psm1') -Force
if($Mode -eq 'inventory'){Get-DcrInstallInventory | ConvertTo-Json -Depth 4;exit 0}
if($Mode -eq 'clients'){Get-DcrClientAudit | ConvertTo-Json -Depth 4;exit 0}
if($Mode -eq 'runtime'){Get-DcrRuntime | ConvertTo-Json -Depth 4;exit 0}
if($Mode -eq 'health'){Get-DcrProcessHealth | ConvertTo-Json -Depth 4;exit 0}
if($Mode -eq 'backups'){Get-DcrBackupReview | ConvertTo-Json -Depth 4;exit 0}
if($Mode -eq 'config'){if(!$ConfigPath){throw 'Specify -ConfigPath'};Get-DcrCustomAudit $ConfigPath | ConvertTo-Json;exit 0}
if($Mode -eq 'status'){Get-DcrStatus | ConvertTo-Json;exit 0}
if($Mode -eq 'doctor'){Get-DcrStatus | Format-List;Get-DcrInventory | Format-Table;Get-DcrLogSummary | ConvertTo-Json -Depth 4;exit 0}
function Confirm-Action([string]$Message){(Read-Host "$Message [s/N]") -match '^(s|sim)$'}
function Show-Header {
    Clear-Host
    Write-Host '  DCR MANAGER   /   v1.2.1' -ForegroundColor Cyan
    Write-Host '  Desktop Commander  |  Windows  |  Community toolkit' -ForegroundColor DarkGray
    Write-Host ('  '+'='*62) -ForegroundColor DarkCyan
    $s=Get-DcrStatus
    Write-Host ('  PACKAGE   '+$s.version+'   |   NODE '+$s.nodeAvailable+'   |   NPM '+$s.npmAvailable)
    Write-Host ('  REMOTE    '+$(if($s.remoteProcesses){'ACTIVE'}else{'STOPPED'})+'  /  '+$s.remoteRoots+' session(s), '+$s.remoteProcesses+' process(es)') -ForegroundColor $(if($s.remoteProcesses){'Green'}else{'Yellow'})
    Write-Host ('  CACHE     '+$s.cacheCandidates+' DCR candidate(s)  |  MCP '+$s.otherDcrProcesses)
    Write-Host '  Local activity is not proof of cloud connectivity.' -ForegroundColor DarkGray
    Write-Host ('  '+'='*62) -ForegroundColor DarkCyan
    Write-Host '  CONNECTION                   MAINTENANCE' -ForegroundColor Cyan
    Write-Host '  [1] Start Remote             [5] Repair / install'
    Write-Host '  [2] Stop Remote              [6] Update package'
    Write-Host '  [3] Restart Remote           [7] Quarantine npm cache'
    Write-Host '  [4] Process inventory        [8] Backups'
    Write-Host ''
    Write-Host '  DIAGNOSTICS                  SUPPORT' -ForegroundColor Cyan
    Write-Host '  [9] npm error history         [P] Device portal'
    Write-Host '  [E] Export safe report        [R] Refresh'
    Write-Host '  [I] Install inventory         [C] Client config audit'
    Write-Host '  [V] Runtime compatibility    [B] Restore package backup'
    Write-Host '  [H] Process health            [K] Restore cache'
    Write-Host '  [T] Backup retention review   [0] Exit'
}
$mutex=New-Object System.Threading.Mutex($false,'Local\DCRManager')
$owned=$false
try {
    try{$owned=$mutex.WaitOne(0)}catch [System.Threading.AbandonedMutexException]{$owned=$true}
    if(!$owned){throw 'Manager already open.'}
    $Host.UI.RawUI.WindowTitle='DCR Manager | Community toolkit'
    while($true){
        try{Show-Header}catch{Write-Host $_ -ForegroundColor Red;Write-Host 'Use R to retry or 0 to quit.'}
        $c=Read-Host '  >'
        if($c -eq '0'){break}
        if($c -eq 'r'){continue}
        try {
            switch($c){
                '1'{Start-DcrRemote}
                '2'{if(Confirm-Action 'Stop Remote and disconnect ChatGPT?'){Stop-DcrRemote}}
                '3'{if(Confirm-Action 'Restart Remote? Active remote tasks will disconnect.'){Stop-DcrRemote;Start-DcrRemote}}
                '4'{Get-DcrInventory | Format-Table -AutoSize | Out-Host}
                '5'{if(Confirm-Action 'Repair current version or install if missing?'){Install-DcrPackage}}
                '6'{if(Confirm-Action 'Install latest package from npm?'){Install-DcrPackage -Update}}
                '7'{
                    $items=@(Get-DcrCacheCandidates)
                    if(!$items.Count){Write-Host 'No exclusive DCR caches found.';break}
                    for($i=0;$i -lt $items.Count;$i++){Write-Host ('  ['+($i+1)+'] '+$items[$i].FullName)}
                    $n=0; if([int]::TryParse((Read-Host 'Cache number (0 cancels)'),[ref]$n) -and $n -ge 1 -and $n -le $items.Count){
                        if(Confirm-Action 'Move selected cache to backup?'){Move-DcrCache $items[$n-1].FullName}
                    }
                }
                '8'{Get-DcrBackups | Format-Table -AutoSize | Out-Host}
                '9'{Get-DcrLogSummary | Format-List | Out-Host}
                'e'{Export-DcrReport}
                'i'{Get-DcrInstallInventory | Format-Table -AutoSize | Out-Host}
                'c'{Get-DcrClientAudit | Format-Table -AutoSize | Out-Host}
                'v'{Get-DcrRuntime | Format-List | Out-Host}
                'b'{
                    $items=@(Get-DcrBackups | Where-Object {$_.Name -match '^package-backup-[0-9a-f]{32}$'})
                    if(!$items.Count){Write-Host 'No package backups found.';break}
                    for($i=0;$i -lt $items.Count;$i++){Write-Host ('  ['+($i+1)+'] '+$items[$i].Name)}
                    $n=0
                    if([int]::TryParse((Read-Host 'Backup number (0 cancels)'),[ref]$n) -and $n -ge 1 -and $n -le $items.Count){
                        if(Confirm-Action 'Restore selected package backup? All DCR clients must be stopped.'){Restore-DcrManagedBackup $items[$n-1].Name}
                    }
                }
                'h'{Get-DcrProcessHealth | Format-Table -AutoSize | Out-Host;Write-Host 'Missing parent is a diagnostic clue, not proof of a stuck process.'}
                't'{Get-DcrBackupReview | Format-Table -AutoSize | Out-Host;Write-Host 'Review only; no backups deleted.'}
                'k'{
                    $items=@(Get-DcrBackups | Where-Object {$_.Name -match '^cache-backup-[a-f0-9]{32}$'})
                    if(!$items.Count){Write-Host 'No cache backups found.';break}
                    for($i=0;$i -lt $items.Count;$i++){Write-Host ('  ['+($i+1)+'] '+$items[$i].Name)}
                    $n=0
                    if([int]::TryParse((Read-Host 'Cache backup number (0 cancels)'),[ref]$n) -and $n -ge 1 -and $n -le $items.Count){
                        if(Confirm-Action 'Restore cache to its original empty path?'){Restore-DcrManagedCache $items[$n-1].Name}
                    }
                }
                'p'{Start-Process 'https://mcp.desktopcommander.app/'}
                default{Write-Host 'Invalid choice.' -ForegroundColor Yellow}
            }
        }catch{Write-Host ('ERROR: '+$_.Exception.Message) -ForegroundColor Red}
        [void](Read-Host '  Enter to continue')
    }
}finally{if($owned){$mutex.ReleaseMutex()};$mutex.Dispose()}
