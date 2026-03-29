package state

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Mgkusumaputra/pg-factory/pkg/types"
)

func (s *Store) Write(v any) error {
	dir := filepath.Dir(s.Path)
	lockPath := s.Path + ".lock"

	lf, err := lockFile(lockPath)
	if err != nil {
		return err
	}
	defer unlockFile(lf)

	tmp, err := os.CreateTemp(dir, "state-*.tmp")
	if err != nil {
		return err
	}

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")

	if err := enc.Encode(v); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), s.Path)
}

// WriteInstances writes the instance list to the store atomically.
func (s *Store) WriteInstances(list types.InstanceList) error {
	return s.Write(list)
}