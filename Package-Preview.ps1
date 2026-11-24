$ErrorActionPreference='Stop'
$root=$PSScriptRoot
Push-Location -LiteralPath $root
try{
 if(@(git status --porcelain).Count){throw 'Build from a clean committed tree.'}
 $commit=(git rev-parse HEAD).Trim()
 if($LASTEXITCODE){throw 'Git revision unavailable'}
 New-Item -ItemType Directory -Path (Join-Path $root 'bin'),(Join-Path $root 'dist') -Force|Out-Null
 go build -trimpath -buildvcs=true -ldflags '-s -w' -o bin/dcr-manager.exe ./cmd/dcr-manager
 if($LASTEXITCODE){throw 'Build failed'}
 $version=(& .\bin\dcr-manager.exe --version).Trim()
 if($LASTEXITCODE -or $version -notmatch '^\d+\.\d+\.\d+-rc\.\d+$'){throw 'Expected a release candidate'}
 $name="DCR-Manager-$version-windows-amd64"
 $dist=Join-Path $root 'dist'
 $zip=Join-Path $dist ($name+'.zip')
 if(Test-Path -LiteralPath $zip){throw 'Candidate archive already exists; do not overwrite a release artifact'}
 $stage=Join-Path $dist ($name+'-'+[guid]::NewGuid().ToString('N'))
 New-Item -ItemType Directory -Path $stage|Out-Null
 $files=@('bin/dcr-manager.exe','Launch-TUI.cmd','src/Bridge.ps1','src/Core.psm1','src/Operations.psm1','src/Maintenance.psm1','src/PackageCache.psm1','src/Recovery.psm1','src/RecoveryCenter.psm1','src/Diagnostics.psm1','src/MaintenanceAction.ps1','src/audit_toml.py','docs/RELEASE-CHECKLIST.md','docs/RECOVERY.md','CHANGELOG.md')
 foreach($f in $files){
  $target=Join-Path $stage $f
  New-Item -ItemType Directory -Path (Split-Path $target) -Force|Out-Null
  Copy-Item -LiteralPath (Join-Path $root $f) -Destination $target -ErrorAction Stop
 }
 $manifest=[ordered]@{version=$version;commit=$commit;platform='windows-amd64';files=@(foreach($f in $files){[pscustomobject]@{path=$f;sha256=(Get-FileHash -LiteralPath (Join-Path $stage $f) -Algorithm SHA256).Hash.ToLowerInvariant()}})}
 $manifest|ConvertTo-Json -Depth 5|Set-Content -LiteralPath (Join-Path $stage 'manifest.json') -Encoding UTF8
 Compress-Archive -Path (Join-Path $stage '*') -DestinationPath $zip
 $extract=Join-Path $dist ('verify-'+[guid]::NewGuid().ToString('N'))
 Expand-Archive -LiteralPath $zip -DestinationPath $extract
 $m=Get-Content -LiteralPath (Join-Path $extract 'manifest.json') -Raw|ConvertFrom-Json
 foreach($f in $m.files){if((Get-FileHash -LiteralPath (Join-Path $extract $f.path) -Algorithm SHA256).Hash -ne $f.sha256){throw 'Extracted hash mismatch'}}
 $actual=& (Join-Path $extract 'bin\dcr-manager.exe') --version
 if($LASTEXITCODE -or $actual -ne $version){throw 'Extracted version mismatch'}
 $sw=[Diagnostics.Stopwatch]::StartNew()
 $snapshot=& (Join-Path $extract 'bin\dcr-manager.exe') --snapshot | ConvertFrom-Json
 $sw.Stop()
 if($LASTEXITCODE -or $snapshot.ManagerVersion -ne $version){throw 'Extracted diagnostic smoke test failed'}
 $hash=(Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash.ToLowerInvariant()
 "$hash  $name.zip"|Set-Content -LiteralPath ($zip+'.sha256') -Encoding ASCII
 [pscustomobject]@{version=$version;commit=$commit;archive=$zip;bytes=(Get-Item -LiteralPath $zip).Length;sha256=$hash;diagnosticMilliseconds=$sw.ElapsedMilliseconds;remoteProcesses=$snapshot.Status.RemoteProcesses;processControlReady=$snapshot.ProcessControlReady;verifiedFiles=$m.files.Count}|ConvertTo-Json -Compress
 # Remove only uniquely created staging/extraction directories after successful verification.
 Remove-Item -LiteralPath $stage,$extract -Recurse -Force
}finally{Pop-Location}
