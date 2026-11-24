Set-StrictMode -Version 2
$script:Package = '@wonderwhy-er/desktop-commander'
function Test-DcrCommand([string]$CommandLine) {
    return ($CommandLine -match '(?i)[\\/]@wonderwhy-er[\\/]desktop-commander[\\/]' -or $CommandLine -match '(?i)[\\/]node_modules[\\/]\.bin[\\/]desktop-commander(?:\.cmd)?(?:\s|"|$)')
}
function Select-DcrRemote($Processes) {
    $all=@($Processes | Where-Object { $_.Name -eq 'node.exe' -and (Test-DcrCommand $_.CommandLine) })
    $ids=@($all | Where-Object { $_.CommandLine -match '(?i)(?:\s|")remote(?:\s|"|$)' } | ForEach-Object ProcessId)
    do {
        $old=$ids.Count
        $ids=@($ids + @($all | Where-Object { $ids -contains $_.ParentProcessId } | ForEach-Object ProcessId) | Select-Object -Unique)
    } while($ids.Count -gt $old)
    $all | Where-Object { $ids -contains $_.ProcessId }
}
function Get-DcrPackageSpec([string]$Version,[switch]$Update) {
    if($Update){return "$script:Package@latest"}
    if($Version -notmatch '^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$'){throw 'Unknown version. Use explicit install/update.'}
    return "$script:Package@$Version"
}
function Find-DcrCache([string]$CacheRoot) {
    if(!$CacheRoot){return}
    $npx=Join-Path $CacheRoot '_npx'
    if(!(Test-Path -LiteralPath $npx -PathType Container)){return}
    if((Get-Item -LiteralPath $npx).Attributes -band [IO.FileAttributes]::ReparsePoint){return}
    foreach($dir in Get-ChildItem -LiteralPath $npx -Directory -ErrorAction Stop){
        if($dir.Attributes -band [IO.FileAttributes]::ReparsePoint){continue}
        $file=Join-Path $dir.FullName 'package.json'
        try {
            $json=Get-Content -LiteralPath $file -Raw -ErrorAction Stop | ConvertFrom-Json
            $deps=@($json.dependencies.PSObject.Properties.Name)
            if($deps.Count -eq 1 -and $deps[0] -eq $script:Package -and (Test-Path -LiteralPath (Join-Path $dir.FullName 'node_modules\@wonderwhy-er\desktop-commander'))){$dir}
        } catch { continue }
    }
}
function Get-DcrErrorCodes($Lines) {
    @($Lines | ForEach-Object { [regex]::Matches([string]$_,'\b(EBUSY|EPERM|EACCES|ENOTFOUND|ETIMEDOUT|ECONNRESET)\b') | ForEach-Object Value } | Sort-Object -Unique)
}
function Switch-DcrPackageFiles {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$Root,[Parameter(Mandatory)][string]$Stage,[Parameter(Mandatory)][string]$Backup,
        [scriptblock]$MoveAction={param($from,$to) Move-Item -LiteralPath $from -Destination $to -ErrorAction Stop})
    $names=@('node_modules','package.json','package-lock.json')
    foreach($n in $names){if(!(Test-Path -LiteralPath (Join-Path $Stage $n))){throw "Incomplete stage: $n"}}
    if(!(Test-Path -LiteralPath $Root)){New-Item -ItemType Directory -Path $Root | Out-Null}
    if(!(Test-Path -LiteralPath $Backup)){New-Item -ItemType Directory -Path $Backup | Out-Null}
    foreach($n in $names){if(Test-Path -LiteralPath (Join-Path $Backup $n)){throw 'Backup is not empty'}}
    $moved=@();$installed=@()
    function Move-Safe($from,$to){
        if(Test-Path -LiteralPath $to){throw 'Destination already exists; refusing overwrite'}
        & $MoveAction $from $to
    }
    try{
        foreach($n in $names){
            $dest=Join-Path $Root $n
            if(Test-Path -LiteralPath $dest){Move-Safe $dest (Join-Path $Backup $n);$moved+=$n}
            Move-Safe (Join-Path $Stage $n) $dest;$installed+=$n
        }
    }catch{
        $failure=$_
        $rollbackErrors=@()
        foreach($n in $installed){
            try{Move-Safe (Join-Path $Root $n) (Join-Path $Stage $n)}catch{$rollbackErrors+="stage/$n"}
        }
        foreach($n in $moved){
            try{Move-Safe (Join-Path $Backup $n) (Join-Path $Root $n)}catch{$rollbackErrors+="backup/$n"}
        }
        if($rollbackErrors.Count){throw ('Package switch failed; rollback incomplete: '+($rollbackErrors -join ', ')+'; preserve stage and backup for manual recovery.')}
        throw $failure
    }
}
Export-ModuleMember -Function Test-DcrCommand,Select-DcrRemote,Get-DcrPackageSpec,Find-DcrCache,Get-DcrErrorCodes,Switch-DcrPackageFiles
