//go:build windows && !cli

package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	NIM_ADD          = 0x00000000
	NIM_MODIFY       = 0x00000001
	NIM_DELETE       = 0x00000002
	NIF_MESSAGE      = 0x00000001
	NIF_ICON         = 0x00000002
	NIF_TIP          = 0x00000004
	NIF_INFO         = 0x00000010
	NIIF_INFO        = 0x00000001

	WM_USER          = 0x0400
	WM_TRAYICON      = WM_USER + 100
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONUP     = 0x0205
	WM_COMMAND       = 0x0111
	WM_DESTROY       = 0x0002

	MF_STRING        = 0x00000000
	MF_SEPARATOR     = 0x00000800
	TPM_BOTTOMALIGN  = 0x0020
	TPM_LEFTALIGN    = 0x0000

	ID_TRAY_OPEN     = 2001
	ID_TRAY_TEST     = 2002
	ID_TRAY_UPDATE   = 2003
	ID_TRAY_EXIT     = 2004
)

type NOTIFYICONDATAW struct {
	CbSize            uint32
	HWnd              uintptr
	UID               uint32
	UFlags            uint32
	UCallbackMessage  uint32
	HIcon             uintptr
	SzTip             [128]uint16
	DwState           uint32
	DwStateMask       uint32
	SzInfo            [256]uint16
	UTimeoutOrVersion uint32
	SzInfoTitle       [64]uint16
	DwInfoFlags       uint32
}

type POINT struct {
	X int32
	Y int32
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	pRegisterClassExW     = user32.NewProc("RegisterClassExW")
	pCreateWindowExW      = user32.NewProc("CreateWindowExW")
	pDefWindowProcW       = user32.NewProc("DefWindowProcW")
	pGetMessageW          = user32.NewProc("GetMessageW")
	pTranslateMessage     = user32.NewProc("TranslateMessage")
	pDispatchMessageW     = user32.NewProc("DispatchMessageW")
	pPostMessageW         = user32.NewProc("PostMessageW")
	pDestroyWindow        = user32.NewProc("DestroyWindow")
	pCreatePopupMenu      = user32.NewProc("CreatePopupMenu")
	pAppendMenuW          = user32.NewProc("AppendMenuW")
	pTrackPopupMenu       = user32.NewProc("TrackPopupMenu")
	pDestroyMenu          = user32.NewProc("DestroyMenu")
	pGetCursorPos         = user32.NewProc("GetCursorPos")
	pSetForegroundWindow  = user32.NewProc("SetForegroundWindow")
	pLoadIconW            = user32.NewProc("LoadIconW")
	pExtractIconExW       = shell32.NewProc("ExtractIconExW")
	pShell_NotifyIconW    = shell32.NewProc("Shell_NotifyIconW")

	trayHwnd      uintptr
	trayNID       NOTIFYICONDATAW
	trayApp       *App
	trayCtx       context.Context
	trayOnce      sync.Once
	balloonShown  bool
	trayMu        sync.Mutex
)

func utf16FromString(s string) []uint16 {
	res, _ := syscall.UTF16FromString(s)
	return res
}

func trayWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_TRAYICON:
		switch lParam {
		case WM_LBUTTONUP, WM_LBUTTONDBLCLK:
			// Clique com botão esquerdo ou duplo clique: reabre a janela
			if trayCtx != nil {
				runtime.WindowShow(trayCtx)
				runtime.WindowUnminimise(trayCtx)
			}
			return 0

		case WM_RBUTTONUP:
			// Clique com botão direito: exibe menu de contexto na bandeja
			var pt POINT
			pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
			pSetForegroundWindow.Call(hwnd)

			hMenu, _, _ := pCreatePopupMenu.Call()
			pAppendMenuW.Call(hMenu, MF_STRING, ID_TRAY_OPEN, uintptr(unsafe.Pointer(&utf16FromString("Abrir UEPG Net Tools")[0])))
			pAppendMenuW.Call(hMenu, MF_STRING, ID_TRAY_TEST, uintptr(unsafe.Pointer(&utf16FromString("Executar Teste Agora")[0])))
			pAppendMenuW.Call(hMenu, MF_STRING, ID_TRAY_UPDATE, uintptr(unsafe.Pointer(&utf16FromString("Verificar Atualizações")[0])))
			pAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
			pAppendMenuW.Call(hMenu, MF_STRING, ID_TRAY_EXIT, uintptr(unsafe.Pointer(&utf16FromString("Sair")[0])))

			pTrackPopupMenu.Call(hMenu, TPM_BOTTOMALIGN|TPM_LEFTALIGN, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
			pDestroyMenu.Call(hMenu)
			return 0
		}

	case WM_COMMAND:
		cmdId := wParam & 0xFFFF
		switch cmdId {
		case ID_TRAY_OPEN:
			if trayCtx != nil {
				runtime.WindowShow(trayCtx)
				runtime.WindowUnminimise(trayCtx)
			}
		case ID_TRAY_TEST:
			if trayApp != nil {
				go trayApp.ExecuteTest("Manual-Tray")
			}
		case ID_TRAY_UPDATE:
			if trayApp != nil {
				go trayApp.CheckUpdateNow()
			}
		case ID_TRAY_EXIT:
			// Encerramento limpo
			cleanupTray()
			if trayCtx != nil {
				runtime.Quit(trayCtx)
			} else {
				os.Exit(0)
			}
		}
		return 0

	case WM_DESTROY:
		return 0
	}

	ret, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

