param([switch]$AllowNetwork)
$ErrorActionPreference='Stop'
if(!$AllowNetwork){throw 'Explicit -AllowNetwork required. This test downloads npm packages to a temporary directory.'}
$project=Split-Path -Parent $PSScriptRoot
$root=Join-Path ([IO.Path]::GetTempPath()) ('dcr-artifact-acceptance-'+[guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $root|Out-Null
$names=@('npm_config_userconfig','npm_config_globalconfig','npm_config_cache')
$oldEnv=@{}
foreach($name in $names){$oldEnv[$name]=[Environment]::GetEnvironmentVariable($name,'Process')}
try{
 Import-Module (Join-Path $project 'src/PackageCache.psm1') -Force
 Import-Module (Join-Path $project 'src/RecoveryCenter.psm1')
 $install=Join-Path $root 'install';$state=Join-Path $root 'state'
 $npm=(Get-Command npm.cmd -ErrorAction Stop).Source
 $env:npm_config_userconfig=Join-Path $root 'npmrc'
 $env:npm_config_globalconfig=Join-Path $root 'global-npmrc'
 $env:npm_config_cache=Join-Path $root 'npm-cache'
 $guard={param($path) if(![IO.Path]::GetFullPath($path).StartsWith($root+'\',[StringComparison]::OrdinalIgnoreCase)){throw 'outside fixture'}}
 $runner={param($stage,$spec)
  $old=$ErrorActionPreference
  try{
   $ErrorActionPreference='Continue'
   & $npm install --prefix $stage $spec --save-exact --ignore-scripts --no-audit --no-fund --prefer-offline --fetch-retries 1 --fetch-timeout 30000 *> (Join-Path $root 'npm-output.log')
   $code=$LASTEXITCODE
  }finally{$ErrorActionPreference=$old}
  if($code -ne 0){throw "npm staging failed (exit $code)"}
 }
 Invoke-DcrT6Package $install $state repair absent $guard $runner
 $v=Get-DcrT6Version $install
 Write-Output ('REAL_INSTALL_VERSION='+$v)
 Invoke-DcrT6Package $install $state repair $v $guard $runner
 Write-Output ('REAL_REPAIR_VERSION='+(Get-DcrT6Version $install))
 $back=@(Get-DcrRecoveryInventory $state '' | Where-Object {$_.kind -eq 'package' -and $_.status -eq 'ready'})
 if(!$back.Count){throw 'No valid package backup'}
 Invoke-DcrRecoveryRestore $install $state '' $back[0].name $back[0].token $guard
 if((Get-DcrT6Version $install) -ne $v){throw 'Restored version differs'}
 Write-Output ('REAL_RESTORE_VERSION='+(Get-DcrT6Version $install))
 Write-Output 'REAL_ARTIFACT_ACCEPTANCE_OK'
}finally{
 foreach($name in $names){[Environment]::SetEnvironmentVariable($name,$oldEnv[$name],'Process')}
 if(Test-Path -LiteralPath $root){Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue}
}
