package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"OpsVault/internal/driver"
	"OpsVault/pkg/credutil"
	"OpsVault/pkg/logger"
	"OpsVault/pkg/sysutil"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

type JenkinsDriver struct {
	*BaseDriver
}

func NewJenkinsDriver(cli DockerClient, cfg *viper.Viper) *JenkinsDriver {
	port := cfg.GetInt("jenkins.port")
	if port == 0 {
		port = 8080
	}
	agentPort := cfg.GetInt("jenkins.agent_port")
	if agentPort == 0 {
		agentPort = 50000
	}
	image := cfg.GetString("jenkins.image")
	if image == "" {
		image = "jenkins/jenkins:lts"
	}
	base := NewBaseDriver("jenkins", cli.Raw(), cfg, image, []string{
		fmt.Sprintf("%d:8080", port),
		fmt.Sprintf("%d:50000", agentPort),
	})
	drv := &JenkinsDriver{BaseDriver: base}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

func (d *JenkinsDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *JenkinsDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("8080/tcp")
	agentPort := nat.Port("50000/tcp")

	hostPort := d.Config.GetString("jenkins.port")
	if hostPort == "" {
		hostPort = "8080"
	}
	hostAgentPort := d.Config.GetString("jenkins.agent_port")
	if hostAgentPort == "" {
		hostAgentPort = "50000"
	}

	return &container.Config{
			Image: d.Image,
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", "curl -f http://localhost:8080/login || exit 1"},
				Interval:    15 * time.Second,
				Timeout:     10 * time.Second,
				StartPeriod: 30 * time.Second,
				Retries:     10,
			},
		}, &container.HostConfig{
			Binds: []string{
				toDockerBind(filepath.Join(d.DataDir, "data"), "/var/jenkins_home"),
			},
			PortBindings: nat.PortMap{
				port:      []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostPort}},
				agentPort: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostAgentPort}},
			},
		}, nil
}

func (d *JenkinsDriver) prepareConfig(confDir string) error {
	dataDir := filepath.Join(d.DataDir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	// Jenkins runs as UID 1000 internally. On Linux, if data directory is created by root,
	// the container will fail with "Permission denied". We must chown it to UID 1000.
	if sysutil.IsLinux() && sysutil.IsRoot() {
		logger.Infof("Linux root environment detected, changing ownership of %s to UID 1000:GID 1000", dataDir)
		if err := os.Chown(dataDir, 1000, 1000); err != nil {
			logger.Errorf("failed to chown %s to 1000:1000: %v", dataDir, err)
		}
	}
	return nil
}

func (d *JenkinsDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

func (d *JenkinsDriver) GetCredentials() []credutil.Credential {
	port := d.Config.GetString("jenkins.port")
	if port == "" {
		port = "8080"
	}
	creds := []credutil.Credential{
		{Label: "访问地址", Value: fmt.Sprintf("http://localhost:%s", port)},
		{Label: "默认用户", Value: "admin (或在首次启动页面引导创建)"},
	}

	pwdFile := filepath.Join(d.DataDir, "data", "secrets", "initialAdminPassword")
	if data, err := os.ReadFile(pwdFile); err == nil {
		pwd := strings.TrimSpace(string(data))
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

// Make sure JenkinsDriver implements ServiceDriver and CredentialProvider
var _ driver.ServiceDriver = (*JenkinsDriver)(nil)
var _ driver.CredentialProvider = (*JenkinsDriver)(nil)
