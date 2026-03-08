package config

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var defaultTimeouts = []int{1, 5, 15, 30}

type ActionType string

const (
	ActionDisplay   ActionType = "display"
	ActionSleep     ActionType = "sleep"
	ActionHibernate ActionType = "hibernate"
)

type SourceType string

const (
	SourceAC SourceType = "ac"
	SourceDC SourceType = "dc"
)

type ScheduledAction struct {
	Type    ActionType `json:"type"`
	Source  SourceType `json:"source"`
	Minutes int        `json:"minutes"`
}

type ScheduledJob struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Weekdays [7]bool           `json:"weekdays"`
	Hour     int               `json:"hour"`
	Minute   int               `json:"minute"`
	Actions  []ScheduledAction `json:"actions"`
	Active   bool              `json:"active"`
}

func GenerateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

type Config struct {
	DisplayTimeouts   []int          `json:"display_timeouts"`
	SleepTimeouts     []int          `json:"sleep_timeouts"`
	HibernateTimeouts []int          `json:"hibernate_timeouts"`
	ScheduledJobs     []ScheduledJob `json:"scheduled_jobs,omitempty"`
}

func configDir() string {
	appData := os.Getenv("APPDATA")
	return filepath.Join(appData, "WinSettingsGui")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func DefaultConfig() Config {
	return Config{
		DisplayTimeouts:   append([]int{}, defaultTimeouts...),
		SleepTimeouts:     append([]int{}, defaultTimeouts...),
		HibernateTimeouts: append([]int{}, defaultTimeouts...),
	}
}

func Load() (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}
	return cfg, nil
}

func Save(cfg Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}
