Write-Host "==============================================================================" -ForegroundColor Cyan
Write-Host "Iniciando UEPG-Net-Tools em Modo de Teste" -ForegroundColor Cyan
Write-Host "==============================================================================" -ForegroundColor Cyan
Write-Host ""

$probeDir = $PSScriptRoot
Set-Location $probeDir

# 1. Sincroniza arquivos da interface (frontend -> frontend/dist)
Write-Host "[1/3] Sincronizando arquivos da interface (frontend)..." -ForegroundColor Yellow
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

if (-not (Test-Path "rsrc_windows_amd64.syso")) {
    & $goCmd build -o "build/embed-icon.exe" ./cmd/embed-icon
    & "build/embed-icon.exe"
    Remove-Item -Force "build/embed-icon.exe" -ErrorAction SilentlyContinue
}

# 3. Compila e executa com tags Wails
Write-Host "[2/3] Compilando aplicativo de teste..." -ForegroundColor Yellow
if (-not (Test-Path "build")) { New-Item -ItemType Directory -Path "build" | Out-Null }
& $goCmd build -tags desktop,production -o "build/test-runner.exe" .
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERRO] Falha ao compilar com Go!" -ForegroundColor Red
    Read-Host "Pressione Enter para sair..."
    exit 1
}

Write-Host "[3/3] Iniciando janela grafica do aplicativo..." -ForegroundColor Green
Write-Host "? Logs e telemetria serao exibidos nesta janela." -ForegroundColor Gray
Write-Host "? Feche a janela do aplicativo ou aperte Ctrl+C para encerrar." -ForegroundColor Gray
Write-Host ""

& "build/test-runner.exe"