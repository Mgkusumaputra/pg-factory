//go:build !windows

package cmd

import (
	"bufio"
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

// scrubLines rewrites path, dropping any line that immediately follows a
// marker comment AND the marker line itself.
func scrubLines(path string, markers []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var out []string
	skipNext := false

	for scanner.Scan() {
		line := scanner.Text()
		if skipNext {
			skipNext = false
			continue // drop the `export PATH=…` line that follows
		}
		isMarker := false
		for _, m := range markers {
			if strings.TrimSpace(line) == m {
				isMarker = true
				break
			}
		}
		if isMarker {
			skipNext = true
			continue // drop the marker line itself
		}
		out = append(out, line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0644)
}
