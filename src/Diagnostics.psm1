Import-Module (Join-Path $PSScriptRoot 'PackageCache.psm1')

function Get-DcrBoundedLogs([string]$CacheRoot){
    $rows=New-Object 'System.Collections.Generic.List[object]'
    $known=$true
    try{
        if(!$CacheRoot){throw 'unknown cache'}
        $dir=Join-Path $CacheRoot '_logs'
        Assert-DcrPlainPath $dir
        if(Test-Path -LiteralPath $dir -PathType Container){
            $files=@(Get-ChildItem -LiteralPath $dir -File -Filter '*debug*.log' -ErrorAction Stop|Sort-Object LastWriteTimeUtc -Descending|Select-Object -First 100)
            $deadline=[datetime]::UtcNow.AddSeconds(5)
            foreach($f in $files){
                if([datetime]::UtcNow -gt $deadline){$known=$false;break}
                if($f.Attributes -band [IO.FileAttributes]::ReparsePoint -or $f.Length -gt 2MB){$known=$false;continue}
                try{
                    # Fixed names only; never export arbitrary filenames or log lines.
                    if($f.Name -notmatch '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}_[0-9]{2}_[0-9]{2}_[0-9]{3}Z-debug-[0-9]+\.log$'){continue}
                    $stream=[IO.File]::Open($f.FullName,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::ReadWrite)
                    try{
                        if($stream.Length -gt 2MB){$known=$false;continue}
                        $buffer=New-Object byte[] ([int]$stream.Length)
                        $offset=0
                        while($offset -lt $buffer.Length){$n=$stream.Read($buffer,$offset,$buffer.Length-$offset);if(!$n){break};$offset+=$n}
                        $raw=[Text.Encoding]::UTF8.GetString($buffer,0,$offset)
                    }finally{$stream.Dispose()}
                    if($raw -notmatch '(?i)@wonderwhy-er(?:/|\\)desktop-commander'){continue}
                    $codes=@([regex]::Matches($raw,'\b(EBUSY|EPERM|EACCES|ENOTFOUND|ETIMEDOUT|ECONNRESET)\b')|ForEach-Object Value|Sort-Object -Unique)
                    $rows.Add([pscustomobject]@{name=$f.Name;codes=$codes;modifiedUtc=$f.LastWriteTimeUtc.ToString('yyyy-MM-ddTHH:mm:ssZ')})
                    if($rows.Count -ge 20){break}
                }catch{$known=$false}
            }
        }elseif(Test-Path -LiteralPath $dir){$known=$false}
    }catch{$known=$false}
    [pscustomobject]@{known=$known;logs=@($rows.ToArray())}
}
Export-ModuleMember -Function Get-DcrBoundedLogs
