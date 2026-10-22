package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

//go:embed payload/*
var payloadFS embed.FS

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_BORDER           = 0x00800000
	WS_TABSTOP          = 0x00010000

	BS_DEFPUSHBUTTON = 0x00000001
	BS_AUTOCHECKBOX  = 0x00000003
	BM_GETCHECK      = 0x00F0
	BM_SETCHECK      = 0x00F1
	BST_CHECKED      = 0x0001

	ES_AUTOHSCROLL = 0x0080
	WM_SETFONT     = 0x0030
	WM_COMMAND     = 0x0111
	WM_DESTROY     = 0x0002
	WM_CLOSE       = 0x0010

	COLOR_WINDOW   = 5
	COLOR_BTNFACE  = 15
	SW_SHOWNORMAL  = 1
	MB_YESNO       = 0x00000004
	MB_ICONQUESTION = 0x00000020
	MB_ICONERROR    = 0x00000010
	MB_OK          = 0x00000000
	IDYES          = 6

	ID_BTN_BROWSE  = 101
	ID_BTN_INSTALL = 102
	ID_BTN_CANCEL  = 103
	ID_CHK_DESKTOP = 104
	ID_CHK_STARTUP = 105
	ID_CHK_STARTMENU = 106
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	pRegisterClassExW = user32.NewProc("RegisterClassExW")
	pCreateWindowExW  = user32.NewProc("CreateWindowExW")
	pDefWindowProcW   = user32.NewProc("DefWindowProcW")
	pGetMessageW      = user32.NewProc("GetMessageW")
	pTranslateMessage = user32.NewProc("TranslateMessage")
	pDispatchMessageW = user32.NewProc("DispatchMessageW")
	pSendMessageW     = user32.NewProc("SendMessageW")
	pPostQuitMessage  = user32.NewProc("PostQuitMessage")
	pMessageBoxW      = user32.NewProc("MessageBoxW")
	pGetWindowTextW   = user32.NewProc("GetWindowTextW")
	pSetWindowTextW   = user32.NewProc("SetWindowTextW")
	pEnableWindow     = user32.NewProc("EnableWindow")

	pCreateFontW       = gdi32.NewProc("CreateFontW")
	pSHBrowseForFolder = shell32.NewProc("SHBrowseForFolderW")
	pSHGetPathFromIDList = shell32.NewProc("SHGetPathFromIDListW")

	hwndMain       uintptr
	hwndPathEdit   uintptr
	hwndServerEdit uintptr
	hwndChkDesktop uintptr
	hwndChkStartup uintptr
	hwndChkMenu    uintptr
	hwndBtnInstall uintptr
	hwndBtnCancel  uintptr
	hwndBtnBrowse  uintptr
	hwndStatus     uintptr
	defaultFont    uintptr
	boldFont       uintptr
)

type WNDCLASSEXW struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type BROWSEINFOW struct {
	HwndOwner      uintptr
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}

func getDefaultInstallDir() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		return filepath.Join(localAppData, "Programs", "UEPG-Net-Tools")
	}
	programFiles := os.Getenv("ProgramFiles")
	if programFiles != "" {
		return filepath.Join(programFiles, "UEPG-Net-Tools")
	}
	return "C:\\UEPG-Net-Tools"
}

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func utf16ToString(p []uint16) string {
	return syscall.UTF16ToString(p)
}

func initFonts() {
	boldFont, _, _ = pCreateFontW.Call(
		18, 0, 0, 0, 700, 0, 0, 0, 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(utf16Ptr("Segoe UI"))),
	)
	defaultFont, _, _ = pCreateFontW.Call(
		15, 0, 0, 0, 400, 0, 0, 0, 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(utf16Ptr("Segoe UI"))),
	)
}

