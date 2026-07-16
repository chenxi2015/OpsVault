package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
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

// GetConfigSearchPaths returns all default candidate paths to search for config files.
func GetConfigSearchPaths() []string {
	paths := []string{"./configs", "."}

	if exePath, err := os.Executable(); err == nil {
		if realPath, errSym := filepath.EvalSymlinks(exePath); errSym == nil {
			exePath = realPath
		}
		exeDir := filepath.Dir(exePath)
		paths = append(paths, filepath.Join(exeDir, "configs"), exeDir)

		if filepath.Base(exeDir) == "bin" {
			parentDir := filepath.Dir(exeDir)
			paths = append(paths, filepath.Join(parentDir, "configs"), parentDir)
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".opsvault"))
	}

	return paths
}

// GetDefaultWriteConfigPath returns the default target path for writing config files.
func GetDefaultWriteConfigPath() string {
	if exePath, err := os.Executable(); err == nil {
		if realPath, errSym := filepath.EvalSymlinks(exePath); errSym == nil {
			exePath = realPath
		}
		exeDir := filepath.Dir(exePath)
		if filepath.Base(exeDir) == "bin" {
			return filepath.Join(filepath.Dir(exeDir), "configs", "default.yaml")
		}
		return filepath.Join(exeDir, "configs", "default.yaml")
	}
	return filepath.Join("configs", "default.yaml")
}
