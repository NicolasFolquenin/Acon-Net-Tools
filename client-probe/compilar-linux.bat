@echo off
chcp 65001 >nul
echo ==================================================================
echo   UEPG Net Tools - Compilação da Sonda Client para Linux (Debian)
echo ==================================================================
echo.

echo [1/3] Compilando binário Linux amd64 (x86_64)...
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -tags cli -ldflags="-s -w" -o build\linux\uepg-probe-linux-amd64 .
if errorlevel 1 (
    echo [ERRO] Falha ao compilar para Linux amd64!
    pause
    exit /b 1
)

echo [2/3] Compilando binário Linux arm64 (Raspberry Pi / ARM)...
set GOARCH=arm64
go build -tags cli -ldflags="-s -w" -o build\linux\uepg-probe-linux-arm64 .
if errorlevel 1 (
    echo [ERRO] Falha ao compilar para Linux arm64!
    pause
    exit /b 1
)

echo [3/3] Publicando binários na pasta de downloads do Central Server...
if exist "..\central-server\public\downloads\" (
    copy /y build\linux\uepg-probe-linux-amd64 ..\central-server\public\downloads\ >nul
    copy /y build\linux\uepg-probe-linux-arm64 ..\central-server\public\downloads\ >nul
    echo [OK] Binários copiados para central-server\public\downloads\ com sucesso!
) else (
    echo [AVISO] Pasta central-server\public\downloads\ não localizada.
)

echo.
echo ==================================================================
echo   Compilação Linux finalizada com sucesso!
echo ==================================================================
pause
