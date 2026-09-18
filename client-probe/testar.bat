@echo off
chcp 65001 >nul
title UEPG-Net-Tools - Modo de Teste (Live Reload)
echo ==============================================================================
echo 🚀 Iniciando UEPG-Net-Tools em Modo Desenvolvimento (Live Reload)
echo.
echo • Qualquer mudança salva em HTML, CSS ou JS atualiza na hora sem reinstalar!
echo • Pressione Ctrl+C nesta janela quando quiser encerrar.
echo ==============================================================================
echo.

set PATH=%PATH%;C:\Program Files\Go\bin;%USERPROFILE%\go\bin

cd /d "%~dp0"
wails dev
pause
