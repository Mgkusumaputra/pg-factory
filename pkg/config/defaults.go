package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// WorkstationMode controls which directory scope pg-factory uses as home base.
type WorkstationMode string

const (
	// WorkstationCWD resolves the project root from os.Getwd() at runtime.
	// Every directory the user is in becomes its own project context.
	WorkstationCWD WorkstationMode = "cwd"

	// WorkstationModeCustomPath uses a fixed parent directory set by the user
	// (e.g. ~/projects). Only sub-directories of that path can auto-link instances.
	WorkstationModeCustomPath WorkstationMode = "path"

	// WorkstationGlobal applies no restriction — any directory on the machine can
	// manage a pg instance. Best for developers with many scattered repos.
	WorkstationGlobal WorkstationMode = "global"
)

// Defaults holds the user's global preferences persisted in ~/.pgfactory/config.json.
type Defaults struct {
	PGVersion       string          `json:"pg_version"`
	BasePort        int             `json:"base_port"`
	WorkstationMode WorkstationMode `json:"workstation_mode"`
	// WorkstationPath is only meaningful when WorkstationMode == WorkstationPath.
	WorkstationPath string `json:"workstation_path,omitempty"`
}

// FallbackDefaults returns sensible built-in defaults (no config.json required).
func FallbackDefaults() Defaults {
	return Defaults{
		PGVersion:       "16-alpine",
		BasePort:        5432,
		WorkstationMode: WorkstationGlobal,
	}
}

// GlobalConfigPath returns the absolute path to ~/.pgfactory/config.json.
func GlobalConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// DefaultsExist returns true when config.json is already present on disk.
func DefaultsExist() bool {
	p, err := GlobalConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// ReadDefaults reads ~/.pgfactory/config.json.
// Returns FallbackDefaults() when the file is absent; returns an error only
// when the file exists but cannot be parsed.
func ReadDefaults() (Defaults, error) {
	p, err := GlobalConfigPath()
	if err != nil {
		return FallbackDefaults(), err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return FallbackDefaults(), nil
	}
	if err != nil {
		return FallbackDefaults(), err
	}
	var d Defaults
	if err := json.Unmarshal(data, &d); err != nil {
		return FallbackDefaults(), err
	}
	return d, nil
}

// WriteDefaults atomically writes d to ~/.pgfactory/config.json.
func WriteDefaults(d Defaults) error {
	p, err := GlobalConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
