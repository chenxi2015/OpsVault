package docker

import (
	"fmt"
	"path/filepath"
	"time"

	"OpsVault/pkg/credutil"
	"OpsVault/pkg/fileutil"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

// OllamaDriver represents the Docker driver for Ollama LLM/Embedding inference service
type OllamaDriver struct {
	*BaseDriver
}

// NewOllamaDriver creates and returns a new OllamaDriver instance
func NewOllamaDriver(cli DockerClient, cfg *viper.Viper) *OllamaDriver {
	port := cfg.GetInt("ollama.port")
	if port == 0 {
		port = 11434
	}
	image := cfg.GetString("ollama.image")
	if image == "" {
		image = "ollama/ollama:latest"
	}

	base := NewBaseDriver("ollama", cli.Raw(), cfg, image, []string{
		fmt.Sprintf("%d:11434", port),
	})
	drv := &OllamaDriver{BaseDriver: base}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

// Install runs the installer for Ollama container
func (d *OllamaDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *OllamaDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("11434/tcp")

	hostPort := d.Config.GetString("ollama.port")
	if hostPort == "" {
		hostPort = "11434"
	}

	modelsPath := filepath.Join(d.DataDir, "models")

	return &container.Config{
			Image: d.Image,
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", "curl -f http://localhost:11434/api/tags || exit 1"},
				Interval:    10 * time.Second,
				Timeout:     5 * time.Second,
				StartPeriod: 10 * time.Second,
				Retries:     5,
			},
		}, &container.HostConfig{
			Binds: []string{
				toDockerBind(modelsPath, "/root/.ollama"),
			},
			PortBindings: nat.PortMap{
				port: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostPort}},
			},
		}, nil
}

func (d *OllamaDriver) prepareConfig(confDir string) error {
	modelsPath := filepath.Join(d.DataDir, "models")
	return fileutil.EnsureDir(modelsPath, 0755)
}

// Upgrade upgrades the Ollama service to target version
func (d *OllamaDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

// GetCredentials returns connection credentials for Ollama
func (d *OllamaDriver) GetCredentials() []credutil.Credential {
	port := d.Config.GetString("ollama.port")
	if port == "" {
		port = "11434"
	}
	return []credutil.Credential{
		{Label: "API Endpoint", Value: fmt.Sprintf("http://localhost:%s", port)},
		{Label: "OpenAI Compatible Base URL", Value: fmt.Sprintf("http://localhost:%s/v1", port)},
	}
}
