$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$serverDir = Split-Path -Parent $scriptDir
$clientDir = Join-Path $serverDir "..\client"
$clientDist = Join-Path $clientDir "dist"
$serverClient = Join-Path $serverDir "client"

Write-Host "Building client..."
Push-Location $clientDir
try {
    npm run build
} finally {
    Pop-Location
}

Write-Host "Removing old client from server..."
if (Test-Path $serverClient) {
    Remove-Item -Recurse -Force $serverClient
}

Write-Host "Copying dist to server/client..."
Copy-Item -Recurse $clientDist $serverClient

Write-Host "Done."
