//go:build !windows

package cmd

import (
	"os"
	"strings"
)

// removePathExport removes lines that were written by install.sh / dev-install.sh
// into the user's shell rc file (e.g. `export PATH="$PATH:/home/user/go/bin"`).
// It tries .bashrc, .zshrc, and .profile; silently skips files that don't exist.
// On Windows this is a no-op — PATH is managed via the registry by install.ps1.
func removePathExport() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	rcFiles := []string{
		home + "/.bashrc",
		home + "/.zshrc",
		home + "/.profile",
	}

	// Markers written by both install scripts
	markers := []string{
		"# pg-factory",
		"# pg-factory dev",
	}

	for _, rc := range rcFiles {
		if _, err := os.Stat(rc); os.IsNotExist(err) {
			continue
		}
		if err := scrubLines(rc, markers); err != nil {
			PrintWarn("Could not clean " + rc + ": " + err.Error())
		} else {
			PrintInfo("Cleaned PATH export from " + rc)
		}
	}
}

// scrubLines rewrites path, dropping each marker line, the export line that
// follows it, and any blank lines that immediately precede the marker block.
// The installer appends: blank → marker → export, so all three are removed.
func scrubLines(path string, markers []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")

	isMarker := func(line string) bool {
		trimmed := strings.TrimSpace(line)
		for _, m := range markers {
			if trimmed == m {
				return true
			}
		}
		return false
	}

	// Build a set of line indices to drop.
	drop := make(map[int]bool)
	for i, line := range lines {
		if isMarker(line) {
			// Drop the marker itself.
			drop[i] = true
			// Drop the export line that follows (if any).
			if i+1 < len(lines) {
				drop[i+1] = true
			}
			// Drop any blank lines immediately before the marker.
			for j := i - 1; j >= 0 && strings.TrimSpace(lines[j]) == ""; j-- {
				drop[j] = true
			}
		}
	}

	var out []string
	for i, line := range lines {
		if !drop[i] {
			out = append(out, line)
		}
	}

	return os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0644)
}

