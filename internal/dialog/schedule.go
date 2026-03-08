package dialog

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"winSettingsGui/internal/config"
)

const (
	idSchList   = 300
	idSchAdd    = 301
	idSchEdit   = 302
	idSchDelete = 303
	idSchToggle = 304
	idSchClose  = 305

	mbYesNo  = 0x00000004
	mbIconQ  = 0x00000020
	idYes    = 6
)

type scheduleListDialog struct {
	hwnd    uintptr
	listbox uintptr
	jobs    []config.ScheduledJob
	cfg     config.Config
	changed bool
}

var schDlg *scheduleListDialog

func ShowSchedule(cfg config.Config) (config.Config, bool) {
	schDlg = &scheduleListDialog{
		cfg:  cfg,
		jobs: make([]config.ScheduledJob, len(cfg.ScheduledJobs)),
	}
	copy(schDlg.jobs, cfg.ScheduledJobs)
	schDlg.run()
	if schDlg.changed {
		schDlg.cfg.ScheduledJobs = schDlg.jobs
		return schDlg.cfg, true
	}
	return cfg, false
}

func scheduleWndProc(hwnd, uMsg, wParam, lParam uintptr) uintptr {
	switch uMsg {
	case wmCreate:
		schDlg.hwnd = hwnd
		hFont := getDefaultFont()

		createLabel(hwnd, "Запланированные задания:", 10, 10, 250, 20, hFont)
		schDlg.listbox = createListBox(hwnd, 10, 32, 460, 200, idSchList, hFont)
		refreshScheduleList()

		createButton(hwnd, "Добавить", 10, 242, 100, 30, idSchAdd, hFont)
		createButton(hwnd, "Редактировать", 120, 242, 120, 30, idSchEdit, hFont)
		createButton(hwnd, "Удалить", 250, 242, 100, 30, idSchDelete, hFont)
		createButton(hwnd, "Вкл/Выкл", 360, 242, 110, 30, idSchToggle, hFont)

		createButton(hwnd, "Закрыть", 380, 285, 90, 30, idSchClose, hFont)

		return 0

	case wmCommand:
		id := int(wParam & 0xFFFF)
		notif := int((wParam >> 16) & 0xFFFF)

		if notif == bnClicked {
			switch id {
			case idSchAdd:
				onScheduleAdd()
			case idSchEdit:
				onScheduleEdit()
			case idSchDelete:
				onScheduleDelete()
			case idSchToggle:
				onScheduleToggle()
			case idSchClose:
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

func (d *scheduleListDialog) run() {
	hInst, _, _ := getModuleHandle.Call(0)
	className := utf16Ptr("WinSettingsGuiSchedule")

	wc := wndClassExW{
		Size:       uint32(unsafe.Sizeof(wndClassExW{})),
		WndProc:    syscall.NewCallback(scheduleWndProc),
		Instance:   hInst,
		ClassName:  className,
		Background: 16,
	}
	registerClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr("Планирование — WinSettingsGui"))),
		uintptr(wsOverlappedWindow),
		180, 180, 500, 365,
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

func refreshScheduleList() {
	sendMessageW.Call(schDlg.listbox, lbResetCont, 0, 0)
	for _, j := range schDlg.jobs {
		listBoxAddStr(schDlg.listbox, formatJobLine(j))
	}
}

func formatJobLine(j config.ScheduledJob) string {
	status := "[ ]"
	if j.Active {
		status = "[✓]"
	}
	days := formatWeekdays(j.Weekdays)
	return fmt.Sprintf("%s %s  %s %02d:%02d  (%d)", status, j.Name, days, j.Hour, j.Minute, len(j.Actions))
}

func formatWeekdays(wd [7]bool) string {
	names := []string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
	var parts []string
	for i, on := range wd {
		if on {
			parts = append(parts, names[i])
		}
	}
	return strings.Join(parts, ",")
}

func getSelectedJobIndex() int {
	sel, _, _ := sendMessageW.Call(schDlg.listbox, lbGetCurSel, 0, 0)
	return int(sel)
}

func onScheduleAdd() {
	enableWindow.Call(schDlg.hwnd, 0)
	newJob, ok := ShowJobEdit(config.ScheduledJob{}, schDlg.cfg)
	enableWindow.Call(schDlg.hwnd, 1)

	if ok {
		schDlg.jobs = append(schDlg.jobs, newJob)
		schDlg.changed = true
		refreshScheduleList()
	}
}

func onScheduleEdit() {
	idx := getSelectedJobIndex()
	if idx < 0 || idx >= len(schDlg.jobs) {
		showError(schDlg.hwnd, "Выберите задание из списка")
		return
	}

	enableWindow.Call(schDlg.hwnd, 0)
	edited, ok := ShowJobEdit(schDlg.jobs[idx], schDlg.cfg)
	enableWindow.Call(schDlg.hwnd, 1)

	if ok {
		schDlg.jobs[idx] = edited
		schDlg.changed = true
		refreshScheduleList()
	}
}

func onScheduleDelete() {
	idx := getSelectedJobIndex()
	if idx < 0 || idx >= len(schDlg.jobs) {
		showError(schDlg.hwnd, "Выберите задание из списка")
		return
	}

	title, _ := syscall.UTF16PtrFromString("Подтверждение")
	text, _ := syscall.UTF16PtrFromString(fmt.Sprintf("Удалить задание \"%s\"?", schDlg.jobs[idx].Name))
	ret, _, _ := messageBoxW.Call(schDlg.hwnd, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), uintptr(mbYesNo|mbIconQ))

	if int(ret) == idYes {
		schDlg.jobs = append(schDlg.jobs[:idx], schDlg.jobs[idx+1:]...)
		schDlg.changed = true
		refreshScheduleList()
	}
}

func onScheduleToggle() {
	idx := getSelectedJobIndex()
	if idx < 0 || idx >= len(schDlg.jobs) {
		showError(schDlg.hwnd, "Выберите задание из списка")
		return
	}

	schDlg.jobs[idx].Active = !schDlg.jobs[idx].Active
	schDlg.changed = true
	refreshScheduleList()
	sendMessageW.Call(schDlg.listbox, lbSetCurSel, uintptr(idx), 0)
}
