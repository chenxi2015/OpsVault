package ansible

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"OpsVault/pkg/mysqlconf"
	"OpsVault/pkg/nginxconf"
	"OpsVault/pkg/rabbitmqconf"
	"OpsVault/pkg/redisconf"
	"OpsVault/pkg/versionutil"
)

// PlaybookVars represents variables to inject into playbooks.
type PlaybookVars struct {
	TargetGroup       string
	DataRoot          string
	NetworkName       string
	CIDR              string
	NamePrefix        string
	RegistryMirrors   []string
	BinaryPath        string
	ConfigPath        string
	Purge             bool
	Force             bool
	ServiceName       string
	MySQLImage        string
	MySQLPort         int
	MySQLRootPassword string
	RedisImage        string
	RedisPort         int
	RedisPassword     string
	RabbitMQImage     string
	RabbitMQPort      int
	RabbitMQUIPort    int
	RabbitMQUser      string
	RabbitMQPwd       string
	MinIOImage        string
	MinIOPort         int
	MinIOConsolePort  int
	MinIORootUser     string
	MinIORootPassword string
	NacosImage        string
	NacosPort         int
	NacosGrpcPort1    int
	NacosGrpcPort2    int
	NacosAuthEnable   bool
	NacosAuthToken    string
	// Nginx binary driver fields
	NginxVersion         string
	NginxPCREVersion     string
	NginxOpenSSLVersion  string
	NginxOpenSSLURL      string
	NginxOpenSSLURLs     []string
	NginxInstallPath     string
	NginxSourceRoot      string
	NginxWWWRoot         string
	NginxSSLRoot         string
	NginxWWWLogsRoot     string
	NginxRunUser         string
	NginxRunGroup        string
	NginxSystemdUnitPath string
	// Pre-rendered nginx config file contents (auto-populated by GeneratePlaybookFile).
	// These ensure the Ansible and binary driver write identical configuration.
	NginxBaseConfig  string
	NginxProxyConfig string
	NginxSystemdUnit string
	NginxLogrotate   string
	// Pre-rendered Docker service configs (auto-populated by GeneratePlaybookFile).
	MySQLMyCnf   string
	RedisCnf     string
	RabbitMQConf string
}

