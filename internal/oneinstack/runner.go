package oneinstack

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/viper"
)

type Runner struct {
	config *viper.Viper
}

func NewRunner(cfg *viper.Viper) *Runner {
	return &Runner{config: cfg}
}

func (r *Runner) InstallNginx() error {
	tmpDir, err := os.MkdirTemp("", "opsvault-oneinstack-*")
	if err != nil {
		return err
	}
	scriptPath, err := NewDownloader(r.config).DownloadAutoScript(tmpDir)
	if err != nil {
		return err
	}

	cmd := exec.Command("bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"ONEINSTACK_COMPONENT=nginx",
		"ONEINSTACK_ONLY_NGINX=1",
		"ONEINSTACK_WWWROOT="+r.config.GetString("oneinstack.www_root"),
		"ONEINSTACK_SSLROOT="+r.config.GetString("oneinstack.ssl_root"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run oneinstack install: %w: %s", err, string(output))
	}
	return nil
}

func (r *Runner) UpgradeNginx(targetVersion string) error {
	upgradeScript := filepath.Join(r.config.GetString("oneinstack.nginx_install_path"), "..", "upgrade.sh")
	cmd := exec.Command("bash", upgradeScript, "nginx", targetVersion)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run nginx upgrade: %w: %s", err, string(output))
	}
	return nil
}

func (r *Runner) UninstallNginx() error {
	uninstallScript := filepath.Join(r.config.GetString("oneinstack.nginx_install_path"), "..", "uninstall.sh")
	cmd := exec.Command("bash", uninstallScript, "nginx")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run nginx uninstall: %w: %s", err, string(output))
	}
	return nil
}
