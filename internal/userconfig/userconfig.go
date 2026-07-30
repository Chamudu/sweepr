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
	Version         int          `json:"version"`
	WelcomeComplete bool         `json:"welcome_complete"`
	Scan            *Preferences `json:"scan,omitempty"`
}

// Preferences contains read-only scan setup. Destructive mode, selections,
// and confirmations are intentionally absent and therefore cannot persist.
type Preferences struct {
	Root     string   `json:"root"`
	Scope    string   `json:"scope"`
	Enabled  []string `json:"enabled_scanners"`
	MinSize  string   `json:"minimum_size,omitempty"`
	MinAge   int      `json:"minimum_age_days,omitempty"`
	Excludes []string `json:"excluded_paths,omitempty"`
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
	value, loadErr := load(path)
	if loadErr != nil && !os.IsNotExist(loadErr) {
		return loadErr
	}
	value.Version, value.WelcomeComplete = configVersion, true
	return save(path, value)
}

func LoadPreferences() (Preferences, bool, error) {
	path, err := configPath()
	if err != nil {
		return Preferences{}, false, err
	}
	value, err := load(path)
	if os.IsNotExist(err) {
		return Preferences{}, false, nil
	}
	if err != nil {
		return Preferences{}, false, err
	}
	if value.Scan == nil {
		return Preferences{}, false, nil
	}
	return *value.Scan, true, nil
}

func SavePreferences(preferences Preferences) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	value, loadErr := load(path)
	if loadErr != nil && !os.IsNotExist(loadErr) {
		return loadErr
	}
	value.Version = configVersion
	copy := preferences
	copy.Enabled = append([]string(nil), preferences.Enabled...)
	copy.Excludes = append([]string(nil), preferences.Excludes...)
	value.Scan = &copy
	return save(path, value)
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
