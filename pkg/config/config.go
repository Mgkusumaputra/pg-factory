// pkg/config/config.go
package config

import (
	"os"
	"path/filepath"
)

const dirName = ".pgfactory"

// Dir returns the absolute path to ~/.pgfactory.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, dirName), nil
}

// InstancesPath returns the path to ~/.pgfactory/instances.json.
func InstancesPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "instances.json"), nil
}

// EnsureDirs creates ~/.pgfactory if it does not exist.
func EnsureDirs() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}
