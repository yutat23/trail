# Trail Installer Script for winget
param(
    [string]$InstallLocation = "$env:LOCALAPPDATA\Programs\Trail"
)

# Create installation directory
if (!(Test-Path $InstallLocation)) {
    New-Item -ItemType Directory -Path $InstallLocation -Force | Out-Null
}

# Copy trail.exe to installation directory
$exePath = Join-Path $PSScriptRoot "trail.exe"
if (Test-Path $exePath) {
    Copy-Item $exePath $InstallLocation -Force
    Write-Host "Trail has been installed to: $InstallLocation" -ForegroundColor Green
} else {
    Write-Error "trail.exe not found in the current directory"
    exit 1
}

# Add to PATH if not already present
$currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($currentPath -notlike "*$InstallLocation*") {
    [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$InstallLocation", "User")
    Write-Host "Added Trail to PATH" -ForegroundColor Green
}

Write-Host "Installation completed successfully!" -ForegroundColor Green
Write-Host "You can now use 'trail' command from anywhere." -ForegroundColor Yellow 