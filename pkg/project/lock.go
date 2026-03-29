package project

import (
	"fmt"
	"os"
	"time"
)

// lockFile creates an exclusive lock file at lockPath by atomically creating
// it with O_EXCL. Retries for up to 5 seconds (50 × 100 ms).
func lockFile(lockPath string) (*os.File, error) {
	for range 50 {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
		if err == nil {
			return f, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("failed to acquire lock %s after 5s (stale lock?)", lockPath)
}

// unlockFile closes and removes the lock file. Called via defer; errors are
// intentionally ignored because the critical work is already done by this point
// and the OS will reclaim the file handle on process exit anyway.
func unlockFile(f *os.File) {
	name := f.Name()
	f.Close()
	os.Remove(name)
}
