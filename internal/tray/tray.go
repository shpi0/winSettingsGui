package tray

import (
	"fmt"
	"strconv"

	"github.com/energye/systray"

	"winSettingsGui/internal/autostart"
	"winSettingsGui/internal/config"
	"winSettingsGui/internal/dialog"
	"winSettingsGui/internal/power"
)

var (
	appConfig config.Config
	iconData  []byte
)

type timeoutGroup struct {
	parent     *systray.MenuItem
	items      []*systray.MenuItem
	neverItem  *systray.MenuItem
	prefix     string
	neverLabel string
	apply      func(int)
	getCurrent func() int
}

var (
	displayACGroup  timeoutGroup
	displayDCGroup  timeoutGroup
	sleepACGroup    timeoutGroup
	sleepDCGroup    timeoutGroup
	hibernateACGroup timeoutGroup
	hibernateDCGroup timeoutGroup
)

func Run(icon []byte) {
	iconData = icon
	cfg, _ := config.Load()
	appConfig = cfg
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle("WinSettingsGui")
	systray.SetTooltip("Управление настройками питания")

	buildMenu()
}

func onExit() {}

func buildMenu() {
	mPower := systray.AddMenuItem("Настройки питания", "")
	mPower.Disable()

	buildDisplayMenu()
	buildSleepMenu()
	buildHibernateMenu()

	systray.AddSeparator()

	mSettings := systray.AddMenuItem("Настройки", "Настроить таймауты")
	mAutostart := systray.AddMenuItemCheckbox("Автозапуск", "Запускать при старте Windows", autostart.IsEnabled())
	mAbout := systray.AddMenuItem("О программе", "")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Выход", "Закрыть приложение")

	mSettings.Click(func() {
		newCfg, ok := dialog.ShowSettings(appConfig)
		if ok {
			appConfig = newCfg
			_ = config.Save(appConfig)
			rebuildAllGroups()
		}
	})

	mAutostart.Click(func() {
		enabled, err := autostart.Toggle()
		if err == nil {
			if enabled {
				mAutostart.Check()
			} else {
				mAutostart.Uncheck()
			}
		}
	})

	mAbout.Click(func() {
		dialog.ShowAbout()
	})

	mQuit.Click(func() {
		systray.Quit()
	})
}

func buildDisplayMenu() {
	mDisplay := systray.AddMenuItem("Экран", "")

	acDisplay, dcDisplay, _ := power.GetDisplayTimeout()

	mAC := mDisplay.AddSubMenuItem("При питании от сети", "")
	displayACGroup = timeoutGroup{
		parent:     mAC,
		prefix:     "Отключать через",
		neverLabel: "Не отключать экран",
		apply:      func(m int) { _ = power.SetDisplayTimeout(m, power.AC) },
		getCurrent: func() int { ac, _, _ := power.GetDisplayTimeout(); return ac },
	}
	populateGroup(&displayACGroup, appConfig.DisplayTimeouts, acDisplay)

	mDC := mDisplay.AddSubMenuItem("При питании от батареи", "")
	displayDCGroup = timeoutGroup{
		parent:     mDC,
		prefix:     "Отключать через",
		neverLabel: "Не отключать экран",
		apply:      func(m int) { _ = power.SetDisplayTimeout(m, power.DC) },
		getCurrent: func() int { _, dc, _ := power.GetDisplayTimeout(); return dc },
	}
	populateGroup(&displayDCGroup, appConfig.DisplayTimeouts, dcDisplay)
}

