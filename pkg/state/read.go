package state

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/Mgkusumaputra/pg-factory/pkg/types"
)

// ErrNotFound is returned by Read when the state file does not exist yet.
var ErrNotFound = errors.New("state file not found")

func (s *Store) readUnlocked(v any) error {
	f, err := os.Open(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(v)
}

func (s *Store) Read(v any) error {
	return s.readUnlocked(v)
}

// ReadInstances reads the instance list from the store.
// Returns an empty list (not an error) when the file doesn't exist yet.
func (s *Store) ReadInstances() (types.InstanceList, error) {
	var list types.InstanceList
	err := s.Read(&list)
	if errors.Is(err, ErrNotFound) {
		return types.InstanceList{}, nil
	}
	return list, err
}
