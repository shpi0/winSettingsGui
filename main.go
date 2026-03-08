package main

import (
	_ "embed"

	"winSettingsGui/internal/tray"
)

//go:embed resources/app.ico
var appIcon []byte

func main() {
	tray.Run(appIcon)
}
