// Package nginxconf provides shared Nginx configuration rendering for both
// the local binary driver and the remote Ansible deploy path.
// All config content is generated here to guarantee consistency.
package nginxconf

import (
	"fmt"
	"strings"
)

// Config holds the parameters needed to render all Nginx config files.
// Both the binary driver (local install) and the Ansible playbook generator
// populate this struct from Viper config, ensuring a single source of truth.
type Config struct {
	InstallPath     string
	WWWRoot         string
	SSLRoot         string
	WWWLogsRoot     string
	RunUser         string
	RunGroup        string
	SystemdUnitPath string
	LogrotatePath   string
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
func RenderLogrotate(c Config) string {
	return fmt.Sprintf(logrotateTemplate, c.WWWLogsRoot)
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