func createControl(className, text string, style uint32, x, y, w, h int32, id uintptr, font uintptr) uintptr {
	hInst, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	hwnd, _, _ := pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr(className))),
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		uintptr(style|WS_CHILD|WS_VISIBLE),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		hwndMain,
		id,
		hInst,
		0,
	)
	if font != 0 {
		pSendMessageW.Call(hwnd, WM_SETFONT, font, 1)
	}
	return hwnd
}

func browseForFolder() string {
	title := utf16Ptr("Selecione a pasta de instalação do UEPG Net Tools:")
	displayName := make([]uint16, 260)
	bi := BROWSEINFOW{
		HwndOwner:      hwndMain,
		PszDisplayName: &displayName[0],
		LpszTitle:      title,
		UlFlags:        0x00000040 | 0x00000001, // BIF_NEWDIALOGSTYLE | BIF_RETURNONLYFSDIRS
	}
	pidl, _, _ := pSHBrowseForFolder.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl != 0 {
		pathBuf := make([]uint16, 260)
		pSHGetPathFromIDList.Call(pidl, uintptr(unsafe.Pointer(&pathBuf[0])))
		return utf16ToString(pathBuf)
	}
	return ""
}

func getControlText(hwnd uintptr) string {
	buf := make([]uint16, 512)
	pGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 512)
	return strings.TrimSpace(utf16ToString(buf))
}

func isChecked(hwnd uintptr) bool {
	res, _, _ := pSendMessageW.Call(hwnd, BM_GETCHECK, 0, 0)
	return res == BST_CHECKED
}

