# Run this script as Administrator once. A Windows restart is required.
$ErrorActionPreference = "Stop"
Write-Host "Enabling WSL2 and Virtual Machine Platform..." -ForegroundColor Cyan
dism.exe /online /enable-feature /featurename:Microsoft-Windows-Subsystem-Linux /all /norestart
dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart
wsl.exe --set-default-version 2
Write-Host "Restart Windows, start Docker Desktop, then run start-local.ps1." -ForegroundColor Green
