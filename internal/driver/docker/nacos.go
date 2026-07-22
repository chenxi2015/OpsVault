package docker

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"time"

	"OpsVault/internal/driver"
	"OpsVault/pkg/credutil"
	"OpsVault/pkg/fileutil"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

// NacosDriver represents the Docker driver for Nacos service
type NacosDriver struct {
	*BaseDriver
	authEnable bool
	authToken  string
}

// NewNacosDriver creates and returns a new NacosDriver instance
func NewNacosDriver(cli DockerClient, cfg *viper.Viper, authToken string) *NacosDriver {
	port := cfg.GetInt("nacos.port")
	if port == 0 {
		port = 8848
	}
	grpcPort1 := cfg.GetInt("nacos.grpc_port_1")
	if grpcPort1 == 0 {
		grpcPort1 = 9848
	}
	grpcPort2 := cfg.GetInt("nacos.grpc_port_2")
	if grpcPort2 == 0 {
		grpcPort2 = 9849
	}
	image := cfg.GetString("nacos.image")
	if image == "" {
		image = "nacos/nacos-server:v2.3.2"
	}
	authEnable := true
	if cfg.IsSet("nacos.auth_enable") {
		authEnable = cfg.GetBool("nacos.auth_enable")
	}
	if authToken == "" {
		authToken = cfg.GetString("nacos.auth_token")
	}

	base := NewBaseDriver("nacos", cli.Raw(), cfg, image, []string{
		fmt.Sprintf("%d:8848", port),
		fmt.Sprintf("%d:9848", grpcPort1),
		fmt.Sprintf("%d:9849", grpcPort2),
	})
	drv := &NacosDriver{BaseDriver: base, authEnable: authEnable, authToken: authToken}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

// Install runs the installer for Nacos container
func (d *NacosDriver) Install() error {
	if d.authEnable && d.authToken == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		d.authToken = base64.StdEncoding.EncodeToString(b)
		d.Config.Set("nacos.auth_token", d.authToken)
		cfgPath := d.Config.ConfigFileUsed()
		if cfgPath == "" {
			cfgPath = fileutil.GetDefaultWriteConfigPath()
		}
		_ = fileutil.UpdateYAMLValue(cfgPath, "nacos", "auth_token", d.authToken)
	}
	return d.installWithSpec(d.containerSpec)
}

func (d *NacosDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("8848/tcp")
	grpcPort1 := nat.Port("9848/tcp")
	grpcPort2 := nat.Port("9849/tcp")

	hostPort := d.Config.GetString("nacos.port")
	if hostPort == "" {
		hostPort = "8848"
	}
	hostGrpcPort1 := d.Config.GetString("nacos.grpc_port_1")
	if hostGrpcPort1 == "" {
		hostGrpcPort1 = "9848"
	}
	hostGrpcPort2 := d.Config.GetString("nacos.grpc_port_2")
	if hostGrpcPort2 == "" {
		hostGrpcPort2 = "9849"
	}

	env := []string{
		"MODE=standalone",
	}
	if d.authEnable {
		env = append(env,
			"NACOS_AUTH_ENABLE=true",
			"NACOS_AUTH_TOKEN="+d.authToken,
			"NACOS_AUTH_TOKEN_SECRET="+d.authToken,
			"NACOS_AUTH_IDENTITY_KEY=opsvault_key",
			"NACOS_AUTH_IDENTITY_VALUE=opsvault_value",
		)
	} else {
		env = append(env, "NACOS_AUTH_ENABLE=false")
	}

	// Health check via actuator endpoint, fallback to exit 1 if curl fails
	healthCmd := "curl -f http://localhost:8848/nacos/actuator/health || exit 1"

	return &container.Config{
			Image: d.Image,
			Env:   env,
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", healthCmd},
				Interval:    15 * time.Second,
				Timeout:     5 * time.Second,
				StartPeriod: 30 * time.Second,
				Retries:     10,
			},
		}, &container.HostConfig{
			Binds: []string{
				toDockerBind(filepath.Join(d.DataDir, "data"), "/home/nacos/data"),
				toDockerBind(filepath.Join(d.DataDir, "logs"), "/home/nacos/logs"),
			},
			PortBindings: nat.PortMap{
				port:      []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostPort}},
				grpcPort1: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostGrpcPort1}},
				grpcPort2: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostGrpcPort2}},
			},
		}, nil
}

func (d *NacosDriver) prepareConfig(confDir string) error {
	dataDir := filepath.Join(d.DataDir, "data")
	logsDir := filepath.Join(d.DataDir, "logs")
	if err := fileutil.EnsureDir(dataDir, 0755); err != nil {
		return err
	}
	return fileutil.EnsureDir(logsDir, 0755)
}

// Upgrade upgrades the Nacos service to target version
func (d *NacosDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

// GetCredentials returns credentials for Nacos
func (d *NacosDriver) GetCredentials() []credutil.Credential {
	port := d.Config.GetString("nacos.port")
	if port == "" {
		port = "8848"
	}
	creds := []credutil.Credential{
		{Label: "Web Console Address", Value: fmt.Sprintf("http://localhost:%s/nacos", port)},
	}
	if d.authEnable {
		creds = append(creds,
			credutil.Credential{Label: "Default Username", Value: "nacos"},
			credutil.Credential{Label: "Default Password", Value: "nacos (Change in Console)"},
			credutil.Credential{Label: "Auth Token Secret", Value: d.authToken},
		)
	} else {
		creds = append(creds, credutil.Credential{Label: "Authentication", Value: "Disabled"})
	}
	return creds
}

// Status overrides BaseDriver status to properly translate health status
func (d *NacosDriver) Status() (*driver.ServiceStatus, error) {
	status, err := d.BaseDriver.Status()
	if err != nil {
		return nil, err
	}
	if status.Status == "healthy" {
		status.Running = true
	}
	return status, nil
}
