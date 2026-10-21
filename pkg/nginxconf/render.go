// Package nginxconf provides shared Nginx configuration rendering for both
// the local binary driver and the remote Ansible deploy path.
// All config content is generated here to guarantee consistency.
package nginxconf

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Config holds the parameters needed to render all Nginx config files.
// Both the binary driver (local install) and the Ansible playbook generator
// populate this struct from Viper config, ensuring a single source of truth.
type Config struct {
	InstallPath       string
	WWWRoot           string
	SSLRoot           string
	WWWLogsRoot       string
	RunUser           string
	RunGroup          string
	SystemdUnitPath   string
	LogrotatePath     string
	LogRotateStrategy string // "daily", "size", "both"
	LogRotateSize     string // e.g. "100M"
	LogRotateRotate   int    // retention count (default: 30)
	LogRotateCompress *bool  // gzip compress old logs (nil defaults to true)
	LogRotateDateExt  *bool  // date extension in archive names (nil defaults to true)
}

// RenderBaseConfig returns the content for nginx.conf (main config file).
func RenderBaseConfig(c Config) string {
	return fmt.Sprintf(baseConfigTemplate, c.RunUser, c.RunGroup, c.WWWLogsRoot, c.WWWLogsRoot, c.WWWRoot)
}

// RenderProxyConfig returns the content for proxy.conf (shared proxy settings).
func RenderProxyConfig() string {
	return proxyConfigTemplate
}

// RenderSystemdUnit returns the content for the nginx systemd service unit file.
func RenderSystemdUnit(c Config) string {
	nginxBin := c.InstallPath + "/sbin/nginx"
	nginxConf := c.InstallPath + "/conf/nginx.conf"
	return fmt.Sprintf(systemdUnitTemplate, nginxBin, nginxConf, nginxBin, nginxConf)
}

// RenderLogrotate returns the content for /etc/logrotate.d/nginx.
// It supports rotation based on time (daily/weekly/monthly), size, or both (daily + maxsize).
// It automatically covers both Nginx install logs (/usr/local/nginx/logs/*.log)
// and OpsVault website logs (wwwlogs/*.log) to ensure access.log and error.log are properly rotated.
func RenderLogrotate(c Config) string {
	installPath := filepath.ToSlash(strings.TrimRight(c.InstallPath, "/\\"))
	if installPath == "" {
		installPath = "/usr/local/nginx"
	}

	var logGlobs []string
	seen := make(map[string]bool)

	addGlob := func(dir string) {
		dir = filepath.ToSlash(strings.TrimRight(dir, "/\\"))
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		logGlobs = append(logGlobs, dir+"/*.log")
	}

	// Always cover Nginx install logs directory (where access.log and error.log are located by default)
	addGlob(filepath.Join(installPath, "logs"))

	// Also cover wwwlogs directory if configured
	if c.WWWLogsRoot != "" {
		addGlob(c.WWWLogsRoot)
	}

	strategy := strings.ToLower(strings.TrimSpace(c.LogRotateStrategy))
	if strategy == "" {
		strategy = "daily"
	}

	size := strings.TrimSpace(c.LogRotateSize)
	if size == "" {
		size = "100M"
	}

	rotate := c.LogRotateRotate
	if rotate <= 0 {
		rotate = 30
	}

	user := strings.TrimSpace(c.RunUser)
	if user == "" {
		user = "www"
	}
	group := strings.TrimSpace(c.RunGroup)
	if group == "" {
		group = user
	}

	compress := true
	if c.LogRotateCompress != nil {
		compress = *c.LogRotateCompress
	}

	dateext := true
	if c.LogRotateDateExt != nil {
		dateext = *c.LogRotateDateExt
	}

	var strategyDirective string
	switch strategy {
	case "size":
		strategyDirective = fmt.Sprintf("    size %s", size)
	case "both":
		strategyDirective = fmt.Sprintf("    daily\n    maxsize %s", size)
	case "weekly":
		strategyDirective = "    weekly"
	case "monthly":
		strategyDirective = "    monthly"
	default:
		strategyDirective = "    daily"
	}

	var dateextDirective string
	if dateext {
		if strategy == "size" || strategy == "both" {
			dateextDirective = "    dateext\n    dateformat -%Y%m%d_%s"
		} else {
			dateextDirective = "    dateext\n    dateformat -%Y%m%d"
		}
	}

	var compressDirective string
	if compress {
		compressDirective = "    compress\n    delaycompress"
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("%s {", strings.Join(logGlobs, " ")))
	lines = append(lines, strategyDirective)
	lines = append(lines, fmt.Sprintf("    rotate %d", rotate))
	lines = append(lines, "    missingok")
	if dateextDirective != "" {
		lines = append(lines, dateextDirective)
	}
	if compressDirective != "" {
		lines = append(lines, compressDirective)
	}
	lines = append(lines, "    notifempty")
	lines = append(lines, fmt.Sprintf("    create 0640 %s %s", user, group))
	lines = append(lines, "    sharedscripts")
	lines = append(lines, "    postrotate")
	lines = append(lines, fmt.Sprintf("        [ -f /var/run/nginx.pid ] && kill -USR1 $(cat /var/run/nginx.pid) 2>/dev/null || [ -f %s/logs/nginx.pid ] && kill -USR1 $(cat %s/logs/nginx.pid) 2>/dev/null || true", installPath, installPath))
	lines = append(lines, "    endscript")
	lines = append(lines, "}\n")

	return strings.Join(lines, "\n")
}