// initSystemTray inicializa o ícone de bandeja no Windows
func initSystemTray(ctx context.Context, app *App) {
	trayOnce.Do(func() {
		trayCtx = ctx
		trayApp = app

		go func() {
			hInst, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
			className := utf16FromString("UEPGNetToolsTrayClass")

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

			wc := WNDCLASSEXW{
				Size:      uint32(unsafe.Sizeof(WNDCLASSEXW{})),
				WndProc:   syscall.NewCallback(trayWndProc),
				Instance:  hInst,
				ClassName: &className[0],
			}
			pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

			hwnd, _, _ := pCreateWindowExW.Call(
				0,
				uintptr(unsafe.Pointer(&className[0])),
				uintptr(unsafe.Pointer(&utf16FromString("UEPGNetToolsTrayWindow")[0])),
				0, 0, 0, 0, 0,
				0, 0, hInst, 0,
			)
			trayHwnd = hwnd

			// Extrai ícone do próprio executável compilado
			var hIconSmall uintptr
			execPath, err := os.Executable()
			if err == nil {
				execPath, _ = filepath.EvalSymlinks(execPath)
				pExtractIconExW.Call(
					uintptr(unsafe.Pointer(&utf16FromString(execPath)[0])),
					0, 0,
					uintptr(unsafe.Pointer(&hIconSmall)),
					1,
				)
			}
			if hIconSmall == 0 {
				hIconSmall, _, _ = pLoadIconW.Call(0, uintptr(32512)) // IDI_APPLICATION
			}

			trayNID = NOTIFYICONDATAW{
				CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATAW{})),
				HWnd:             trayHwnd,
				UID:              1001,
				UFlags:           NIF_ICON | NIF_MESSAGE | NIF_TIP,
				UCallbackMessage: WM_TRAYICON,
				HIcon:            hIconSmall,
			}

			tip := utf16FromString("UEPG Net Tools • Monitor de Rede Ativo")
			copy(trayNID.SzTip[:], tip)

			pShell_NotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&trayNID)))
			log.Println("[TRAY] Ícone da bandeja do sistema (System Tray) ativo com sucesso.")

			// Message pump do Windows para a janela oculta do tray
			var msg struct {
				Hwnd    uintptr
				Message uint32
				WParam  uintptr
				LParam  uintptr
				Time    uint32
				Pt      POINT
			}
			for {
				ret, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
				if ret == 0 || ret == ^uintptr(0) {
					break
				}
				pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
				pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
			}
		}()
	})
}

// showTrayBalloon exibe uma notificação suave quando o usuário fecha a janela pela primeira vez
func showTrayBalloon(title, message string) {
	trayMu.Lock()
	defer trayMu.Unlock()

	if trayHwnd == 0 || balloonShown {
		return
	}
	balloonShown = true

	trayNID.UFlags |= NIF_INFO
	trayNID.DwInfoFlags = NIIF_INFO

	titleU := utf16FromString(title)
	copy(trayNID.SzInfoTitle[:], titleU)

	msgU := utf16FromString(message)
	copy(trayNID.SzInfo[:], msgU)

	pShell_NotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&trayNID)))
}

// cleanupTray remove o ícone da bandeja ao encerrar o aplicativo
func cleanupTray() {
	if trayHwnd != 0 {
		pShell_NotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&trayNID)))
		pDestroyWindow.Call(trayHwnd)
		trayHwnd = 0
		log.Println("[TRAY] Ícone da bandeja removido com sucesso.")
	}
}
