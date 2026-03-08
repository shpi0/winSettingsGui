package dialog

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"winSettingsGui/internal/config"
)

const (
	cbsDropDownList = 0x0003
	cbAddString     = 0x0143
	cbGetCurSel     = 0x0147
	cbSetCurSel     = 0x014E
	cbDeleteString  = 0x0144
	cbGetCount      = 0x0146
	cbResetContent  = 0x014B
	cbnSelChange    = 1

	lbsNotify    = 0x0001
	lbAddString  = 0x0180
	lbDeleteStr  = 0x0182
	lbGetCurSel  = 0x0188
	lbSetCurSel  = 0x0186
	lbGetCount   = 0x018B
	lbResetCont  = 0x0184

	bsAutoCheckBox = 0x0003
	bmGetCheck     = 0x00F0
	bmSetCheck     = 0x00F1
	bstChecked     = 1

	wsVScroll = 0x00200000

	idJobName      = 200
	idWeekdayStart = 201
	idTimeHour     = 210
	idTimeMinute   = 211
	idActionList   = 220
	idComboType    = 230
	idComboSource  = 231
	idComboValue   = 232
	idActionAdd    = 240
	idActionRemove = 241
	idJobOK        = 250
	idJobCancel    = 251
)

type jobEditDialog struct {
	hwnd        uintptr
	editName    uintptr
	checkDays   [7]uintptr
	editHour    uintptr
	editMinute  uintptr
	actionList  uintptr
	comboType   uintptr
	comboSource uintptr
	comboValue  uintptr
	job         config.ScheduledJob
	cfg         config.Config
	result      config.ScheduledJob
	ok          bool
}

var jobDlg *jobEditDialog

func ShowJobEdit(job config.ScheduledJob, cfg config.Config) (config.ScheduledJob, bool) {
	jobDlg = &jobEditDialog{job: job, cfg: cfg}
	jobDlg.run()
	return jobDlg.result, jobDlg.ok
}

var dayNames = []string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}

func jobEditWndProc(hwnd, uMsg, wParam, lParam uintptr) uintptr {
	switch uMsg {
	case wmCreate:
		jobDlg.hwnd = hwnd
		hFont := getDefaultFont()

		createLabel(hwnd, "Название:", 10, 12, 70, 20, hFont)
		jobDlg.editName = createEdit(hwnd, jobDlg.job.Name, 85, 10, 390, 24, hFont)

		createLabel(hwnd, "Дни недели:", 10, 45, 80, 20, hFont)
		for i, name := range dayNames {
			jobDlg.checkDays[i] = createCheckbox(hwnd, name, 10+i*65, 67, 58, 22, idWeekdayStart+i, jobDlg.job.Weekdays[i], hFont)
		}

		createLabel(hwnd, "Время:", 10, 102, 50, 20, hFont)
		hourStr := fmt.Sprintf("%02d", jobDlg.job.Hour)
		minStr := fmt.Sprintf("%02d", jobDlg.job.Minute)
		jobDlg.editHour = createEdit(hwnd, hourStr, 65, 100, 40, 24, hFont)
		createLabel(hwnd, ":", 108, 102, 10, 20, hFont)
		jobDlg.editMinute = createEdit(hwnd, minStr, 120, 100, 40, 24, hFont)

		createLabel(hwnd, "Действия:", 10, 135, 80, 20, hFont)
		jobDlg.actionList = createListBox(hwnd, 10, 157, 465, 120, idActionList, hFont)
		for _, a := range jobDlg.job.Actions {
			addActionToListBox(a)
		}

		createLabel(hwnd, "Параметр:", 10, 290, 70, 20, hFont)
		jobDlg.comboType = createComboBox(hwnd, 80, 288, 120, 200, idComboType, hFont)
		comboAddStr(jobDlg.comboType, "Экран")
		comboAddStr(jobDlg.comboType, "Сон")
		comboAddStr(jobDlg.comboType, "Гибернация")
		sendMessageW.Call(jobDlg.comboType, cbSetCurSel, 0, 0)

		createLabel(hwnd, "Источник:", 210, 290, 65, 20, hFont)
		jobDlg.comboSource = createComboBox(hwnd, 278, 288, 100, 200, idComboSource, hFont)
		comboAddStr(jobDlg.comboSource, "Сеть (AC)")
		comboAddStr(jobDlg.comboSource, "Батарея (DC)")
		sendMessageW.Call(jobDlg.comboSource, cbSetCurSel, 0, 0)

		createLabel(hwnd, "Значение:", 388, 290, 65, 20, hFont)
		jobDlg.comboValue = createComboBox(hwnd, 388, 310, 87, 200, idComboValue, hFont)
		repopulateValueCombo()

		createButton(hwnd, "Добавить действие", 10, 340, 150, 28, idActionAdd, hFont)
		createButton(hwnd, "Удалить действие", 170, 340, 150, 28, idActionRemove, hFont)

		createButton(hwnd, "OK", 290, 385, 90, 30, idJobOK, hFont)
		createButton(hwnd, "Отмена", 390, 385, 85, 30, idJobCancel, hFont)

		return 0

	case wmCommand:
		id := int(wParam & 0xFFFF)
		notif := int((wParam >> 16) & 0xFFFF)

		if id == idComboType && notif == cbnSelChange {
			repopulateValueCombo()
			return 0
		}

		if notif == bnClicked {
			switch id {
			case idActionAdd:
				addNewAction()
			case idActionRemove:
				removeSelectedAction()
			case idJobOK:
				if validateAndSave() {
					destroyWindow.Call(hwnd)
				}
			case idJobCancel:
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

func (d *jobEditDialog) run() {
	hInst, _, _ := getModuleHandle.Call(0)
	className := utf16Ptr("WinSettingsGuiJobEdit")

	wc := wndClassExW{
		Size:       uint32(unsafe.Sizeof(wndClassExW{})),
		WndProc:    syscall.NewCallback(jobEditWndProc),
		Instance:   hInst,
		ClassName:  className,
		Background: 16,
	}
	registerClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	title := "Задание — WinSettingsGui"
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		uintptr(wsOverlappedWindow),
		150, 150, 510, 465,
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

func createComboBox(parent uintptr, x, y, w, dropH int, id int, hFont uintptr) uintptr {
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr("COMBOBOX"))),
		0,
		uintptr(wsChild|wsVisible|wsTabStop|wsVScroll|cbsDropDownList),
		uintptr(x), uintptr(y), uintptr(w), uintptr(dropH),
		parent, uintptr(id), 0, 0,
	)
	sendMessageW.Call(hwnd, wmSetFont, hFont, 1)
	return hwnd
}

func createCheckbox(parent uintptr, text string, x, y, w, h int, id int, checked bool, hFont uintptr) uintptr {
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr("BUTTON"))),
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		uintptr(wsChild|wsVisible|wsTabStop|bsAutoCheckBox),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, uintptr(id), 0, 0,
	)
	sendMessageW.Call(hwnd, wmSetFont, hFont, 1)
	if checked {
		sendMessageW.Call(hwnd, bmSetCheck, bstChecked, 0)
	}
	return hwnd
}

