$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
Get-Process oil-supervision-server -ErrorAction SilentlyContinue | Stop-Process -Force
$Docker = (Get-Command docker -ErrorAction SilentlyContinue).Source
if (-not $Docker) { $Docker = "C:\Program Files\Docker\Docker\resources\bin\docker.exe" }
& $Docker compose -f (Join-Path $Root "docker-compose.yml") down -v
Write-Host "Demo database reset. Run start-local.ps1 to seed it again."
