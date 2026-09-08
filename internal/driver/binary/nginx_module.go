package binary

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"OpsVault/pkg/fileutil"
	"OpsVault/pkg/logger"
)

// ModuleInfo represents the metadata of an Nginx dynamic module.
type ModuleInfo struct {
	Name     string    `json:"name"`
	SoName   string    `json:"so_name"`
	Path     string    `json:"path"`
	ConfPath string    `json:"conf_path"`
	Enabled  bool      `json:"enabled"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
}

// AddModuleOptions defines the options for adding a dynamic module.
type AddModuleOptions struct {
	Name string
	URL  string
	Path string
}

// PresetModules map popular module names to their Git repositories.
var PresetModules = map[string]string{
	"headers-more": "https://github.com/openresty/headers-more-nginx-module.git",
	"echo":         "https://github.com/openresty/echo-nginx-module.git",
	"fancyindex":   "https://github.com/aperezdc/ngx-fancyindex.git",
	"brotli":       "https://github.com/google/ngx_brotli.git",
	"cache-purge":  "https://github.com/FRiCKLE/ngx_cache_purge.git",
}

// ListModules returns all installed dynamic modules and their status.
func (d *NginxDriver) ListModules() ([]ModuleInfo, error) {
	plan := newNginxInstallPlan(d.Config)
	modulesDir := filepath.Join(plan.installPath, "modules")
	confDir := filepath.Join(plan.installPath, "conf", "modules")

	if _, err := os.Stat(modulesDir); os.IsNotExist(err) {
		return []ModuleInfo{}, nil
	}

	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return nil, fmt.Errorf("read modules directory: %w", err)
	}

	var results []ModuleInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".so") {
			continue
		}
		soPath := filepath.Join(modulesDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		modName := strings.TrimSuffix(entry.Name(), ".so")
		modName = strings.TrimPrefix(modName, "ngx_")
		modName = strings.TrimPrefix(modName, "http_")

		confFile := filepath.Join(confDir, modName+".conf")
		enabled := false
		if _, err := os.Stat(confFile); err == nil {
			enabled = true
		} else {
			// Try matching any .conf inside confDir loading this .so
			confFile = ""
			if confEntries, err := os.ReadDir(confDir); err == nil {
				for _, ce := range confEntries {
					if strings.HasSuffix(ce.Name(), ".conf") {
						cp := filepath.Join(confDir, ce.Name())
						data, err := os.ReadFile(cp)
						if err == nil && strings.Contains(string(data), entry.Name()) {
							enabled = true
							confFile = cp
							break
						}
					}
				}
			}
		}

		results = append(results, ModuleInfo{
			Name:     modName,
			SoName:   entry.Name(),
			Path:     soPath,
			ConfPath: confFile,
			Enabled:  enabled,
			Size:     info.Size(),
			ModTime:  info.ModTime(),
		})
	}

	return results, nil
}

// AddModule compiles and installs a dynamic module into Nginx.
func (d *NginxDriver) AddModule(opts AddModuleOptions) error {
	if !d.isLinuxOrTest() {
		return fmt.Errorf("adding nginx modules is only supported on Linux")
	}

	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return fmt.Errorf("module name cannot be empty")
	}

	moduleURL := strings.TrimSpace(opts.URL)
	moduleLocalPath := strings.TrimSpace(opts.Path)

	if moduleURL == "" && moduleLocalPath == "" {
		if presetURL, ok := PresetModules[name]; ok {
			moduleURL = presetURL
		} else {
			return fmt.Errorf("unknown preset module %q; please specify --url or --path", name)
		}
	}

	plan := newNginxInstallPlan(d.Config)
	nginxSrcDir := plan.nginxSourceDir()

	// Step 1: Ensure Nginx source directory exists
	if _, err := os.Stat(nginxSrcDir); os.IsNotExist(err) {
		logger.Infof("[nginx-module] Nginx source not found at %s. Downloading Nginx v%s...", nginxSrcDir, plan.version)
		installer := newNginxInstaller(d.Config)
		if err := installer.downloadSources(); err != nil {
			return fmt.Errorf("download nginx source: %w", err)
		}
		if err := installer.extractSources(); err != nil {
			return fmt.Errorf("extract nginx source: %w", err)
		}
	}

	// Step 2: Prepare module source code
	var modSrcDir string
	if moduleLocalPath != "" {
		if _, err := os.Stat(moduleLocalPath); os.IsNotExist(err) {
			return fmt.Errorf("specified module path does not exist: %s", moduleLocalPath)
		}
		modSrcDir = moduleLocalPath
	} else {
		targetDir := filepath.Join(plan.sourceRoot, "modules", name)
		if err := fileutil.EnsureDir(targetDir, 0o755); err != nil {
			return fmt.Errorf("create module source dir: %w", err)
		}
		if _, err := os.Stat(filepath.Join(targetDir, "config")); os.IsNotExist(err) {
			logger.Infof("[nginx-module] Cloning module repository %s into %s...", moduleURL, targetDir)
			_ = os.RemoveAll(targetDir)
			if output, err := runNginxCommand("", "git", "clone", "--depth", "1", moduleURL, targetDir); err != nil {
				return fmt.Errorf("git clone module repo (%s): %w: %s", moduleURL, err, string(output))
			}
		}
		modSrcDir = targetDir
	}

	// Step 3: Run configure with --add-dynamic-module
	logger.Infof("[nginx-module] Configuring Nginx with dynamic module: %s...", name)
	configureFlags := append(plan.configureArgs(), "--add-dynamic-module="+modSrcDir)
	if output, err := runNginxCommand(nginxSrcDir, "./configure", configureFlags...); err != nil {
		return fmt.Errorf("configure nginx module %s: %w: %s", name, err, string(output))
	}

	// Step 4: Run make modules
	logger.Infof("[nginx-module] Compiling module .so file(s)...")
	if output, err := runNginxCommand(nginxSrcDir, "make", "modules"); err != nil {
		return fmt.Errorf("make modules failed for %s: %w: %s", name, err, string(output))
	}

	// Step 5: Locate generated .so file(s) in objs/
	objsDir := filepath.Join(nginxSrcDir, "objs")
	objsEntries, err := os.ReadDir(objsDir)
	if err != nil {
		return fmt.Errorf("read objs directory: %w", err)
	}

	var generatedSos []string
	for _, entry := range objsEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".so") {
			generatedSos = append(generatedSos, filepath.Join(objsDir, entry.Name()))
		}
	}

	if len(generatedSos) == 0 {
		return fmt.Errorf("no .so dynamic module files generated in %s", objsDir)
	}

	// Step 6: Install .so files and create module .conf
	destModulesDir := filepath.Join(plan.installPath, "modules")
	if err := fileutil.EnsureDir(destModulesDir, 0o755); err != nil {
		return fmt.Errorf("create modules dir: %w", err)
	}

	destConfDir := filepath.Join(plan.installPath, "conf", "modules")
	if err := fileutil.EnsureDir(destConfDir, 0o755); err != nil {
		return fmt.Errorf("create modules conf dir: %w", err)
	}

	var installedFiles []string
	var confLines []string

	for _, soPath := range generatedSos {
		soName := filepath.Base(soPath)
		destSoPath := filepath.Join(destModulesDir, soName)
		if err := copyFile(soPath, destSoPath); err != nil {
			return fmt.Errorf("copy module .so to %s: %w", destSoPath, err)
		}
		installedFiles = append(installedFiles, destSoPath)
		confLines = append(confLines, fmt.Sprintf("load_module modules/%s;", soName))
	}

	moduleConfPath := filepath.Join(destConfDir, name+".conf")
	confContent := strings.Join(confLines, "\n") + "\n"
	if err := os.WriteFile(moduleConfPath, []byte(confContent), 0o644); err != nil {
		return fmt.Errorf("write module conf file %s: %w", moduleConfPath, err)
	}
	installedFiles = append(installedFiles, moduleConfPath)

	// Step 7: Ensure nginx.conf imports modules/*.conf
	nginxConfPath := filepath.Join(plan.installPath, "conf", "nginx.conf")
	if err := ensureModulesImportInConfig(nginxConfPath); err != nil {
		logger.Warnf("[nginx-module] Warning: failed to verify include directive in nginx.conf: %v", err)
	}

	// Step 8: Test syntax and reload
	logger.Infof("[nginx-module] Testing Nginx configuration...")
	nginxBin := filepath.Join(plan.installPath, "sbin", "nginx")
	if output, err := runNginxCommand("", nginxBin, "-t"); err != nil {
		// Rollback on syntax error
		for _, file := range installedFiles {
			_ = os.Remove(file)
		}
		return fmt.Errorf("nginx configuration test failed: %w: %s (rolled back module %s)", err, string(output), name)
	}

	logger.Infof("[nginx-module] Reloading Nginx service...")
	if err := d.Reload(); err != nil {
		logger.Warnf("[nginx-module] Reload service failed: %v", err)
	}

	logger.Infof("[nginx-module] Dynamic module %q installed and enabled successfully!", name)
	return nil
}

// RemoveModule uninstalls and disables a dynamic module.
func (d *NginxDriver) RemoveModule(moduleName string) error {
	if !d.isLinuxOrTest() {
		return fmt.Errorf("removing nginx modules is only supported on Linux")
	}

	name := strings.TrimSpace(moduleName)
	if name == "" {
		return fmt.Errorf("module name cannot be empty")
	}

	plan := newNginxInstallPlan(d.Config)
	destModulesDir := filepath.Join(plan.installPath, "modules")
	destConfDir := filepath.Join(plan.installPath, "conf", "modules")

	moduleConfPath := filepath.Join(destConfDir, name+".conf")
	var removed []string

	if data, err := os.ReadFile(moduleConfPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "load_module modules/") {
				soName := strings.TrimPrefix(line, "load_module modules/")
				soName = strings.TrimSuffix(soName, ";")
				soPath := filepath.Join(destModulesDir, soName)
				if err := os.Remove(soPath); err == nil {
					removed = append(removed, soPath)
				}
			}
		}
		if err := os.Remove(moduleConfPath); err == nil {
			removed = append(removed, moduleConfPath)
		}
	} else {
		// Fallback: try removing any matching .so
		soPath := filepath.Join(destModulesDir, name+".so")
		if err := os.Remove(soPath); err == nil {
			removed = append(removed, soPath)
		}
		soPathHttp := filepath.Join(destModulesDir, "ngx_http_"+name+"_module.so")
		if err := os.Remove(soPathHttp); err == nil {
			removed = append(removed, soPathHttp)
		}
	}

	if len(removed) == 0 {
		return fmt.Errorf("dynamic module %q not found or already removed", name)
	}

	// Verify syntax and reload
	nginxBin := filepath.Join(plan.installPath, "sbin", "nginx")
	if output, err := runNginxCommand("", nginxBin, "-t"); err != nil {
		logger.Warnf("[nginx-module] Syntax test warning after module removal: %s", string(output))
	} else {
		_ = d.Reload()
	}

	logger.Infof("[nginx-module] Dynamic module %q removed successfully.", name)
	return nil
}

// ensureModulesImportInConfig ensures that nginx.conf has an `include modules/*.conf;` line.
func ensureModulesImportInConfig(nginxConfPath string) error {
	data, err := os.ReadFile(nginxConfPath)
	if err != nil {
		return err
	}

	content := string(data)
	if strings.Contains(content, "include modules/*.conf;") || strings.Contains(content, "include modules/") {
		return nil
	}

	// Insert before `events {`
	if idx := strings.Index(content, "events {"); idx != -1 {
		newContent := content[:idx] + "include modules/*.conf;\n\n" + content[idx:]
		return os.WriteFile(nginxConfPath, []byte(newContent), 0o644)
	}

	return nil
}
