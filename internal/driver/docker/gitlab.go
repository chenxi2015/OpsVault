package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"OpsVault/internal/driver"
	"OpsVault/pkg/credutil"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

type GitLabDriver struct {
	*BaseDriver
}

func NewGitLabDriver(cli DockerClient, cfg *viper.Viper) *GitLabDriver {
	port := cfg.GetInt("gitlab.port")
	if port == 0 {
		port = 8082
	}
	sshPort := cfg.GetInt("gitlab.ssh_port")
	if sshPort == 0 {
		sshPort = 2222
	}
	httpsPort := cfg.GetInt("gitlab.https_port")
	if httpsPort == 0 {
		httpsPort = 8443
	}
	image := cfg.GetString("gitlab.image")
	if image == "" {
		image = "gitlab/gitlab-ce:latest"
	}
	base := NewBaseDriver("gitlab", cli.Raw(), cfg, image, []string{
		fmt.Sprintf("%d:80", port),
		fmt.Sprintf("%d:22", sshPort),
		fmt.Sprintf("%d:443", httpsPort),
	})
	drv := &GitLabDriver{BaseDriver: base}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

func (d *GitLabDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *GitLabDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("80/tcp")
	sshPort := nat.Port("22/tcp")
	httpsPort := nat.Port("443/tcp")

	hostPort := d.Config.GetString("gitlab.port")
	if hostPort == "" {
		hostPort = "8082"
	}
	hostSSHPort := d.Config.GetString("gitlab.ssh_port")
	if hostSSHPort == "" {
		hostSSHPort = "2222"
	}
	hostHTTPSPort := d.Config.GetString("gitlab.https_port")
	if hostHTTPSPort == "" {
		hostHTTPSPort = "8443"
	}

	return &container.Config{
			Image: d.Image,
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", "gitlab-ctl status || exit 1"},
				Interval:    30 * time.Second,
				Timeout:     10 * time.Second,
				StartPeriod: 120 * time.Second,
				Retries:     10,
			},
		}, &container.HostConfig{
			Binds: []string{
				filepath.Join(d.DataDir, "config") + ":/etc/gitlab",
				filepath.Join(d.DataDir, "logs") + ":/var/log/gitlab",
				filepath.Join(d.DataDir, "data") + ":/var/opt/gitlab",
			},
			PortBindings: nat.PortMap{
				port:      []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostPort}},
				sshPort:   []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostSSHPort}},
				httpsPort: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostHTTPSPort}},
			},
		}, nil
}

func (d *GitLabDriver) prepareConfig(confDir string) error {
	configDir := filepath.Join(d.DataDir, "config")
	logsDir := filepath.Join(d.DataDir, "logs")
	dataDir := filepath.Join(d.DataDir, "data")

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	return nil
}

func (d *GitLabDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

func (d *GitLabDriver) GetCredentials() []credutil.Credential {
	port := d.Config.GetString("gitlab.port")
	if port == "" {
		port = "8082"
	}
	creds := []credutil.Credential{
		{Label: "访问地址", Value: fmt.Sprintf("http://localhost:%s", port)},
		{Label: "默认用户", Value: "root"},
	}

	pwdFile := filepath.Join(d.DataDir, "config", "initial_root_password")
	if data, err := os.ReadFile(pwdFile); err == nil {
		lines := strings.Split(string(data), "\n")
		pwd := ""
		for _, line := range lines {
			if strings.HasPrefix(line, "Password:") {
				pwd = strings.TrimSpace(strings.TrimPrefix(line, "Password:"))
				break
			}
		}
		if pwd != "" {
			creds = append(creds, credutil.Credential{Label: "初始密码", Value: pwd})
		}
	} else {
		creds = append(creds, credutil.Credential{
			Label: "初始密码",
			Value: fmt.Sprintf("可在宿主机运行此命令查看:\ncat %s", pwdFile),
		})
	}
	return creds
}

// Make sure GitLabDriver implements ServiceDriver and CredentialProvider
var _ driver.ServiceDriver = (*GitLabDriver)(nil)
var _ driver.CredentialProvider = (*GitLabDriver)(nil)
