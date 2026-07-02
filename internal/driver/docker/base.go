package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"OpsVault/internal/driver"
	"OpsVault/pkg/dockercli"
	"OpsVault/pkg/fileutil"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/spf13/viper"
)

type BaseDriver struct {
	Name          string
	Client        *client.Client
	Config        *viper.Viper
	Image         string
	ContainerName string
	DataDir       string
	Ports         []string
	NetworkName   string
}

func NewBaseDriver(name string, cli *client.Client, cfg *viper.Viper, image string, ports []string) *BaseDriver {
	dataRoot := cfg.GetString("docker.data_root")
	return &BaseDriver{
		Name:          name,
		Client:        cli,
		Config:        cfg,
		Image:         image,
		ContainerName: "opsvault-" + name,
		DataDir:       filepath.Join(dataRoot, name),
		Ports:         ports,
		NetworkName:   cfg.GetString("docker.network_name"),
	}
}

func (d *BaseDriver) EnsureReady(ctx context.Context) error {
	if err := fileutil.EnsureDir(d.DataDir, 0o755); err != nil {
		return err
	}
	return dockercli.EnsureNetwork(ctx, d.Client, d.NetworkName, d.Config.GetString("docker.cidr"))
}

func (d *BaseDriver) Start() error {
	if d.Client == nil {
		return fmt.Errorf("docker client is not available")
	}
	return d.Client.ContainerStart(context.Background(), d.ContainerName, container.StartOptions{})
}

func (d *BaseDriver) Stop() error {
	if d.Client == nil {
		return fmt.Errorf("docker client is not available")
	}
	timeout := 10
	return d.Client.ContainerStop(context.Background(), d.ContainerName, container.StopOptions{Timeout: &timeout})
}

func (d *BaseDriver) Restart() error {
	if err := d.Stop(); err != nil {
		return err
	}
	return d.Start()
}

func (d *BaseDriver) Uninstall(purgeData bool) error {
	if d.Client != nil {
		_ = d.Client.ContainerRemove(context.Background(), d.ContainerName, container.RemoveOptions{Force: true})
	}
	if purgeData {
		return os.RemoveAll(d.DataDir)
	}
	return nil
}

func (d *BaseDriver) Upgrade(targetVersion string) error {
	if targetVersion == "" {
		return fmt.Errorf("target version is required")
	}
	return fmt.Errorf("upgrade is not implemented for %s", d.Name)
}

func (d *BaseDriver) Status() (*driver.ServiceStatus, error) {
	status := &driver.ServiceStatus{
		Name:      d.Name,
		Mode:      driver.ModeDocker,
		Status:    "unknown",
		DataPath:  d.DataDir,
		Ports:     append([]string(nil), d.Ports...),
		Network:   d.NetworkName,
		UpdatedAt: time.Now(),
		Details: map[string]string{
			"image": d.Image,
		},
	}
	if d.Client == nil {
		status.Status = "docker client unavailable"
		return status, nil
	}
	inspect, err := d.Client.ContainerInspect(context.Background(), d.ContainerName)
	if err != nil {
		status.Status = "not installed"
		return status, nil
	}
	status.Running = inspect.State != nil && inspect.State.Running
	if inspect.State != nil {
		status.Status = inspect.State.Status
	}
	return status, nil
}

func (d *BaseDriver) recreateWithImage(targetVersion string, specFn func() (*container.Config, *container.HostConfig, error)) error {
	if d.Client == nil {
		return fmt.Errorf("docker client is not available")
	}
	if targetVersion == "" {
		return fmt.Errorf("target version is required")
	}
	d.Image = replaceImageTag(d.Image, targetVersion)
	if err := d.EnsureReady(context.Background()); err != nil {
		return err
	}
	timeout := 10
	_ = d.Client.ContainerStop(context.Background(), d.ContainerName, container.StopOptions{Timeout: &timeout})
	_ = d.Client.ContainerRemove(context.Background(), d.ContainerName, container.RemoveOptions{Force: true})
	cfg, hostCfg, err := specFn()
	if err != nil {
		return err
	}
	resp, err := d.Client.ContainerCreate(context.Background(), cfg, hostCfg, nil, nil, d.ContainerName)
	if err != nil {
		return err
	}
	return d.Client.ContainerStart(context.Background(), resp.ID, container.StartOptions{})
}

func replaceImageTag(image, targetVersion string) string {
	lastColon := strings.LastIndex(image, ":")
	lastSlash := strings.LastIndex(image, "/")
	if lastColon > lastSlash {
		return image[:lastColon+1] + targetVersion
	}
	return image + ":" + targetVersion
}
