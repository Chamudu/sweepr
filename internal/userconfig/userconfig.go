// Package userconfig stores small, non-sensitive UI preferences in the
// operating system's standard per-user configuration directory.
package userconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configVersion = 1

type config struct {
	Version         int  `json:"version"`
	WelcomeComplete bool `json:"welcome_complete"`
}

// WelcomeComplete reports whether this user has acknowledged the first-run
// safety introduction. A missing file is a normal first launch, not an error.
func WelcomeComplete() (bool, error) {
	path, err := configPath()
	if err != nil {
		return false, err
	}
	value, err := load(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value.WelcomeComplete, nil
}

// MarkWelcomeComplete remembers the acknowledgement for later launches.
func MarkWelcomeComplete() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	return save(path, config{Version: configVersion, WelcomeComplete: true})
}

func configPath() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user configuration directory: %w", err)
	}
	return filepath.Join(directory, "sweepr", "config.json"), nil
}

func load(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	var value config
	if err := json.Unmarshal(data, &value); err != nil {
		return config{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return value, nil
}

func save(path string, value config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode user configuration: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write user configuration: %w", err)
	}
	return nil
}
