// pkg/project/project.go
// Tracks which project directories are linked to which pg-factory instances.
// State is persisted to ~/.pgfactory/projects.json as a JSON object where
// keys are project slugs (cwd basenames) and values are slices of instance names.
package project

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ProjectMap maps project slugs to the instance names they reference.
type ProjectMap map[string][]string

// Store persists project→instance links to a JSON file.
type Store struct {
	Path string
}

// New returns a Store backed by the given file path.
func New(path string) *Store {
	return &Store{Path: path}
}

// Load reads the project map from disk. Returns an empty map if the file does
// not exist yet.
func (s *Store) Load() (ProjectMap, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return ProjectMap{}, nil
		}
		return nil, err
	}
	var m ProjectMap
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Save writes the project map to disk atomically (lock-file + temp-rename).
// Two concurrent pg create calls will not corrupt projects.json.
func (s *Store) Save(m ProjectMap) error {
	lockPath := s.Path + ".lock"
	lf, err := lockFile(lockPath)
	if err != nil {
		return err
	}
	defer unlockFile(lf) //nolint:errcheck

	dir := filepath.Dir(s.Path)
	tmp, err := os.CreateTemp(dir, "projects-*.tmp")
	if err != nil {
		return err
	}

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), s.Path)
}

// Link records that instance is associated with project. Idempotent.
func (s *Store) Link(project, instance string) error {
	m, err := s.Load()
	if err != nil {
		return err
	}
	for _, existing := range m[project] {
		if existing == instance {
			return nil // already linked
		}
	}
	m[project] = append(m[project], instance)
	return s.Save(m)
}

// Unlink removes the association between instance and project. No-op if not
// linked.
func (s *Store) Unlink(project, instance string) error {
	m, err := s.Load()
	if err != nil {
		return err
	}
	instances := m[project]
	updated := instances[:0]
	for _, v := range instances {
		if v != instance {
			updated = append(updated, v)
		}
	}
	if len(updated) == 0 {
		delete(m, project)
	} else {
		m[project] = updated
	}
	return s.Save(m)
}

// InstancesFor returns the instance names linked to a project slug.
func (s *Store) InstancesFor(project string) ([]string, error) {
	m, err := s.Load()
	if err != nil {
		return nil, err
	}
	return m[project], nil
}

// ProjectsFor returns all project slugs that are linked to a given instance.
func (s *Store) ProjectsFor(instance string) ([]string, error) {
	m, err := s.Load()
	if err != nil {
		return nil, err
	}
	var projects []string
	for proj, instances := range m {
		for _, inst := range instances {
			if inst == instance {
				projects = append(projects, proj)
				break
			}
		}
	}
	return projects, nil
}
