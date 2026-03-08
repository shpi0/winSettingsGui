package power

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"syscall"
)

const (
	subgroupVideo  = "7516b95f-f776-4464-8c53-06167f40cc99"
	settingDisplay = "3c0bc021-c8a8-4e07-a973-6b14cbcb2b7e"

	subgroupSleep    = "238c9fa8-0aad-41ed-83f4-97be242c8f20"
	settingStandby   = "29f6c1db-86da-48c5-9fdb-f2b67b1f44da"
	settingHibernate = "9d7815a6-7ee4-497e-8888-515a05f02364"
)

type PowerSource int

const (
	AC PowerSource = iota
	DC
)

var hexLineRe = regexp.MustCompile(`(?m)^\s+.*0x([0-9a-fA-F]+)\s*$`)

func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func queryTimeout(subgroup, setting string) (acMinutes, dcMinutes int, err error) {
	cmd := exec.Command("powercfg", "/query", "SCHEME_CURRENT", subgroup, setting)
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return -1, -1, fmt.Errorf("powercfg query failed: %w", err)
	}

	matches := hexLineRe.FindAllStringSubmatch(string(out), -1)
	// powercfg output always has 5 hex lines for a setting:
	// [0] min, [1] max, [2] increment, [3] AC index, [4] DC index
	if len(matches) < 5 {
		return -1, -1, fmt.Errorf("unexpected powercfg output: found %d hex values", len(matches))
	}

	acSeconds, _ := strconv.ParseInt(matches[3][1], 16, 64)
	dcSeconds, _ := strconv.ParseInt(matches[4][1], 16, 64)

	return int(acSeconds / 60), int(dcSeconds / 60), nil
}

func setTimeout(param string, minutes int, source PowerSource) error {
	suffix := "-ac"
	if source == DC {
		suffix = "-dc"
	}
	cmd := exec.Command("powercfg", "/change", param+suffix, strconv.Itoa(minutes))
	hideWindow(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("powercfg change failed: %s: %w", string(out), err)
	}
	return nil
}

func GetDisplayTimeout() (ac, dc int, err error) {
	return queryTimeout(subgroupVideo, settingDisplay)
}

func SetDisplayTimeout(minutes int, source PowerSource) error {
	return setTimeout("monitor-timeout", minutes, source)
}

func GetSleepTimeout() (ac, dc int, err error) {
	return queryTimeout(subgroupSleep, settingStandby)
}

func SetSleepTimeout(minutes int, source PowerSource) error {
	return setTimeout("standby-timeout", minutes, source)
}

func GetHibernateTimeout() (ac, dc int, err error) {
	return queryTimeout(subgroupSleep, settingHibernate)
}

func SetHibernateTimeout(minutes int, source PowerSource) error {
	return setTimeout("hibernate-timeout", minutes, source)
}