func createListBox(parent uintptr, x, y, w, h int, id int, hFont uintptr) uintptr {
	hwnd, _, _ := createWindowExW.Call(
		0x200,
		uintptr(unsafe.Pointer(utf16Ptr("LISTBOX"))),
		0,
		uintptr(wsChild|wsVisible|wsTabStop|wsVScroll|wsBorder|lbsNotify),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, uintptr(id), 0, 0,
	)
	sendMessageW.Call(hwnd, wmSetFont, hFont, 1)
	return hwnd
}

func comboAddStr(combo uintptr, text string) {
	p := utf16Ptr(text)
	sendMessageW.Call(combo, cbAddString, 0, uintptr(unsafe.Pointer(p)))
}

func listBoxAddStr(lb uintptr, text string) {
	p := utf16Ptr(text)
	sendMessageW.Call(lb, lbAddString, 0, uintptr(unsafe.Pointer(p)))
}

func repopulateValueCombo() {
	sendMessageW.Call(jobDlg.comboValue, cbResetContent, 0, 0)

	sel, _, _ := sendMessageW.Call(jobDlg.comboType, cbGetCurSel, 0, 0)
	var timeouts []int
	switch int(sel) {
	case 0:
		timeouts = jobDlg.cfg.DisplayTimeouts
	case 1:
		timeouts = jobDlg.cfg.SleepTimeouts
	case 2:
		timeouts = jobDlg.cfg.HibernateTimeouts
	}

	comboAddStr(jobDlg.comboValue, "Никогда")
	for _, t := range timeouts {
		comboAddStr(jobDlg.comboValue, formatMinutesRu(t))
	}
	sendMessageW.Call(jobDlg.comboValue, cbSetCurSel, 0, 0)
}

