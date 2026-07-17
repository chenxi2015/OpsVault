package backup

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveConfigPaths resolves the host configuration file or directory paths for each service.
func (m *BackupManager) ResolveConfigPaths() map[string]string {
	dataRoot := m.config.GetString("docker.data_root")
	if dataRoot == "" {
		dataRoot = m.config.GetString("system.root_dir")
		if dataRoot == "" {
			dataRoot = "/data/opsvault"
		}
	}

	nginxInstallPath := m.config.GetString("nginx.install_path")
	if nginxInstallPath == "" {
		nginxInstallPath = "/usr/local/nginx"
	}

	nginxVhostDir := m.config.GetString("nginx.vhost_dir")
	if nginxVhostDir == "" {
		nginxVhostDir = filepath.Join(nginxInstallPath, "conf", "vhost")
	}

	paths := map[string]string{
		"nginx":       nginxVhostDir,
		"mysql":       filepath.Join(dataRoot, "mysql", "conf"),
		"redis":       filepath.Join(dataRoot, "redis", "conf"),
		"rocketmq":    filepath.Join(dataRoot, "rocketmq", "conf"),
		"rabbitmq":    filepath.Join(dataRoot, "rabbitmq", "conf"),
		"postgres":    filepath.Join(dataRoot, "postgres", "conf"),
		"elk":         filepath.Join(dataRoot, "elk", "conf"),
		"nacos":       filepath.Join(dataRoot, "nacos", "conf"),
		"system_root": dataRoot,
	}

	// Add global configuration file if loaded
	cfgFile := m.config.ConfigFileUsed()
	if cfgFile != "" {
		paths["global"] = cfgFile
	} else {
		// Try to see if default.yaml exists in configs
		if _, err := os.Stat("configs/default.yaml"); err == nil {
			paths["global"] = "configs/default.yaml"
		}
	}

	return paths
}

// shouldExclude checks if a path should be excluded from the backup.
func (m *BackupManager) shouldExclude(path string, prefix string, isDir bool) (bool, bool) {
	if prefix != "system_root" {
		return false, false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, false
	}

	// 1. Exclude backup storage directory itself
	backupDir, err := filepath.Abs(m.GetBackupDir())
	if err == nil && (absPath == backupDir || strings.HasPrefix(absPath, backupDir+string(filepath.Separator))) {
		return true, true
	}

	// 2. Exclude nginx website root
	wwwRoot := m.config.GetString("nginx.www_root")
	if wwwRoot != "" {
		absWww, err := filepath.Abs(wwwRoot)
		if err == nil && (absPath == absWww || strings.HasPrefix(absPath, absWww+string(filepath.Separator))) {
			return true, true
		}
	}

	// 3. Exclude nginx logs
	wwwLogs := m.config.GetString("nginx.wwwlogs_root")
	if wwwLogs != "" {
		absLogs, err := filepath.Abs(wwwLogs)
		if err == nil && (absPath == absLogs || strings.HasPrefix(absPath, absLogs+string(filepath.Separator))) {
			return true, true
		}
	}

	// 4. Exclude system global log path
	sysLogs := m.config.GetString("log.storage_path")
	if sysLogs != "" {
		absSysLogs, err := filepath.Abs(sysLogs)
		if err == nil && (absPath == absSysLogs || strings.HasPrefix(absPath, absSysLogs+string(filepath.Separator))) {
			return true, true
		}
	}

	// 5. Exclude component directories specified in configuration under system_root (e.g. mysql/data, mysql/logs, bin)
	dataRoot := m.config.GetString("docker.data_root")
	if dataRoot == "" {
		dataRoot = m.config.GetString("system.root_dir")
		if dataRoot == "" {
			dataRoot = "/data/opsvault"
		}
	}
	absDataRoot, err := filepath.Abs(dataRoot)
	if err == nil {
		rel, err := filepath.Rel(absDataRoot, absPath)
		if err == nil {
			parts := strings.Split(rel, string(filepath.Separator))

			// Load dynamic exclude directories, fallback to standard defaults if empty
			excludeDirs := m.config.GetStringSlice("backup.exclude_dirs")
			if len(excludeDirs) == 0 {
				excludeDirs = []string{"data", "logs", "log", "wwwlogs", "bin", "wwwroot"}
			}

			for _, part := range parts {
				partLower := strings.ToLower(part)
				for _, exDir := range excludeDirs {
					if partLower == strings.ToLower(exDir) {
						return true, true
					}
				}
			}
		}
	}

	return false, false
}
