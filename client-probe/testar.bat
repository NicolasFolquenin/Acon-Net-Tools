@echo off
setlocal
title UEPG-Net-Tools - Modo de Teste
echo ==============================================================================
echo Iniciando UEPG-Net-Tools em Modo de Teste...
echo ==============================================================================
echo.

cd /d "%~dp0"

echo [1/3] Sincronizando arquivos da interface (frontend)...
if not exist "frontend\dist" mkdir "frontend\dist"
copy /y "frontend\index.html" "frontend\dist\" >nul
copy /y "frontend\main.js" "frontend\dist\" >nul
copy /y "frontend\style.css" "frontend\dist\" >nul
if exist "frontend\uepg-logo.png" copy /y "frontend\uepg-logo.png" "frontend\dist\" >nul

echo [2/3] Localizando compilador Go...
set "GO_CMD=go"
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    if exist "C:\Program Files\Go\bin\go.exe" (
        set "GO_CMD=C:\Program Files\Go\bin\go.exe"
    ) else (
        echo [ERRO] Compilador Go nao encontrado no sistema!
        echo        Instale o Go ou adicione C:\Program Files\Go\bin ao PATH.
        pause
        exit /b 1
    )
)

if not exist "rsrc_windows_amd64.syso" (
    "%GO_CMD%" build -o "build\embed-icon.exe" ./cmd/embed-icon
    "build\embed-icon.exe"
    del "build\embed-icon.exe" >nul 2>nul
)

echo [3/3] Compilando e iniciando com tags Wails (desktop,production)...
if not exist "build" mkdir "build"
"%GO_CMD%" build -tags desktop,production -o "build\test-runner.exe" .
if %ERRORLEVEL% neq 0 (
    echo.
    echo [ERRO] Falha ao compilar o aplicativo de teste.
    pause
    exit /b 1
)

echo.
echo ==============================================================================
echo [OK] Aplicativo compilado! Abrindo janela grafica...
echo * Logs e telemetria serao exibidos abaixo em tempo real.
echo * Para fechar, basta fechar a janela do app ou apertar Ctrl+C.
echo ==============================================================================
echo.

"build\test-runner.exe"

pause