func addNewAction() {
	typeSel, _, _ := sendMessageW.Call(jobDlg.comboType, cbGetCurSel, 0, 0)
	srcSel, _, _ := sendMessageW.Call(jobDlg.comboSource, cbGetCurSel, 0, 0)
	valSel, _, _ := sendMessageW.Call(jobDlg.comboValue, cbGetCurSel, 0, 0)

	if int(typeSel) < 0 || int(srcSel) < 0 || int(valSel) < 0 {
		return
	}

	actionTypes := []config.ActionType{config.ActionDisplay, config.ActionSleep, config.ActionHibernate}
	sources := []config.SourceType{config.SourceAC, config.SourceDC}

	minutes := 0
	if int(valSel) > 0 {
		var timeouts []int
		switch int(typeSel) {
		case 0:
			timeouts = jobDlg.cfg.DisplayTimeouts
		case 1:
			timeouts = jobDlg.cfg.SleepTimeouts
		case 2:
			timeouts = jobDlg.cfg.HibernateTimeouts
		}
		idx := int(valSel) - 1
		if idx < len(timeouts) {
			minutes = timeouts[idx]
		}
	}

	a := config.ScheduledAction{
		Type:    actionTypes[int(typeSel)],
		Source:  sources[int(srcSel)],
		Minutes: minutes,
	}

	jobDlg.job.Actions = append(jobDlg.job.Actions, a)
	addActionToListBox(a)
}

func addActionToListBox(a config.ScheduledAction) {
	listBoxAddStr(jobDlg.actionList, formatActionLine(a))
}

func removeSelectedAction() {
	sel, _, _ := sendMessageW.Call(jobDlg.actionList, lbGetCurSel, 0, 0)
	idx := int(sel)
	if idx < 0 || idx >= len(jobDlg.job.Actions) {
		return
	}
	jobDlg.job.Actions = append(jobDlg.job.Actions[:idx], jobDlg.job.Actions[idx+1:]...)
	sendMessageW.Call(jobDlg.actionList, lbDeleteStr, sel, 0)
}

func validateAndSave() bool {
	name := getWindowText(jobDlg.editName)
	if strings.TrimSpace(name) == "" {
		showError(jobDlg.hwnd, "Введите название задания")
		return false
	}

	var weekdays [7]bool
	anyDay := false
	for i := 0; i < 7; i++ {
		checked, _, _ := sendMessageW.Call(jobDlg.checkDays[i], bmGetCheck, 0, 0)
		weekdays[i] = checked == bstChecked
		if weekdays[i] {
			anyDay = true
		}
	}
	if !anyDay {
		showError(jobDlg.hwnd, "Выберите хотя бы один день недели")
		return false
	}

	hourStr := getWindowText(jobDlg.editHour)
	minStr := getWindowText(jobDlg.editMinute)
	hour, err1 := strconv.Atoi(strings.TrimSpace(hourStr))
	min, err2 := strconv.Atoi(strings.TrimSpace(minStr))
	if err1 != nil || err2 != nil || hour < 0 || hour > 23 || min < 0 || min > 59 {
		showError(jobDlg.hwnd, "Введите корректное время (ЧЧ:ММ, 00:00-23:59)")
		return false
	}

	if len(jobDlg.job.Actions) == 0 {
		showError(jobDlg.hwnd, "Добавьте хотя бы одно действие")
		return false
	}

	jobDlg.result = config.ScheduledJob{
		ID:       jobDlg.job.ID,
		Name:     strings.TrimSpace(name),
		Weekdays: weekdays,
		Hour:     hour,
		Minute:   min,
		Actions:  jobDlg.job.Actions,
		Active:   jobDlg.job.Active,
	}
	if jobDlg.result.ID == "" {
		jobDlg.result.ID = config.GenerateID()
		jobDlg.result.Active = true
	}
	jobDlg.ok = true
	return true
}

func showError(parent uintptr, text string) {
	t, _ := syscall.UTF16PtrFromString("Ошибка")
	m, _ := syscall.UTF16PtrFromString(text)
	messageBoxW.Call(parent, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x10)
}

func formatActionLine(a config.ScheduledAction) string {
	typeNames := map[config.ActionType]string{
		config.ActionDisplay:   "Экран",
		config.ActionSleep:     "Сон",
		config.ActionHibernate: "Гибернация",
	}
	sourceNames := map[config.SourceType]string{
		config.SourceAC: "Сеть (AC)",
		config.SourceDC: "Батарея (DC)",
	}
	val := "Никогда"
	if a.Minutes > 0 {
		val = formatMinutesRu(a.Minutes)
	}
	return fmt.Sprintf("%s / %s / %s", typeNames[a.Type], sourceNames[a.Source], val)
}

func formatMinutesRu(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d мин", minutes)
	}
	h := minutes / 60
	m := minutes % 60
	if m == 0 {
		return fmt.Sprintf("%d ч", h)
	}
	return fmt.Sprintf("%d ч %d мин", h, m)
}
