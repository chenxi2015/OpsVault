package fileutil

import (
	"fmt"
	"os"
)

func EnsureDir(path string, perm os.FileMode) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	return os.MkdirAll(path, perm)
}

func RemoveIfExists(path string) error {
	err := os.RemoveAll(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
