Write-Host "==============================================================================" -ForegroundColor Cyan
Write-Host "Gerador do UEPG-Net-Tools para Windows" -ForegroundColor Cyan
Write-Host "==============================================================================" -ForegroundColor Cyan
Write-Host ""

$probeDir = $PSScriptRoot
Set-Location $probeDir

# 1. Sincroniza arquivos da interface (frontend -> frontend/dist)
Write-Host "[1/4] Sincronizando arquivos da interface (frontend)..." -ForegroundColor Yellow
if (-not (Test-Path "frontend/dist")) { New-Item -ItemType Directory -Path "frontend/dist" | Out-Null }
Copy-Item "frontend/index.html", "frontend/main.js", "frontend/style.css" -Destination "frontend/dist/" -Force
if (Test-Path "frontend/uepg-logo.png") { Copy-Item "frontend/uepg-logo.png" -Destination "frontend/dist/" -Force }

# 2. Localiza Go
$goCmd = "go"
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    if (Test-Path "C:\Program Files\Go\bin\go.exe") {
        $goCmd = "C:\Program Files\Go\bin\go.exe"
    } else {
        Write-Host "[ERRO] Compilador Go nao encontrado no sistema!" -ForegroundColor Red
        Read-Host "Pressione Enter para sair..."
        exit 1
    }
}

# 3. Assegura ?cone oficial no execut?vel Windows (.syso)
Write-Host "[2/4] Assegurando icone oficial e metadados (.syso)..." -ForegroundColor Yellow
if (-not (Test-Path "rsrc_windows_amd64.syso")) {
    & $goCmd build -o "build/embed-icon.exe" ./cmd/embed-icon
    & "build/embed-icon.exe"
    Remove-Item -Force "build/embed-icon.exe" -ErrorAction SilentlyContinue
}

# 4. Compilacao do Executavel Oficial de Producao
Write-Host "[3/4] Compilando executavel oficial de producao..." -ForegroundColor Yellow
if (-not (Test-Path "build/bin")) { New-Item -ItemType Directory -Path "build/bin" | Out-Null }

& $goCmd build -tags desktop,production -ldflags "-H windowsgui" -o "build/bin/uepg-net-tools.exe" .
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERRO] Falha ao compilar com Go!" -ForegroundColor Red
    Read-Host "Pressione Enter para sair..."
    exit 1
}

Write-Host "[OK] Executavel gerado: build\bin\uepg-net-tools.exe" -ForegroundColor Green
Copy-Item "config.json" -Destination "build/bin/config.json" -Force
if (Test-Path ".env.example") { Copy-Item ".env.example" -Destination "build/bin/.env.example" -Force }

# 5. Verificacao do NSIS para gerar instalador
Write-Host ""
Write-Host "[4/4] Verificando gerador de instalador (NSIS)..." -ForegroundColor Yellow

$makensis = $null
if (Get-Command makensis -ErrorAction SilentlyContinue) {
    $makensis = "makensis"
} elseif (Test-Path "C:\Program Files (x86)\NSIS\makensis.exe") {
    $makensis = "C:\Program Files (x86)\NSIS\makensis.exe"
} elseif (Test-Path "C:\Program Files\NSIS\makensis.exe") {
    $makensis = "C:\Program Files\NSIS\makensis.exe"
}

if ($makensis) {
    Write-Host "[OK] NSIS detectado: $makensis" -ForegroundColor Green
    Write-Host "     Gerando assistente oficial de instalacao..." -ForegroundColor Gray
    Push-Location "build/windows"
    & $makensis installer.nsi
    Pop-Location
    if (Test-Path "build/bin/UEPG-Net-Tools-Instalador.exe") {
        Write-Host ""
        Write-Host "==============================================================================" -ForegroundColor Green
        Write-Host "SUCESSO COMPLETO!" -ForegroundColor Green
        Write-Host "  1. Instalador Oficial: build\bin\UEPG-Net-Tools-Instalador.exe" -ForegroundColor Cyan
        Write-Host "  2. Executavel Portatil: build\bin\uepg-net-tools.exe" -ForegroundColor Cyan
        Write-Host "==============================================================================" -ForegroundColor Green
    }
} else {
    Write-Host ""
    Write-Host "==============================================================================" -ForegroundColor Yellow
    Write-Host "Executavel portatil com icone gerado com sucesso em: build\bin\uepg-net-tools.exe" -ForegroundColor Green
    Write-Host ""
    Write-Host "[INFO] Para gerar o arquivo Instalador com assistente 'Avancar e Concluir':" -ForegroundColor Gray
    Write-Host "       Execute no PowerShell: winget install NSIS.NSIS" -ForegroundColor Cyan
    Write-Host "       Depois execute este script novamente." -ForegroundColor Gray
    Write-Host "==============================================================================" -ForegroundColor Yellow
}

Write-Host ""
Read-Host "Pressione Enter para fechar..."