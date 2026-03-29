package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const envKey = "DATABASE_URL"

// writeEnvLocal writes or updates DATABASE_URL in <dir>/.env.local.
// Returns (created, error) where created is true when the file was newly made.
func writeEnvLocal(dir, connStr string) (created bool, err error) {
	envPath := filepath.Join(dir, ".env.local")
	newLine := envKey + "=" + connStr

	// ── File does not exist yet — create it ─────────────────────────────────
	if _, statErr := os.Stat(envPath); os.IsNotExist(statErr) {
		if err := os.WriteFile(envPath, []byte(newLine+"\n"), 0644); err != nil {
			return false, fmt.Errorf("could not create .env.local: %w", err)
		}
		return true, nil
	}

	// ── File exists — update DATABASE_URL line or append it ─────────────────
	f, err := os.Open(envPath)
	if err != nil {
		return false, fmt.Errorf("could not read .env.local: %w", err)
	}

	var lines []string
	found := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, envKey+"=") {
			lines = append(lines, newLine)
			found = true
		} else {
			lines = append(lines, line)
		}
	}
	f.Close()
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("could not read .env.local: %w", err)
	}

	if !found {
		lines = append(lines, newLine)
	}

	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	tmpPath := envPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return false, fmt.Errorf("could not update .env.local: %w", err)
	}
	if err := os.Rename(tmpPath, envPath); err != nil {
		os.Remove(tmpPath)
		return false, fmt.Errorf("could not update .env.local: %w", err)
	}
	return false, nil
}

// removeFromEnvLocal removes the DATABASE_URL line from <dir>/.env.local.
// Returns (true, nil) when the key was found and removed.
// Returns (false, nil) when the file doesn't exist or the key wasn't present.
// If the file becomes empty after removal it is deleted entirely.
func removeFromEnvLocal(dir string) (removed bool, err error) {
	envPath := filepath.Join(dir, ".env.local")

	data, readErr := os.ReadFile(envPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return false, nil // nothing to do
		}
		return false, fmt.Errorf("could not read .env.local: %w", readErr)
	}

	var kept []string
	found := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, envKey+"=") {
			found = true // skip — this is the line we're removing
		} else {
			kept = append(kept, line)
		}
	}

	if !found {
		return false, nil // key wasn't in the file
	}

	// Trim trailing empty lines left behind
	for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1]) == "" {
		kept = kept[:len(kept)-1]
	}

	// If nothing remains, remove the file
	if len(kept) == 0 {
		return true, os.Remove(envPath)
	}

	content := strings.Join(kept, "\n") + "\n"
	if writeErr := os.WriteFile(envPath, []byte(content), 0644); writeErr != nil {
		return false, fmt.Errorf("could not update .env.local: %w", writeErr)
	}
	return true, nil
}

