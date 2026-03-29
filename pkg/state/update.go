package state

func (s *Store) Update(fn func(map[string]any) error) error {
	data := map[string]any{}
	_ = s.Read(&data)

	if err := fn(data); err != nil {
		return err
	}

	return s.Write(data)
}
