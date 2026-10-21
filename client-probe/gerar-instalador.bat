@echo off
setlocal
title UEPG-Net-Tools - Gerador de Executavel e Instalador
echo ==============================================================================
echo Gerador do UEPG-Net-Tools para Windows
echo ==============================================================================
echo.

cd /d "%~dp0"

echo [1/4] Sincronizando arquivos visuais da interface (frontend)...
if not exist "frontend\dist" mkdir "frontend\dist"
copy /y "frontend\index.html" "frontend\dist\" >nul
copy /y "frontend\main.js" "frontend\dist\" >nul
copy /y "frontend\style.css" "frontend\dist\" >nul
if exist "frontend\uepg-logo.png" copy /y "frontend\uepg-logo.png" "frontend\dist\" >nul

echo [2/4] Localizando compilador Go...
set "GO_CMD=go"
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    if exist "C:\Program Files\Go\bin\go.exe" (
        set "GO_CMD=C:\Program Files\Go\bin\go.exe"
    ) else (
        echo [ERRO] Compilador Go nao encontrado no sistema!
        pause
        exit /b 1
    )
)

echo [3/4] Assegurando icone oficial e metadados (.syso)...
if not exist "rsrc_windows_amd64.syso" (
    "%GO_CMD%" build -o "build\embed-icon.exe" ./cmd/embed-icon
    "build\embed-icon.exe"
    del "build\embed-icon.exe" >nul 2>nul
)

echo [4/4] Compilando executavel oficial de producao com Go...
if not exist "build\bin" mkdir "build\bin"

"%GO_CMD%" build -tags desktop,production -ldflags "-H windowsgui" -o "build\bin\uepg-net-tools.exe" .
if %ERRORLEVEL% neq 0 (
    echo.
    echo [ERRO] Falha ao compilar o executavel com Go!
    pause
    exit /b 1
)

copy /y "config.json" "build\bin\config.json" >nul
if exist ".env.example" copy /y ".env.example" "build\bin\.env.example" >nul
(echo @echo off & echo title UEPG-Net-Tools & echo cd /d "%%~dp0" & echo uepg-net-tools.exe) > "build\bin\executar.bat"


echo.
echo ==============================================================================
echo ? O SEU EXECUTAVEL COM ICONE OFICIAL ESTA PRONTO PARA USO:
echo   Arquivo: build\bin\uepg-net-tools.exe
echo   Configuracao: build\bin\config.json
echo ==============================================================================
echo.

set "MAKENSIS_CMD="
where makensis >nul 2>nul
if %ERRORLEVEL% equ 0 set "MAKENSIS_CMD=makensis"
if exist "%ProgramFiles(x86)%\NSIS\makensis.exe" set "MAKENSIS_CMD=%ProgramFiles(x86)%\NSIS\makensis.exe"
if exist "%ProgramFiles%\NSIS\makensis.exe" set "MAKENSIS_CMD=%ProgramFiles%\NSIS\makensis.exe"

if "%MAKENSIS_CMD%"=="" goto no_nsis

echo [OK] NSIS detectado: %MAKENSIS_CMD%
echo      Gerando assistente oficial de instalacao...
pushd "build\windows"
"%MAKENSIS_CMD%" installer.nsi
popd

if exist "build\bin\UEPG-Net-Tools-Instalador.exe" (
    echo.
    echo ==============================================================================
    echo ? INSTALADOR COM ASSISTENTE GERADO COM SUCESSO!
    echo   Instalador: build\bin\UEPG-Net-Tools-Instalador.exe
    echo   Executavel: build\bin\uepg-net-tools.exe
    echo ==============================================================================
    goto sync_server
)

:no_nsis
echo.
echo [DICA] O executavel portatil 'uepg-net-tools.exe' ja funciona perfeitamente com icone!
echo        Se quiser criar tambem o Instalador (.exe com assistente 'Avancar e Concluir'):
echo        Instale o NSIS no terminal: winget install NSIS.NSIS
echo        E execute este script novamente.

:sync_server
if not exist "..\central-server\public\downloads" mkdir "..\central-server\public\downloads" >nul 2>nul
if exist "build\bin\uepg-net-tools.exe" copy /y "build\bin\uepg-net-tools.exe" "..\central-server\public\downloads\uepg-net-tools.exe" >nul 2>nul
if exist "build\bin\UEPG-Net-Tools-Instalador.exe" (
    copy /y "build\bin\UEPG-Net-Tools-Instalador.exe" "..\central-server\public\downloads\UEPG-Net-Tools-Instalador.exe" >nul 2>nul
) else if exist "build\bin\uepg-net-tools.exe" (
    copy /y "build\bin\uepg-net-tools.exe" "..\central-server\public\downloads\UEPG-Net-Tools-Instalador.exe" >nul 2>nul
)

:end
echo.
echo Abrindo a pasta build\bin no Windows Explorer...
start "" "build\bin"

pause