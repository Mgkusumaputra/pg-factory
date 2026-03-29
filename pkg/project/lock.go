package project

import (
	"errors"
	"os"
	"time"
)

func lockFile(lockPath string) (*os.File, error) {
	for range 50 {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
		if err == nil {
			return f, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, errors.New("failed to acquire lock")
}

func unlockFile(f *os.File) error {
	name := f.Name()
	err := f.Close()
	if err != nil {
		return err
	}
	return os.Remove(name)
}
