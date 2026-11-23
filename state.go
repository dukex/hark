package hark

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// StatePath is where a provider keeps its cursor between runs:
// $XDG_STATE_HOME/hark/<name>.json, or ~/.local/state/hark/<name>.json.
func StatePath(name string) (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "hark", name+".json"), nil
}

// LoadState reads path into v. A missing file leaves v untouched.
func LoadState(path string, v any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// SaveState writes v to path atomically, creating its directory.
func SaveState(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path+".tmp", data, 0o600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}