// FormatProxyPass standardizes the proxy pass address (e.g. "8080" -> "http://127.0.0.1:8080").
func FormatProxyPass(proxy string) string {
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		return ""
	}
	if !strings.HasPrefix(proxy, "http://") && !strings.HasPrefix(proxy, "https://") {
		if strings.HasPrefix(proxy, ":") {
			proxy = "http://127.0.0.1" + proxy
		} else if !strings.Contains(proxy, "/") && !strings.Contains(proxy, ":") {
			proxy = "http://127.0.0.1:" + proxy
		} else {
			proxy = "http://" + proxy
		}
	}
	return proxy
}

// RenderVHostHTTP generates production-ready Nginx configuration for an HTTP virtual host.
func RenderVHostHTTP(domain, root, proxyPass, wwwLogsRoot string) string {
	proxyPass = FormatProxyPass(proxyPass)
	logDir := strings.TrimRight(wwwLogsRoot, "/")
	if logDir == "" {
		logDir = "/data/wwwlogs"
	}

	accessLog := fmt.Sprintf("%s/%s_access.log", logDir, domain)
	errorLog := fmt.Sprintf("%s/%s_error.log", logDir, domain)

	if proxyPass != "" {
		return fmt.Sprintf(vhostHTTPProxyTemplate, domain, accessLog, errorLog, root, proxyPass)
	}

	return fmt.Sprintf(vhostHTTPStaticTemplate, domain, accessLog, errorLog, root)
}

// RenderVHostSSL generates production-ready Nginx configuration with SSL and HTTP-to-HTTPS redirect.
func RenderVHostSSL(domain, root, proxyPass, fullchain, privkey, wwwLogsRoot string) string {
	proxyPass = FormatProxyPass(proxyPass)
	logDir := strings.TrimRight(wwwLogsRoot, "/")
	if logDir == "" {
		logDir = "/data/wwwlogs"
	}

	accessLog := fmt.Sprintf("%s/%s_access.log", logDir, domain)
	errorLog := fmt.Sprintf("%s/%s_error.log", logDir, domain)

	var mainLocation string
	if proxyPass != "" {
		mainLocation = fmt.Sprintf(vhostSSLProxyLocationTemplate, proxyPass)
	} else {
		mainLocation = fmt.Sprintf(vhostSSLStaticLocationTemplate, root)
	}

	return fmt.Sprintf(vhostSSLTemplate, domain, root, domain, fullchain, privkey, accessLog, errorLog, mainLocation)
}
