# Portable Go: %USERPROFILE%\.gbootcms-tools\go
$Tools = Join-Path $env:USERPROFILE ".gbootcms-tools"
$Go = Join-Path $Tools "go"
if (Test-Path (Join-Path $Go "bin")) {
  $env:Path = (Join-Path $Go "bin") + ";" + $env:Path
  $env:GOROOT = $Go
}
Write-Host "go=$((go version 2>$null) -join ' ')"
