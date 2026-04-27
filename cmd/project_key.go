package cmd

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Mgkusumaputra/pg-factory/pkg/project"
)

// canonicalProjectKey returns the normalized project identity used for new
// links. It uses absolute cwd-based keys to avoid basename collisions.
func canonicalProjectKey(cwd string) string {
	key := filepath.Clean(cwd)
	if abs, err := filepath.Abs(key); err == nil {
		key = abs
	}
	if runtime.GOOS == "windows" {
		// Windows paths are case-insensitive; normalize casing for stable keys.
		key = strings.ToLower(key)
	}
	return key
}

// projectKeysForDir returns the canonical key plus the legacy basename key to
// keep existing users' links backward-compatible.
func projectKeysForDir(cwd string) []string {
	canonical := canonicalProjectKey(cwd)
	legacy := filepath.Base(cwd)
	if legacy == "" || legacy == "." || legacy == canonical {
		return []string{canonical}
	}
	return []string{canonical, legacy}
}

func linkedInstancesForDir(ps *project.Store, cwd string) ([]string, error) {
	var instances []string
	seen := make(map[string]bool)
	for _, key := range projectKeysForDir(cwd) {
		found, err := ps.InstancesFor(key)
		if err != nil {
			return nil, err
		}
		for _, instance := range found {
			if !seen[instance] {
				seen[instance] = true
				instances = append(instances, instance)
			}
		}
	}
	return instances, nil
}

func displayProjectName(projectKey string) string {
	// Legacy keys are already concise; canonical keys are rendered as basename.
	if filepath.IsAbs(projectKey) || strings.Contains(projectKey, string(filepath.Separator)) {
		name := filepath.Base(projectKey)
		if name != "" && name != "." {
			return name
		}
	}
	return projectKey
}