// indentLines prefixes every line of s (except the first) with spaces*indent.
func indentLines(spaces int, s string) string {
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i := 1; i < len(lines); i++ {
		if lines[i] != "" {
			lines[i] = pad + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

// GeneratePlaybookFile parses the playbook template and writes it to a temporary file.
// For the nginx service the pre-rendered config file contents are automatically
// populated from pkg/nginxconf so both the local binary driver and the Ansible
// path write byte-for-byte identical configuration.
func GeneratePlaybookFile(tempDir string, serviceName string, vars PlaybookVars) (string, error) {
	if vars.TargetGroup == "" {
		vars.TargetGroup = "all"
	}
	// Auto-populate pre-rendered nginx config contents from the shared package.
	switch serviceName {
	case "nginx":
		if len(vars.NginxOpenSSLURLs) == 0 && vars.NginxOpenSSLVersion != "" {
			vars.NginxOpenSSLURLs = versionutil.GetOpenSSLDownloadURLs(vars.NginxOpenSSLVersion)
		}
		if vars.NginxOpenSSLURL == "" && len(vars.NginxOpenSSLURLs) > 0 {
			vars.NginxOpenSSLURL = vars.NginxOpenSSLURLs[0]
		}
		cfg := nginxconf.Config{
			InstallPath:     vars.NginxInstallPath,
			WWWRoot:         vars.NginxWWWRoot,
			SSLRoot:         vars.NginxSSLRoot,
			WWWLogsRoot:     vars.NginxWWWLogsRoot,
			RunUser:         vars.NginxRunUser,
			RunGroup:        vars.NginxRunGroup,
			SystemdUnitPath: vars.NginxSystemdUnitPath,
		}
		vars.NginxBaseConfig = nginxconf.RenderBaseConfig(cfg)
		vars.NginxProxyConfig = nginxconf.RenderProxyConfig()
		vars.NginxSystemdUnit = nginxconf.RenderSystemdUnit(cfg)
		vars.NginxLogrotate = nginxconf.RenderLogrotate(cfg)
	case "mysql":
		vars.MySQLMyCnf = mysqlconf.RenderMyCnf()
	case "redis":
		vars.RedisCnf = redisconf.RenderRedisCnf(vars.RedisPassword)
	case "rabbitmq":
		vars.RabbitMQConf = rabbitmqconf.RenderRabbitMQConf(vars.RabbitMQUser, vars.RabbitMQPwd)
	}

	tmplStr, exists := PlaybookTemplates[serviceName]
	if !exists {
		return "", fmt.Errorf("playbook template for service %s not found", serviceName)
	}

	// Register custom template functions.
	funcMap := template.FuncMap{
		"indent": indentLines,
		"join":   strings.Join,
	}

	tmpl, err := template.New(serviceName).Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse playbook template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute playbook template: %w", err)
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir %s: %w", tempDir, err)
	}

	tempFile := filepath.Join(tempDir, fmt.Sprintf("%s_playbook.yml", serviceName))
	if err := os.WriteFile(tempFile, buf.Bytes(), 0600); err != nil {
		return "", fmt.Errorf("failed to write playbook file: %w", err)
	}

	return tempFile, nil
}

// GenerateUninstallPlaybookFile parses the uninstall playbook template and writes it to a temporary file.
func GenerateUninstallPlaybookFile(tempDir string, serviceName string, vars PlaybookVars) (string, error) {
	if vars.TargetGroup == "" {
		vars.TargetGroup = "all"
	}
	if serviceName == "nginx" && vars.NginxInstallPath == "" {
		vars.NginxInstallPath = "/usr/local/nginx"
	}
	if serviceName == "nginx" && vars.NginxWWWRoot == "" {
		vars.NginxWWWRoot = "/data/wwwroot"
	}
	if serviceName == "nginx" && vars.NginxSSLRoot == "" {
		vars.NginxSSLRoot = "/data/ssl"
	}
	if serviceName == "nginx" && vars.NginxWWWLogsRoot == "" {
		vars.NginxWWWLogsRoot = "/data/wwwlogs"
	}

	tmplStr, exists := UninstallTemplates[serviceName]
	if !exists {
		return "", fmt.Errorf("uninstall playbook template for service %s not found", serviceName)
	}

	funcMap := template.FuncMap{
		"indent": indentLines,
		"join":   strings.Join,
	}

	tmpl, err := template.New(serviceName + "_uninstall").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse uninstall playbook template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute uninstall playbook template: %w", err)
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir %s: %w", tempDir, err)
	}

	tempFile := filepath.Join(tempDir, fmt.Sprintf("%s_uninstall_playbook.yml", serviceName))
	if err := os.WriteFile(tempFile, buf.Bytes(), 0600); err != nil {
		return "", fmt.Errorf("failed to write uninstall playbook file: %w", err)
	}

	return tempFile, nil
}

// GenerateReloadPlaybookFile parses the reload playbook template and writes it to a temporary file.
func GenerateReloadPlaybookFile(tempDir string, serviceName string, vars PlaybookVars) (string, error) {
	if vars.TargetGroup == "" {
		vars.TargetGroup = "all"
	}

	tmplStr, exists := ReloadTemplates[serviceName]
	if !exists {
		return "", fmt.Errorf("reload playbook template for service %s not found", serviceName)
	}

	funcMap := template.FuncMap{
		"indent": indentLines,
		"join":   strings.Join,
	}

	tmpl, err := template.New(serviceName + "_reload").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse reload playbook template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute reload playbook template: %w", err)
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir %s: %w", tempDir, err)
	}

	tempFile := filepath.Join(tempDir, fmt.Sprintf("%s_reload_playbook.yml", serviceName))
	if err := os.WriteFile(tempFile, buf.Bytes(), 0600); err != nil {
		return "", fmt.Errorf("failed to write reload playbook file: %w", err)
	}

	return tempFile, nil
}