func createShortcut(targetPath, shortcutPath, iconPath string) error {
	dir := filepath.Dir(shortcutPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	workDir := filepath.Dir(targetPath)
	psCmd := fmt.Sprintf(
		`$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut('%s'); $s.TargetPath = '%s'; $s.WorkingDirectory = '%s'; if ('%s' -ne '') { $s.IconLocation = '%s,0' }; $s.Save()`,
		strings.ReplaceAll(shortcutPath, `'`, `''`),
		strings.ReplaceAll(targetPath, `'`, `''`),
		strings.ReplaceAll(workDir, `'`, `''`),
		strings.ReplaceAll(iconPath, `'`, `''`),
		strings.ReplaceAll(iconPath, `'`, `''`),
	)
	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", psCmd)
	return cmd.Run()
}

func setWindowsStartup(targetPath string, enable bool) error {
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	valName := "UEPGNetTools"
	if enable {
		cmd := exec.Command("reg", "add", key, "/v", valName, "/t", "REG_SZ", "/d", fmt.Sprintf(`"%s"`, targetPath), "/f")
		return cmd.Run()
	} else {
		cmd := exec.Command("reg", "delete", key, "/v", valName, "/f")
		_ = cmd.Run()
		return nil
	}
}

func extractFile(embedPath, destPath string) error {
	in, err := payloadFS.Open(embedPath)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func performInstallation() {
	destDir := getControlText(hwndPathEdit)
	if destDir == "" {
		pMessageBoxW.Call(hwndMain, uintptr(unsafe.Pointer(utf16Ptr("Por favor, selecione uma pasta de instalação válida."))), uintptr(unsafe.Pointer(utf16Ptr("Atenção"))), MB_OK|MB_ICONERROR)
		return
	}

	chkDesktop := isChecked(hwndChkDesktop)
	chkStartup := isChecked(hwndChkStartup)
	chkMenu := isChecked(hwndChkMenu)

	pEnableWindow.Call(hwndBtnInstall, 0)
	pEnableWindow.Call(hwndBtnCancel, 0)
	pSetWindowTextW.Call(hwndStatus, uintptr(unsafe.Pointer(utf16Ptr("Instalando arquivos do UEPG Net Tools..."))))

	if err := os.MkdirAll(destDir, 0755); err != nil {
		pMessageBoxW.Call(hwndMain, uintptr(unsafe.Pointer(utf16Ptr("Falha ao criar pasta de destino: "+err.Error()))), uintptr(unsafe.Pointer(utf16Ptr("Erro de Permissão"))), MB_OK|MB_ICONERROR)
		pEnableWindow.Call(hwndBtnInstall, 1)
		pEnableWindow.Call(hwndBtnCancel, 1)
		pSetWindowTextW.Call(hwndStatus, uintptr(unsafe.Pointer(utf16Ptr("Pronto para instalar"))))
		return
	}

	exePath := filepath.Join(destDir, "uepg-net-tools.exe")
	configPath := filepath.Join(destDir, "config.json")
	iconPath := filepath.Join(destDir, "icon.ico")

	_ = extractFile("payload/uepg-net-tools.exe", exePath)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		_ = extractFile("payload/config.json", configPath)
	}
	_ = extractFile("payload/icon.ico", iconPath)

	// Atualiza server_url no config.json com o valor informado pelo usuário
	serverURL := getControlText(hwndServerEdit)
	if serverURL != "" {
		if data, err := os.ReadFile(configPath); err == nil {
			var cfgMap map[string]interface{}
			if err := json.Unmarshal(data, &cfgMap); err == nil {
				cfgMap["server_url"] = strings.TrimRight(serverURL, "/")
				if updatedData, err := json.MarshalIndent(cfgMap, "", "  "); err == nil {
					_ = os.WriteFile(configPath, updatedData, 0644)
				}
			}
		}
	}

	userProfile := os.Getenv("USERPROFILE")
	appData := os.Getenv("APPDATA")

	if chkDesktop && userProfile != "" {
		desktopShortcut := filepath.Join(userProfile, "Desktop", "UEPG Net Tools.lnk")
		_ = createShortcut(exePath, desktopShortcut, iconPath)
	}

	if chkMenu && appData != "" {
		menuShortcut := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "UEPG Net Tools", "UEPG Net Tools.lnk")
		_ = createShortcut(exePath, menuShortcut, iconPath)
	}

	if chkStartup {
		_ = setWindowsStartup(exePath, true)
	}

	pSetWindowTextW.Call(hwndStatus, uintptr(unsafe.Pointer(utf16Ptr("Instalação concluída com sucesso!"))))

	msg := fmt.Sprintf("O UEPG Net Tools foi instalado com sucesso em:\n\n%s\n\nDeseja iniciar o aplicativo agora?", destDir)
	res, _, _ := pMessageBoxW.Call(hwndMain, uintptr(unsafe.Pointer(utf16Ptr(msg))), uintptr(unsafe.Pointer(utf16Ptr("Instalação Concluída"))), MB_YESNO|MB_ICONQUESTION)

	if res == IDYES {
		cmd := exec.Command(exePath)
		_ = cmd.Start()
	}

	pPostQuitMessage.Call(0)
}

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		controlId := wParam & 0xFFFF
		switch controlId {
		case ID_BTN_BROWSE:
			selected := browseForFolder()
			if selected != "" {
				pSetWindowTextW.Call(hwndPathEdit, uintptr(unsafe.Pointer(utf16Ptr(selected))))
			}
		case ID_BTN_INSTALL:
			performInstallation()
		case ID_BTN_CANCEL:
			pPostQuitMessage.Call(0)
		}
		return 0
	case WM_CLOSE:
		pPostQuitMessage.Call(0)
		return 0
	case WM_DESTROY:
		pPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func main() {
	runtime.LockOSThread()

	initFonts()
	hInst, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	className := utf16Ptr("UEPGNetToolsInstaller")

	wc := WNDCLASSEXW{
		Size:       uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:      0,
		WndProc:    syscall.NewCallback(wndProc),
		Instance:   hInst,
		Background: uintptr(COLOR_BTNFACE + 1),
		ClassName:  className,
	}
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	winW := int32(520)
	winH := int32(450)

	scrW, _, _ := user32.NewProc("GetSystemMetrics").Call(0)
	scrH, _, _ := user32.NewProc("GetSystemMetrics").Call(1)
	posX := (int32(scrW) - winW) / 2
	posY := (int32(scrH) - winH) / 2

	hwndMain, _, _ = pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr("Assistente de Instalação • UEPG Net Tools"))),
		uintptr(WS_OVERLAPPEDWINDOW&^0x00050000|WS_VISIBLE), // sem resize ou maximize
		uintptr(posX), uintptr(posY), uintptr(winW), uintptr(winH),
		0, 0, hInst, 0,
	)

	// Título e Subtítulo
	createControl("STATIC", "Instalar UEPG Net Tools", 0, 24, 16, 460, 24, 0, boldFont)
	createControl("STATIC", "Configure as opções de instalação da sonda de rede neste computador:", 0, 24, 42, 460, 20, 0, defaultFont)

	// Seção 1: Pasta de Instalação
	createControl("STATIC", "Pasta de Destino:", 0, 24, 72, 460, 18, 0, defaultFont)
	hwndPathEdit = createControl("EDIT", getDefaultInstallDir(), WS_BORDER|ES_AUTOHSCROLL|WS_TABSTOP, 24, 92, 360, 26, 0, defaultFont)
	hwndBtnBrowse = createControl("BUTTON", "Procurar...", WS_TABSTOP, 394, 92, 90, 26, ID_BTN_BROWSE, defaultFont)

	// Seção 2: URL do Servidor Central
	createControl("STATIC", "URL do Servidor Central (Backend):", 0, 24, 126, 460, 18, 0, defaultFont)
	hwndServerEdit = createControl("EDIT", "https://akvorado.acon.dev.uepg.br", WS_BORDER|ES_AUTOHSCROLL|WS_TABSTOP, 24, 146, 460, 26, 0, defaultFont)

	// Seção 3: Opções Adicionais (Atalhos e Inicialização)
	createControl("STATIC", "Opções Adicionais:", 0, 24, 182, 460, 18, 0, defaultFont)

	hwndChkDesktop = createControl("BUTTON", "Criar atalho na Área de Trabalho (Desktop)", BS_AUTOCHECKBOX|WS_TABSTOP, 24, 204, 460, 22, ID_CHK_DESKTOP, defaultFont)
	pSendMessageW.Call(hwndChkDesktop, BM_SETCHECK, BST_CHECKED, 0)

	hwndChkStartup = createControl("BUTTON", "Iniciar automaticamente com o computador (Windows Startup)", BS_AUTOCHECKBOX|WS_TABSTOP, 24, 230, 460, 22, ID_CHK_STARTUP, defaultFont)
	pSendMessageW.Call(hwndChkStartup, BM_SETCHECK, BST_CHECKED, 0)

	hwndChkMenu = createControl("BUTTON", "Criar atalhos no Menu Iniciar do Windows", BS_AUTOCHECKBOX|WS_TABSTOP, 24, 256, 460, 22, ID_CHK_STARTMENU, defaultFont)
	pSendMessageW.Call(hwndChkMenu, BM_SETCHECK, BST_CHECKED, 0)

	// Status text
	hwndStatus = createControl("STATIC", "Clique em 'Instalar' para iniciar a cópia e configuração.", 0, 24, 298, 460, 20, 0, defaultFont)

	// Linha divisória / Botões inferiores
	createControl("STATIC", "", WS_BORDER, 24, 328, 460, 1, 0, 0)

	hwndBtnCancel = createControl("BUTTON", "Cancelar", WS_TABSTOP, 384, 345, 100, 32, ID_BTN_CANCEL, defaultFont)
	hwndBtnInstall = createControl("BUTTON", "Instalar", BS_DEFPUSHBUTTON|WS_TABSTOP, 274, 345, 100, 32, ID_BTN_INSTALL, defaultFont)

	var msg MSG
	for {
		res, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(res) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
