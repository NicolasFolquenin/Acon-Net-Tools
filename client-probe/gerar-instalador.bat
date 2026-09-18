@echo off
chcp 65001 >nul
title UEPG-Net-Tools - Gerador de Executavel e Instalador
echo ==============================================================================
echo 📦 Gerador do UEPG-Net-Tools para Windows
echo ==============================================================================
echo.

set PATH=%PATH%;C:\Program Files\Go\bin;%USERPROFILE%\go\bin;C:\Program Files (x86)\NSIS;C:\Program Files\NSIS

cd /d "%~dp0"

where makensis >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [INFO] NSIS não encontrado no PATH.
    echo        Para gerar o instalador clássico (.exe com assistente de instalação),
    echo        basta instalar o NSIS pelo terminal: winget install NSIS.NSIS
    echo.
    echo [INFO] Gerando executável portátil oficial em build\bin\ ...
    wails build -clean -platform windows/amd64
) else (
    echo [OK] NSIS detectado!
    echo [INFO] Gerando executável e Instalador Oficial (.exe com assistente de instalação)...
    wails build -clean -platform windows/amd64 -nsis
)

echo.
echo ==============================================================================
echo ✔ Concluído! Os arquivos gerados estão na pasta: build\bin\
echo ==============================================================================
pause
