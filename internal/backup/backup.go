package backup

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"OpsVault/pkg/fileutil"
	"OpsVault/pkg/logger"

	"github.com/spf13/viper"
)

// BackupMetadata holds information about a configuration backup.
type BackupMetadata struct {
	Name        string    `json:"name"`
	Timestamp   time.Time `json:"timestamp"`
	Services    []string  `json:"services"`
	SizeBytes   int64     `json:"size_bytes"`
	Description string    `json:"description"`
}

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

// ResolveConfigPaths resolves the host configuration file or directory paths for each service.
func (m *BackupManager) ResolveConfigPaths() map[string]string {
	dataRoot := m.config.GetString("docker.data_root")
	if dataRoot == "" {
		dataRoot = "/data/opsvault"
	}

	nginxInstallPath := m.config.GetString("nginx.install_path")
	if nginxInstallPath == "" {
		nginxInstallPath = "/usr/local/nginx"
	}

	paths := map[string]string{
		"nginx":    filepath.Join(nginxInstallPath, "conf"),
		"mysql":    filepath.Join(dataRoot, "mysql", "conf"),
		"redis":    filepath.Join(dataRoot, "redis", "conf"),
		"rocketmq": filepath.Join(dataRoot, "rocketmq", "conf"),
		"rabbitmq": filepath.Join(dataRoot, "rabbitmq", "conf"),
		"postgres": filepath.Join(dataRoot, "postgres", "conf"),
		"elk":      filepath.Join(dataRoot, "elk", "conf"),
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

// CreateBackup creates a new config backup for the specified services.
// If services is empty or contains "all", it backs up all configured services.
func (m *BackupManager) CreateBackup(services []string, customName, description string) (*BackupMetadata, error) {
	backupDir := m.GetBackupDir()
	if err := fileutil.EnsureDir(backupDir, 0o755); err != nil {
		return nil, fmt.Errorf("ensure backup dir exists: %w", err)
	}

	resolvedPaths := m.ResolveConfigPaths()
	backupAll := len(services) == 0
	for _, s := range services {
		if strings.ToLower(s) == "all" {
			backupAll = true
			break
		}
	}

	// Determine which services we are backing up and filter out non-existent paths
	var targets []string
	if backupAll {
		for s := range resolvedPaths {
			targets = append(targets, s)
		}
	} else {
		for _, s := range services {
			name := strings.ToLower(s)
			if _, ok := resolvedPaths[name]; ok {
				targets = append(targets, name)
			} else {
				return nil, fmt.Errorf("unknown service: %s", s)
			}
		}
	}

	// Double check that we actually have directories/files to back up
	var activeTargets []string
	for _, t := range targets {
		path := resolvedPaths[t]
		if _, err := os.Stat(path); err == nil {
			activeTargets = append(activeTargets, t)
		} else {
			if !backupAll {
				return nil, fmt.Errorf("config path for service %s does not exist: %s", t, path)
			}
			logger.Debugf("Config path for service %s does not exist, skipping: %s", t, path)
		}
	}

	if len(activeTargets) == 0 {
		return nil, fmt.Errorf("no existing configurations found to back up")
	}

	// Define archive name
	timestamp := time.Now()
	name := customName
	if name == "" {
		name = fmt.Sprintf("backup_%s", timestamp.Format("20060102_150405"))
	}
	tarGzPath := filepath.Join(backupDir, name+".tar.gz")
	jsonPath := filepath.Join(backupDir, name+".json")

	// Create .tar.gz archive
	file, err := os.Create(tarGzPath)
	if err != nil {
		return nil, fmt.Errorf("create backup file: %w", err)
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	tw := tar.NewWriter(gw)

	for _, t := range activeTargets {
		path := resolvedPaths[t]
		err := m.addPathToTar(tw, path, t)
		if err != nil {
			// Clean up on failure
			_ = tw.Close()
			_ = gw.Close()
			_ = os.Remove(tarGzPath)
			return nil, fmt.Errorf("add path %s to archive: %w", path, err)
		}
	}

	// Close archive writers explicitly so all data is flushed
	if err := tw.Close(); err != nil {
		_ = gw.Close()
		_ = os.Remove(tarGzPath)
		return nil, fmt.Errorf("finalize tar archive: %w", err)
	}
	if err := gw.Close(); err != nil {
		_ = os.Remove(tarGzPath)
		return nil, fmt.Errorf("finalize gzip stream: %w", err)
	}

	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("get backup file size: %w", err)
	}

	// Write metadata JSON file
	meta := &BackupMetadata{
		Name:        name,
		Timestamp:   timestamp,
		Services:    activeTargets,
		SizeBytes:   fi.Size(),
		Description: description,
	}

	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal backup metadata: %w", err)
	}

	if err := os.WriteFile(jsonPath, metaData, 0o644); err != nil {
		return nil, fmt.Errorf("write backup metadata file: %w", err)
	}

	return meta, nil
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

// RestoreBackup restores configuration from a backup.
// If serviceName is specified, it only restores that service's config.
func (m *BackupManager) RestoreBackup(name, serviceName string) error {
	backupDir := m.GetBackupDir()
	tarGzPath := filepath.Join(backupDir, name+".tar.gz")
	jsonPath := filepath.Join(backupDir, name+".json")

	if _, err := os.Stat(tarGzPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file %s does not exist", tarGzPath)
	}

	// Read metadata
	var meta BackupMetadata
	jsonData, err := os.ReadFile(jsonPath)
	if err == nil {
		_ = json.Unmarshal(jsonData, &meta)
	}

	file, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("open backup file: %w", err)
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("init gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	resolvedPaths := m.ResolveConfigPaths()

	var restoredServices []string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar archive: %w", err)
		}

		parts := strings.Split(header.Name, "/")
		if len(parts) < 2 {
			continue
		}
		service := parts[0]

		// Filter by service if requested
		if serviceName != "" && strings.ToLower(serviceName) != "all" && strings.ToLower(serviceName) != service {
			continue
		}

		targetBaseDir, ok := resolvedPaths[service]
		if !ok {
			continue
		}

		// Reconstruct target path
		var targetPath string
		if service == "global" {
			targetPath = targetBaseDir
		} else {
			// e.g. header.Name is "mysql/conf/my.cnf", targetBaseDir is "/data/opsvault/mysql/conf"
			// subPath should be "my.cnf"
			subPath := strings.Join(parts[2:], "/")
			targetPath = filepath.Join(targetBaseDir, subPath)
		}

		// Process entry based on type
		switch header.Typeflag {
		case tar.TypeDir:
			if err := fileutil.EnsureDir(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("create directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			parentDir := filepath.Dir(targetPath)
			if err := fileutil.EnsureDir(parentDir, 0o755); err != nil {
				return fmt.Errorf("create parent directory for %s: %w", targetPath, err)
			}

			// Open file for writing
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("write file %s: %w", targetPath, err)
			}
			outFile.Close()
		}

		// Add service to list of restored services
		found := false
		for _, s := range restoredServices {
			if s == service {
				found = true
				break
			}
		}
		if !found {
			restoredServices = append(restoredServices, service)
		}
	}

	logger.Infof("Successfully restored configurations for services: %s", strings.Join(restoredServices, ", "))
	return nil
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

// addPathToTar walks the given sourcePath and writes it to the tar writer.
// prefix is the top level directory name inside the tarball (e.g., "nginx").
func (m *BackupManager) addPathToTar(tw *tar.Writer, sourcePath, prefix string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	var baseDir string
	if info.IsDir() {
		baseDir = sourcePath
	} else {
		baseDir = filepath.Dir(sourcePath)
	}

	return filepath.Walk(sourcePath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create the relative path inside the tarball
		var relPath string
		if info.IsDir() {
			relPath, err = filepath.Rel(baseDir, path)
			if relPath == "." {
				return nil
			}
			if err != nil {
				return err
			}
			relPath = filepath.Join(prefix, "conf", relPath)
		} else {
			relPath = filepath.Join(prefix, filepath.Base(path))
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if fi.Mode().IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	})
}
