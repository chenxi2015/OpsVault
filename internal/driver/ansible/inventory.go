package ansible

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// HostConfig defines the SSH connection details for a remote host.
type HostConfig struct {
	IP                string `mapstructure:"ip"`
	Port              int    `mapstructure:"port"`
	User              string `mapstructure:"user"`
	SSHPrivateKey     string `mapstructure:"ssh_private_key"`
	SSHPassword       string `mapstructure:"ssh_password"`
	PythonInterpreter string `mapstructure:"python_interpreter"`
}

// GroupConfig represents a group of hosts.
type GroupConfig struct {
	Name  string       `mapstructure:"name"`
	Hosts []HostConfig `mapstructure:"hosts"`
}

// Config wraps all ansible-related configurations.
type Config struct {
	BinPath         string        `mapstructure:"bin_path"`
	PlaybookBinPath string        `mapstructure:"playbook_bin_path"`
	TempDir         string        `mapstructure:"temp_dir"`
	Groups          []GroupConfig `mapstructure:"groups"`
}

// LoadConfig loads the Ansible configuration from viper.
func LoadConfig(v *viper.Viper) (*Config, error) {
	cfg := &Config{
		BinPath:         v.GetString("ansible.bin_path"),
		PlaybookBinPath: v.GetString("ansible.playbook_bin_path"),
		TempDir:         v.GetString("ansible.temp_dir"),
	}
	if cfg.BinPath == "" {
		cfg.BinPath = "ansible"
	}
	if cfg.PlaybookBinPath == "" {
		cfg.PlaybookBinPath = "ansible-playbook"
	}
	if cfg.TempDir == "" {
		cfg.TempDir = "./ansible_tmp"
	}

	var groups []GroupConfig
	if err := v.UnmarshalKey("ansible.inventory.groups", &groups); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ansible inventory groups: %w", err)
	}
	cfg.Groups = groups
	return cfg, nil
}

// GenerateInventoryFile generates an INI format inventory file in the temp directory.
// Returns the file path of the temporary inventory.
func GenerateInventoryFile(cfg *Config) (string, error) {
	// Ensure temp directory exists
	if err := os.MkdirAll(cfg.TempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir %s: %w", cfg.TempDir, err)
	}

	var builder strings.Builder

	for _, group := range cfg.Groups {
		if group.Name == "" {
			continue
		}
		builder.WriteString(fmt.Sprintf("[%s]\n", group.Name))
		for _, host := range group.Hosts {
			if host.IP == "" {
				continue
			}
			port := host.Port
			if port == 0 {
				port = 22
			}
			user := host.User
			if user == "" {
				user = "root"
			}

			line := fmt.Sprintf("%s ansible_port=%d ansible_user=%s", host.IP, port, user)
			if host.SSHPrivateKey != "" {
				line += fmt.Sprintf(" ansible_ssh_private_key_file=%s", host.SSHPrivateKey)
			}
			if host.SSHPassword != "" {
				line += fmt.Sprintf(" ansible_ssh_pass=%s", host.SSHPassword)
			}
			if host.PythonInterpreter != "" {
				line += fmt.Sprintf(" ansible_python_interpreter=%s", host.PythonInterpreter)
			}
			// Strict host key checking disable to prevent interactive prompts
			line += " ansible_ssh_common_args='-o StrictHostKeyChecking=no'"
			builder.WriteString(line)
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	tempFile := filepath.Join(cfg.TempDir, "hosts")
	if err := os.WriteFile(tempFile, []byte(builder.String()), 0600); err != nil {
		return "", fmt.Errorf("failed to write inventory file: %w", err)
	}

	return tempFile, nil
}
