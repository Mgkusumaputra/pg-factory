package state

import (
	"errors"
	"fmt"

	"github.com/Mgkusumaputra/pg-factory/pkg/types"
)

// UpdateInstances runs a read-modify-write transaction for instances.json under
// one lock to prevent lost updates under concurrent command execution.
func (s *Store) UpdateInstances(mutator func(*types.InstanceList) error) error {
	if mutator == nil {
		return fmt.Errorf("instances mutator must not be nil")
	}
	return s.withLock(func() error {
		var list types.InstanceList
		if err := s.readUnlocked(&list); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if err := mutator(&list); err != nil {
			return err
		}
		return s.writeUnlocked(list)
	})
}
