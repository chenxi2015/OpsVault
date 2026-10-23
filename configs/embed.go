package configs

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultConfigTemplate holds the embedded default.example.yaml template.
//
//go:embed default.example.yaml
var DefaultConfigTemplate []byte

// AnsibleConfigTemplate holds the embedded ansible.example.yaml template.
//
//go:embed ansible.example.yaml
var AnsibleConfigTemplate []byte

// InitResult represents the outcome of initializing a single configuration file.
type InitResult struct {
	Path    string
	Created bool
	Skipped bool
}

// WriteTemplates writes the embedded default.yaml and ansible.yaml templates into targetDir.
// If overwrite is false, existing files are preserved and marked as Skipped.
func WriteTemplates(targetDir string, overwrite bool) ([]InitResult, error) {
	if targetDir == "" {
		return nil, fmt.Errorf("target config directory is required")
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("create config directory %s: %w", targetDir, err)
	}

	items := []struct {
		name string
		data []byte
	}{
		{name: "default.yaml", data: DefaultConfigTemplate},
		{name: "ansible.yaml", data: AnsibleConfigTemplate},
	}

	results := make([]InitResult, 0, len(items))
	for _, item := range items {
		dst := filepath.Join(targetDir, item.name)
		if !overwrite {
			if _, err := os.Stat(dst); err == nil {
				results = append(results, InitResult{
					Path:    dst,
					Skipped: true,
				})
				continue
			}
		}

		if err := os.WriteFile(dst, item.data, 0o644); err != nil {
			return nil, fmt.Errorf("write config file %s: %w", dst, err)
		}
		results = append(results, InitResult{
			Path:    dst,
			Created: true,
		})
	}

	return results, nil
}

