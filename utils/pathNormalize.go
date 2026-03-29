package utils

import (
	"path"
	"path/filepath"
)

func PathNormalize(p string) (string, error) {
	if p == "" {
		return "", nil
	}

	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}

	return path.Clean(filepath.ToSlash(abs)), nil
}
