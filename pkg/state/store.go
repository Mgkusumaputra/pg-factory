package state

type Store struct {
	Path string
}

func New(path string) *Store {
	return &Store{Path: path}
}