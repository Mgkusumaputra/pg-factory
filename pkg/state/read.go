package state

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/Mgkusumaputra/pg-factory/pkg/types"
)

func (s *Store) Read(v any) error {
	f, err := os.Open(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("state file not found")
		}
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(v)
}

// ReadInstances reads the instance list from the store.
// Returns an empty list (not an error) when the file doesn't exist yet.
func (s *Store) ReadInstances() (types.InstanceList, error) {
	var list types.InstanceList
	err := s.Read(&list)
	if err != nil && err.Error() == "state file not found" {
		return types.InstanceList{}, nil
	}
	return list, err
}
