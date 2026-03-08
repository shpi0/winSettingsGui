package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var defaultTimeouts = []int{1, 5, 15, 30}

type Config struct {
	DisplayTimeouts   []int `json:"display_timeouts"`
	SleepTimeouts     []int `json:"sleep_timeouts"`
	HibernateTimeouts []int `json:"hibernate_timeouts"`
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