func buildSleepMenu() {
	mSleep := systray.AddMenuItem("Спящий режим", "")

	acSleep, dcSleep, _ := power.GetSleepTimeout()

	mAC := mSleep.AddSubMenuItem("При питании от сети", "")
	sleepACGroup = timeoutGroup{
		parent:     mAC,
		prefix:     "Засыпать через",
		neverLabel: "Не уходить в сон",
		apply:      func(m int) { _ = power.SetSleepTimeout(m, power.AC) },
		getCurrent: func() int { ac, _, _ := power.GetSleepTimeout(); return ac },
	}
	populateGroup(&sleepACGroup, appConfig.SleepTimeouts, acSleep)

	mDC := mSleep.AddSubMenuItem("При питании от батареи", "")
	sleepDCGroup = timeoutGroup{
		parent:     mDC,
		prefix:     "Засыпать через",
		neverLabel: "Не уходить в сон",
		apply:      func(m int) { _ = power.SetSleepTimeout(m, power.DC) },
		getCurrent: func() int { _, dc, _ := power.GetSleepTimeout(); return dc },
	}
	populateGroup(&sleepDCGroup, appConfig.SleepTimeouts, dcSleep)
}

func buildHibernateMenu() {
	mHibernate := systray.AddMenuItem("Гибернация", "")

	acHib, dcHib, _ := power.GetHibernateTimeout()

	mAC := mHibernate.AddSubMenuItem("При питании от сети", "")
	hibernateACGroup = timeoutGroup{
		parent:     mAC,
		prefix:     "Гибернация через",
		neverLabel: "Отключить гибернацию",
		apply:      func(m int) { _ = power.SetHibernateTimeout(m, power.AC) },
		getCurrent: func() int { ac, _, _ := power.GetHibernateTimeout(); return ac },
	}
	populateGroup(&hibernateACGroup, appConfig.HibernateTimeouts, acHib)

	mDC := mHibernate.AddSubMenuItem("При питании от батареи", "")
	hibernateDCGroup = timeoutGroup{
		parent:     mDC,
		prefix:     "Гибернация через",
		neverLabel: "Отключить гибернацию",
		apply:      func(m int) { _ = power.SetHibernateTimeout(m, power.DC) },
		getCurrent: func() int { _, dc, _ := power.GetHibernateTimeout(); return dc },
	}
	populateGroup(&hibernateDCGroup, appConfig.HibernateTimeouts, dcHib)
}

func rebuildAllGroups() {
	rebuildGroup(&displayACGroup, appConfig.DisplayTimeouts)
	rebuildGroup(&displayDCGroup, appConfig.DisplayTimeouts)
	rebuildGroup(&sleepACGroup, appConfig.SleepTimeouts)
	rebuildGroup(&sleepDCGroup, appConfig.SleepTimeouts)
	rebuildGroup(&hibernateACGroup, appConfig.HibernateTimeouts)
	rebuildGroup(&hibernateDCGroup, appConfig.HibernateTimeouts)
}

func rebuildGroup(g *timeoutGroup, timeouts []int) {
	for _, item := range g.items {
		item.Hide()
	}
	if g.neverItem != nil {
		g.neverItem.Hide()
	}
	current := g.getCurrent()
	populateGroup(g, timeouts, current)
}

func populateGroup(g *timeoutGroup, timeouts []int, currentMinutes int) {
	g.items = nil

	for _, t := range timeouts {
		label := g.prefix + " " + formatMinutes(t)
		checked := currentMinutes >= 0 && t == currentMinutes
		item := g.parent.AddSubMenuItemCheckbox(label, "", checked)
		g.items = append(g.items, item)

		t := t
		item.Click(func() {
			for _, other := range g.items {
				other.Uncheck()
			}
			g.neverItem.Uncheck()
			item.Check()
			g.apply(t)
		})
	}

	neverChecked := currentMinutes == 0
	g.neverItem = g.parent.AddSubMenuItemCheckbox(g.neverLabel, "", neverChecked)

	g.neverItem.Click(func() {
		for _, other := range g.items {
			other.Uncheck()
		}
		g.neverItem.Check()
		g.apply(0)
	})
}

func formatMinutes(minutes int) string {
	if minutes < 60 {
		return strconv.Itoa(minutes) + " мин"
	}
	h := minutes / 60
	m := minutes % 60
	if m == 0 {
		return fmt.Sprintf("%d ч", h)
	}
	return fmt.Sprintf("%d ч %d мин", h, m)
}
