; ==============================================================================
; UEPG-Net-Tools - Script de Instalação NSIS
; Universidade Estadual de Ponta Grossa (UEPG)
; ==============================================================================

!include "MUI2.nsh"
!include "FileFunc.nsh"

; Definições Gerais
!define PRODUCT_NAME "UEPG Net Tools"
!define PRODUCT_VERSION "1.0.0"
!define PRODUCT_PUBLISHER "UEPG - NTI"
!define PRODUCT_DIR_REGKEY "Software\Microsoft\Windows\CurrentVersion\App Paths\uepg-net-tools.exe"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
!define PRODUCT_UNINST_ROOT_KEY "HKLM"

; Configurações do Instalador
Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "..\bin\UEPG-Net-Tools-Instalador.exe"
InstallDir "$PROGRAMFILES64\UEPG-Net-Tools"
InstallDirRegKey HKLM "${PRODUCT_DIR_REGKEY}" ""
RequestExecutionLevel admin

; Configurações da Interface Moderna (MUI2)
!define MUI_ABORTWARNING
!define MUI_ICON "icon.ico"
!define MUI_UNICON "icon.ico"

; Páginas do Instalador
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES

!define MUI_FINISHPAGE_RUN "$INSTDIR\uepg-net-tools.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Iniciar o UEPG Net Tools agora"
!insertmacro MUI_PAGE_FINISH

; Páginas do Desinstalador
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Idioma
!insertmacro MUI_LANGUAGE "PortugueseBR"

; Seção de Instalação Principal (Obrigatória)
Section "!Arquivos do Programa (Obrigatório)" SecMain
  SectionIn RO
  SetOutPath "$INSTDIR"
  SetOverwrite on

  ; Arquivos do Programa
  File "..\bin\uepg-net-tools.exe"
  File "icon.ico"

  ; Preserva config.json se já existir no cliente para não sobrescrever configurações locais
  IfFileExists "$INSTDIR\config.json" skip_config
    File "..\..\config.json"
  skip_config:

  ; Cria Atalhos no Menu Iniciar
  CreateDirectory "$SMPROGRAMS\UEPG Net Tools"
  CreateShortcut "$SMPROGRAMS\UEPG Net Tools\UEPG Net Tools.lnk" "$INSTDIR\uepg-net-tools.exe" "" "$INSTDIR\icon.ico" 0
  CreateShortcut "$SMPROGRAMS\UEPG Net Tools\Desinstalar.lnk" "$INSTDIR\uninst.exe"

  ; Grava Desinstalador
  WriteUninstaller "$INSTDIR\uninst.exe"

  ; Registros do Windows para Desinstalação (Painel de Controle / Configurações)
  WriteRegStr HKLM "${PRODUCT_DIR_REGKEY}" "" "$INSTDIR\uepg-net-tools.exe"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\uninst.exe"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\icon.ico"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegDWORD ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "NoModify" 1
  WriteRegDWORD ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "NoRepair" 1
SectionEnd

; Seção Opcional 1: Atalho no Desktop
Section "Criar atalho na Área de Trabalho" SecDesktop
  CreateShortcut "$DESKTOP\UEPG Net Tools.lnk" "$INSTDIR\uepg-net-tools.exe" "" "$INSTDIR\icon.ico" 0
SectionEnd

; Seção Opcional 2: Iniciar junto com o computador (Windows Startup)
Section "Iniciar junto com o computador (Windows Startup)" SecStartup
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "UEPGNetTools" '"$INSTDIR\uepg-net-tools.exe"'
SectionEnd

; Descrições dos Componentes para a Interface
LangString DESC_SecMain ${LANG_PORTUGUESEBR} "Arquivos essenciais do executável e configurações da sonda."
LangString DESC_SecDesktop ${LANG_PORTUGUESEBR} "Cria um atalho rápido na sua Área de Trabalho para fácil acesso."
LangString DESC_SecStartup ${LANG_PORTUGUESEBR} "Inicia o monitoramento automaticamente sempre que o computador for ligado."

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecMain} $(DESC_SecMain)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} $(DESC_SecDesktop)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecStartup} $(DESC_SecStartup)
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; Seção do Desinstalador
Section Uninstall
  ; Remove Atalhos
  Delete "$DESKTOP\UEPG Net Tools.lnk"
  Delete "$SMPROGRAMS\UEPG Net Tools\UEPG Net Tools.lnk"
  Delete "$SMPROGRAMS\UEPG Net Tools\Desinstalar.lnk"
  RMDir "$SMPROGRAMS\UEPG Net Tools"

  ; Remove Registro de Inicialização
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "UEPGNetTools"

  ; Remove Arquivos
  Delete "$INSTDIR\uepg-net-tools.exe"
  Delete "$INSTDIR\icon.ico"
  Delete "$INSTDIR\uninst.exe"
  Delete "$INSTDIR\measurement_profile.json"

  ; Remove Diretório de Instalação se vazio
  RMDir "$INSTDIR"

  ; Limpa Registros
  DeleteRegKey HKLM "${PRODUCT_DIR_REGKEY}"
  DeleteRegKey ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}"
SectionEnd
