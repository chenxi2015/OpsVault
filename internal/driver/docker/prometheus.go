package docker

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"OpsVault/pkg/credutil"
	"OpsVault/pkg/fileutil"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

// PrometheusDriver represents the Docker driver for Prometheus service
type PrometheusDriver struct {
	*BaseDriver
}

// NewPrometheusDriver creates and returns a new PrometheusDriver instance
func NewPrometheusDriver(cli DockerClient, cfg *viper.Viper) *PrometheusDriver {
	port := cfg.GetInt("prometheus.port")
	if port == 0 {
		port = 9090
	}
	image := cfg.GetString("prometheus.image")
	if image == "" {
		image = "prom/prometheus:latest"
	}

	base := NewBaseDriver("prometheus", cli.Raw(), cfg, image, []string{
		fmt.Sprintf("%d:9090", port),
	})
	drv := &PrometheusDriver{BaseDriver: base}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

// Install runs the installer for Prometheus container
func (d *PrometheusDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *PrometheusDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("9090/tcp")
	hostPort := d.Config.GetString("prometheus.port")
	if hostPort == "" {
		hostPort = "9090"
	}

	configPath := filepath.Join(d.DataDir, "prometheus.yml")
	dataPath := filepath.Join(d.DataDir, "data")

	return &container.Config{
			Image: d.Image,
			Cmd: []string{
				"--config.file=/etc/prometheus/prometheus.yml",
				"--storage.tsdb.path=/prometheus",
				"--web.enable-lifecycle",
			},
		}, &container.HostConfig{
			Binds: []string{
				toDockerBind(configPath, "/etc/prometheus/prometheus.yml"),
				toDockerBind(dataPath, "/prometheus"),
			},
			PortBindings: nat.PortMap{
				port: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostPort}},
			},
		}, nil
}

func (d *PrometheusDriver) prepareConfig(confDir string) error {
	dataPath := filepath.Join(d.DataDir, "data")
	if err := fileutil.EnsureDir(dataPath, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(d.DataDir, "prometheus.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultContent := `global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'node_exporter'
    static_configs:
      - targets: ['localhost:9100']
`
		if err := os.WriteFile(configPath, []byte(defaultContent), 0644); err != nil {
			return fmt.Errorf("write default prometheus.yml: %w", err)
		}
	}
	return nil
}

// Upgrade upgrades the Prometheus service to target version
func (d *PrometheusDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

// Reload triggers Prometheus configuration reload via API endpoint
func (d *PrometheusDriver) Reload() error {
	port := d.Config.GetString("prometheus.port")
	if port == "" {
		port = "9090"
	}
	url := fmt.Sprintf("http://127.0.0.1:%s/-/reload", port)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return d.Restart()
	}

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return d.Restart()
	}
	_ = resp.Body.Close()
	return nil
}

// GetCredentials returns connection credentials for Prometheus
func (d *PrometheusDriver) GetCredentials() []credutil.Credential {
	port := d.Config.GetString("prometheus.port")
	if port == "" {
		port = "9090"
	}
	return []credutil.Credential{
		{Label: "Web UI", Value: fmt.Sprintf("http://localhost:%s", port)},
		{Label: "Config File", Value: filepath.Join(d.DataDir, "prometheus.yml")},
	}
}
