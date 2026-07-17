package fileutil

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

// UpdateYAMLValue updates a specific key's value under a specific service section in a YAML file,
// preserving the original layout, key order, and comments.
func UpdateYAMLValue(filePath, service, key, newValue string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	inServiceBlock := false

	// Regexp to match the key line. e.g. "  root_password: ..."
	// Group 1: Leading spaces and key name with colon (e.g. "  root_password: ")
	// Group 2: The comment and anything after it (e.g. " # comment")
	keyRegex, err := regexp.Compile(fmt.Sprintf(`^(\s+%s:\s*)(?:"[^"]*"|'[^']*'|[^\s#]*)(.*)$`, regexp.QuoteMeta(key)))
	if err != nil {
		return err
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check if we enter/leave the service block.
		// A service block starts with a non-indented line "service:"
		if strings.HasPrefix(trimmed, service+":") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			inServiceBlock = true
		} else if inServiceBlock && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			// Leave service block if we encounter another non-indented non-comment key
			inServiceBlock = false
		}

		if inServiceBlock && keyRegex.MatchString(line) {
			matches := keyRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				line = matches[1] + `"` + newValue + `"` + matches[2]
			}
			// Only update the first occurrence under the service
			inServiceBlock = false
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Write back the modified content
	output := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(filePath, []byte(output), 0644)
}
