package dialog

import (
	"syscall"
	"unsafe"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	messageBoxW    = user32.NewProc("MessageBoxW")
)

func ShowAbout() {
	title, _ := syscall.UTF16PtrFromString("О программе — WinSettingsGui")
	text, _ := syscall.UTF16PtrFromString(
		"WinSettingsGui v1.0\r\n\r\n" +
			"Управление настройками питания Windows\r\n" +
			"через системный трей.\r\n\r\n" +
			"Автор: Vladimir Iarovoi <me@iarovoivv.com>\r\n" +
			"Соавторство: Claude Code (Anthropic)\r\n\r\n" +
			"© 2026")

	const mbOK = 0x00000000
	const mbIconInformation = 0x00000040

	messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(text)),
		uintptr(unsafe.Pointer(title)),
		uintptr(mbOK|mbIconInformation),
	)
}
