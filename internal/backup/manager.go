package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"OpsVault/pkg/fileutil"
	"OpsVault/pkg/logger"

	"github.com/spf13/viper"
)

// BackupManager coordinates config backup and restore operations.
type BackupManager struct {
	config *viper.Viper
}

// NewBackupManager creates a new BackupManager instance.
func NewBackupManager(cfg *viper.Viper) *BackupManager {
	return &BackupManager{config: cfg}
}

// GetBackupDir returns the directory where backups are stored.
func (m *BackupManager) GetBackupDir() string {
	dir := m.config.GetString("backup.storage_path")
	if dir == "" {
		// Fallback to docker data root/bak
		dataRoot := m.config.GetString("docker.data_root")
		if dataRoot == "" {
			dataRoot = "/data/opsvault"
		}
		dir = filepath.Join(dataRoot, "bak")
	}
	return dir
}

// ListBackups returns a list of all available backups sorted by timestamp descending.
func (m *BackupManager) ListBackups() ([]*BackupMetadata, error) {
	backupDir := m.GetBackupDir()
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("read backup dir: %w", err)
	}

	var backups []*BackupMetadata
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			path := filepath.Join(backupDir, f.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				logger.Errorf("failed to read metadata file %s: %v", path, err)
				continue
			}

			var meta BackupMetadata
			if err := json.Unmarshal(data, &meta); err != nil {
				logger.Errorf("failed to parse metadata file %s: %v", path, err)
				continue
			}
			backups = append(backups, &meta)
		}
	}

	// Sort backups by timestamp descending (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// DeleteBackup deletes a backup and its metadata file.
func (m *BackupManager) DeleteBackup(name string) error {
	backupDir := m.GetBackupDir()
	tarGzPath := filepath.Join(backupDir, name+".tar.gz")
	jsonPath := filepath.Join(backupDir, name+".json")

	if err := fileutil.RemoveIfExists(tarGzPath); err != nil {
		return fmt.Errorf("delete tar.gz file: %w", err)
	}
	if err := fileutil.RemoveIfExists(jsonPath); err != nil {
		return fmt.Errorf("delete json metadata file: %w", err)
	}

	return nil
}
