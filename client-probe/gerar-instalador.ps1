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

# 5. Compilando assistente nativo de instalacao (com pop-up, atalhos e selecao de pasta)
Write-Host "[4/5] Compilando assistente nativo de instalacao (pop-up, atalhos, selecao de pasta)..." -ForegroundColor Yellow
if (-not (Test-Path "cmd/installer/payload")) { New-Item -ItemType Directory -Path "cmd/installer/payload" | Out-Null }
Copy-Item "build/bin/uepg-net-tools.exe" -Destination "cmd/installer/payload/uepg-net-tools.exe" -Force
Copy-Item "config.json" -Destination "cmd/installer/payload/config.json" -Force
Copy-Item "build/windows/icon.ico" -Destination "cmd/installer/payload/icon.ico" -Force
if (Test-Path "rsrc_windows_amd64.syso") { Copy-Item "rsrc_windows_amd64.syso" -Destination "cmd/installer/rsrc_windows_amd64.syso" -Force }

& $goCmd build -ldflags "-H windowsgui" -o "build/bin/UEPG-Net-Tools-Instalador.exe" ./cmd/installer
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Instalador oficial gerado: build\bin\UEPG-Net-Tools-Instalador.exe" -ForegroundColor Green
}

# 6. Verificacao do NSIS para gerar instalador adicional (opcional)
Write-Host ""
Write-Host "[5/5] Verificando gerador adicional NSIS (opcional)..." -ForegroundColor Yellow

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
    Write-Host "     Gerando assistente oficial com NSIS..." -ForegroundColor Gray
    Push-Location "build/windows"
    & $makensis installer.nsi
    Pop-Location
}

# 7. Sincroniza executaveis para o download web no Servidor Central
$centralDownloads = "..\central-server\public\downloads"
if (Test-Path "..\central-server\public") {
    if (-not (Test-Path $centralDownloads)) {
        New-Item -ItemType Directory -Force -Path $centralDownloads | Out-Null
    }
    if (Test-Path "build/bin/uepg-net-tools.exe") {
        Copy-Item "build/bin/uepg-net-tools.exe" "$centralDownloads\uepg-net-tools.exe" -Force
    }
    if (Test-Path "build/bin/UEPG-Net-Tools-Instalador.exe") {
        Copy-Item "build/bin/UEPG-Net-Tools-Instalador.exe" "$centralDownloads\UEPG-Net-Tools-Instalador.exe" -Force
    }
    Write-Host "[OK] Instalador sincronizado para a central web (pasta public/downloads)" -ForegroundColor Green
}

Write-Host ""
Read-Host "Pressione Enter para fechar..."