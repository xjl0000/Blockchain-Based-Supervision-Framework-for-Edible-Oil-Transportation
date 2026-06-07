$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Backend = Join-Path $Root "application\backend"
$Web = Join-Path $Root "application\web"

$Docker = (Get-Command docker -ErrorAction SilentlyContinue).Source
if (-not $Docker -and (Test-Path "C:\Program Files\Docker\Docker\resources\bin\docker.exe")) {
    $Docker = "C:\Program Files\Docker\Docker\resources\bin\docker.exe"
}
if (-not $Docker) {
    throw "Docker was not found. Install Docker Desktop first."
}
$Go = (Get-Command go -ErrorAction SilentlyContinue).Source
if (-not $Go -and (Test-Path "C:\Program Files\Go\bin\go.exe")) {
    $Go = "C:\Program Files\Go\bin\go.exe"
}
if (-not $Go) {
    throw "Go was not found. Install Go 1.20 or later."
}
if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    throw "Node.js and npm were not found."
}

Write-Host "[1/4] Starting MySQL 8..." -ForegroundColor Cyan
& $Docker info *> $null
if ($LASTEXITCODE -ne 0) {
    throw "Docker engine is not running. Run setup-docker.ps1 as Administrator, restart Windows, then start Docker Desktop."
}

$imageReady = $false
for ($attempt = 1; $attempt -le 5; $attempt++) {
    Write-Host "Pulling MySQL image (attempt $attempt/5)..." -ForegroundColor Cyan
    & $Docker pull mysql:8.0
    if ($LASTEXITCODE -eq 0) {
        $imageReady = $true
        break
    }
    Write-Host "Image download interrupted. Retrying in 5 seconds..." -ForegroundColor Yellow
    Start-Sleep -Seconds 5
}
if (-not $imageReady) {
    throw "Unable to download mysql:8.0 after 5 attempts. Check Docker Desktop network/proxy settings and run start-local.ps1 again."
}

& $Docker compose -f (Join-Path $Root "docker-compose.yml") up -d mysql
if ($LASTEXITCODE -ne 0) {
    throw "Unable to create the MySQL container."
}
$ready = ""
for ($i = 0; $i -lt 30; $i++) {
    $ready = (& $Docker inspect --format="{{.State.Health.Status}}" oil-supervision-mysql 2>$null | Out-String).Trim()
    if ($ready -eq "healthy") { break }
    Start-Sleep -Seconds 2
}
if ($ready -ne "healthy") { throw "MySQL startup timed out. Check Docker Desktop." }

Write-Host "[2/4] Building frontend..." -ForegroundColor Cyan
Push-Location $Web
if (-not (Test-Path "node_modules")) { npm install }
npm run build:prod
Pop-Location
Copy-Item -Path (Join-Path $Web "dist\*") -Destination (Join-Path $Backend "dist") -Recurse -Force

Write-Host "[3/4] Starting Go backend..." -ForegroundColor Cyan
Get-Process oil-supervision-server -ErrorAction SilentlyContinue | Stop-Process -Force
Push-Location $Backend
& $Go build -o oil-supervision-server.exe .
Start-Process -FilePath (Join-Path $Backend "oil-supervision-server.exe") -WorkingDirectory $Backend -WindowStyle Hidden
Pop-Location
Start-Sleep -Seconds 3

Write-Host "[4/4] System ready: http://127.0.0.1:9090" -ForegroundColor Green
Write-Host "Demo accounts: admin / supplier / factory / driver / retailer / regulator; password: 123456"
Start-Process "http://127.0.0.1:9090"
