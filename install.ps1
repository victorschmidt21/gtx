# GTX — Instalador para Windows (sem WSL)
# Uso: iwr -useb https://raw.githubusercontent.com/victorschmidt21/gtx/main/install.ps1 | iex
# Ou localmente: .\install.ps1

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:USERPROFILE\.local\bin"
)

$ErrorActionPreference = "Stop"
$Repo = "victorschmidt21/gtx"
$BinaryName = "gtx.exe"
$AssetName = "gtx-windows-amd64.exe"

Write-Host "GTX Installer para Windows"
Write-Host "--------------------------"

# Determina a versão a instalar
if ($Version -eq "latest") {
    Write-Host "Buscando versão mais recente..."
    $releaseUrl = "https://api.github.com/repos/$Repo/releases/latest"
    $release = Invoke-RestMethod -Uri $releaseUrl -Headers @{ "User-Agent" = "gtx-installer" }
    $Version = $release.tag_name
}

Write-Host "Instalando GTX $Version..."

# Cria o diretório de instalação se não existir
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}

# URL do binário
$downloadUrl = "https://github.com/$Repo/releases/download/$Version/$AssetName"
$destPath = Join-Path $InstallDir $BinaryName

Write-Host "Baixando de $downloadUrl..."
Invoke-WebRequest -Uri $downloadUrl -OutFile $destPath

# Adiciona ao PATH do usuário se necessário
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$InstallDir*") {
    Write-Host "Adicionando $InstallDir ao PATH do usuário..."
    [Environment]::SetEnvironmentVariable("PATH", "$userPath;$InstallDir", "User")
    $env:PATH = "$env:PATH;$InstallDir"
}

Write-Host ""
Write-Host "GTX instalado em $destPath"
Write-Host ""
Write-Host "Próximos passos:"
Write-Host "  1. Abra um novo terminal (para o PATH ser atualizado)"
Write-Host "  2. Execute: gtx init"
Write-Host "  3. Reinicie o Claude Code"
Write-Host ""
Write-Host "Para verificar a instalação do hook: gtx init --verify"
