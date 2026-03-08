package dialog

import (
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"winSettingsGui/internal/config"
)

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	getModuleHandle = kernel32.NewProc("GetModuleHandleW")

	createWindowExW  = user32.NewProc("CreateWindowExW")
	defWindowProcW   = user32.NewProc("DefWindowProcW")
	registerClassExW = user32.NewProc("RegisterClassExW")
	getMessageW      = user32.NewProc("GetMessageW")
	translateMessage = user32.NewProc("TranslateMessage")
	dispatchMessageW = user32.NewProc("DispatchMessageW")
	postQuitMessage  = user32.NewProc("PostQuitMessage")
	destroyWindow    = user32.NewProc("DestroyWindow")
	sendMessageW     = user32.NewProc("SendMessageW")
	getWindowTextW   = user32.NewProc("GetWindowTextW")
	setFocus         = user32.NewProc("SetFocus")
	showWindow       = user32.NewProc("ShowWindow")
	updateWindow     = user32.NewProc("UpdateWindow")
	enableWindow     = user32.NewProc("EnableWindow")
)

const (
	wsOverlappedWindow = 0x00CF0000
	wsChild            = 0x40000000
	wsVisible          = 0x10000000
	wsBorder           = 0x00800000
	wsTabStop          = 0x00010000
	esAutoHScroll      = 0x0080
	bsPushButton       = 0x00000000
	swShow             = 5
	wmCommand          = 0x0111
	wmClose            = 0x0010
	wmDestroy          = 0x0002
	wmCreate           = 0x0001
	wmSetFont          = 0x0030
	bnClicked          = 0
	idOK               = 1
	idCancel           = 2
)

type settingsDialog struct {
	hwnd       uintptr
	editDisp   uintptr
	editSleep  uintptr
	editHib    uintptr
	cfg        config.Config
	result     config.Config
	ok         bool
}

var dlg *settingsDialog

func ShowSettings(cfg config.Config) (config.Config, bool) {
	dlg = &settingsDialog{cfg: cfg}
	dlg.run()
	return dlg.result, dlg.ok
}

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func intsToString(vals []int) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ", ")
}

func parseInts(s string) ([]int, bool) {
	parts := strings.Split(s, ",")
	var result []int
	seen := make(map[int]bool)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil || v <= 0 {
			return nil, false
		}
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	if len(result) == 0 {
		return nil, false
	}
	sort.Ints(result)
	return result, true
}

type wndClassExW struct {
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

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

func settingsWndProc(hwnd, uMsg, wParam, lParam uintptr) uintptr {
	switch uMsg {
	case wmCreate:
		dlg.hwnd = hwnd

		hFont := getDefaultFont()

		createLabel(hwnd, "Таймауты экрана (мин, через запятую):", 10, 15, 380, 20, hFont)
		dlg.editDisp = createEdit(hwnd, intsToString(dlg.cfg.DisplayTimeouts), 10, 38, 380, 24, hFont)

		createLabel(hwnd, "Таймауты сна (мин, через запятую):", 10, 72, 380, 20, hFont)
		dlg.editSleep = createEdit(hwnd, intsToString(dlg.cfg.SleepTimeouts), 10, 95, 380, 24, hFont)

		createLabel(hwnd, "Таймауты гибернации (мин, через запятую):", 10, 129, 380, 20, hFont)
		dlg.editHib = createEdit(hwnd, intsToString(dlg.cfg.HibernateTimeouts), 10, 152, 380, 24, hFont)

		createButton(hwnd, "OK", 210, 190, 85, 30, idOK, hFont)
		createButton(hwnd, "Отмена", 305, 190, 85, 30, idCancel, hFont)

		return 0

	case wmCommand:
		id := int(wParam & 0xFFFF)
		notif := int((wParam >> 16) & 0xFFFF)
		if notif == bnClicked {
			switch id {
			case idOK:
				dispText := getWindowText(dlg.editDisp)
				sleepText := getWindowText(dlg.editSleep)
				hibText := getWindowText(dlg.editHib)

				dispVals, ok1 := parseInts(dispText)
				sleepVals, ok2 := parseInts(sleepText)
				hibVals, ok3 := parseInts(hibText)

				if !ok1 || !ok2 || !ok3 {
					errTitle, _ := syscall.UTF16PtrFromString("Ошибка")
					errText, _ := syscall.UTF16PtrFromString("Введите положительные целые числа через запятую")
					messageBoxW.Call(hwnd, uintptr(unsafe.Pointer(errText)), uintptr(unsafe.Pointer(errTitle)), 0x10)
					return 0
				}

				dlg.result = config.Config{
					DisplayTimeouts:   dispVals,
					SleepTimeouts:     sleepVals,
					HibernateTimeouts: hibVals,
				}
				dlg.ok = true
				destroyWindow.Call(hwnd)

			case idCancel:
				destroyWindow.Call(hwnd)
			}
		}
		return 0

	case wmClose:
		destroyWindow.Call(hwnd)
		return 0

	case wmDestroy:
		postQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := defWindowProcW.Call(hwnd, uMsg, wParam, lParam)
	return ret
}

func (d *settingsDialog) run() {
	hInst, _, _ := getModuleHandle.Call(0)

	className := utf16Ptr("WinSettingsGuiSettings")

	wc := wndClassExW{
		Size:       uint32(unsafe.Sizeof(wndClassExW{})),
		WndProc:    syscall.NewCallback(settingsWndProc),
		Instance:   hInst,
		ClassName:  className,
		Background: 16,
	}

	registerClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr("Настройки — WinSettingsGui"))),
		uintptr(wsOverlappedWindow),
		200, 200, 420, 270,
		0, 0, hInst, 0,
	)

	showWindow.Call(hwnd, swShow)
	updateWindow.Call(hwnd)

	var m msg
	for {
		ret, _, _ := getMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if ret == 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&m)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func createLabel(parent uintptr, text string, x, y, w, h int, hFont uintptr) uintptr {
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr("STATIC"))),
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		uintptr(wsChild|wsVisible),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, 0, 0, 0,
	)
	sendMessageW.Call(hwnd, wmSetFont, hFont, 1)
	return hwnd
}

func createEdit(parent uintptr, text string, x, y, w, h int, hFont uintptr) uintptr {
	hwnd, _, _ := createWindowExW.Call(
		0x200,
		uintptr(unsafe.Pointer(utf16Ptr("EDIT"))),
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		uintptr(wsChild|wsVisible|wsBorder|wsTabStop|esAutoHScroll),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, 0, 0, 0,
	)
	sendMessageW.Call(hwnd, wmSetFont, hFont, 1)
	return hwnd
}

func createButton(parent uintptr, text string, x, y, w, h int, id int, hFont uintptr) uintptr {
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		uintptr(wsChild|wsVisible|wsTabStop|bsPushButton),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, uintptr(id), 0, 0,
	)
	sendMessageW.Call(hwnd, wmSetFont, hFont, 1)
	return hwnd
}

func getWindowText(hwnd uintptr) string {
	buf := make([]uint16, 256)
	getWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 256)
	return syscall.UTF16ToString(buf)
}

var (
	gdi32          = syscall.NewLazyDLL("gdi32.dll")
	getStockObject = gdi32.NewProc("GetStockObject")
)

func getDefaultFont() uintptr {
	const defaultGuiFont = 17
	hFont, _, _ := getStockObject.Call(defaultGuiFont)
	return hFont
}
