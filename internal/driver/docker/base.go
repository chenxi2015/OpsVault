package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"OpsVault/internal/driver"
	"OpsVault/pkg/dockercli"
	"OpsVault/pkg/fileutil"
	"OpsVault/pkg/logger"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spf13/viper"
)

type BaseDriver struct {
	Name           string
	Client         *client.Client
	Config         *viper.Viper
	Image          string
	ContainerName  string
	DataDir        string
	Ports          []string
	NetworkName    string
	PollInterval   time.Duration
	StartupTimeout time.Duration

	execInContainer func(containerName string, cmd []string) (string, error)
}

func NewBaseDriver(name string, cli *client.Client, cfg *viper.Viper, image string, ports []string) *BaseDriver {
	dataRoot := cfg.GetString("docker.data_root")
	driver := &BaseDriver{
		Name:           name,
		Client:         cli,
		Config:         cfg,
		Image:          image,
		ContainerName:  "opsvault-" + name,
		DataDir:        filepath.Join(dataRoot, name),
		Ports:          ports,
		NetworkName:    cfg.GetString("docker.network_name"),
		PollInterval:   2 * time.Second,
		StartupTimeout: 2 * time.Minute,
	}
	driver.execInContainer = driver.defaultExecInContainer
	return driver
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
		if inspect.State.Health != nil {
			status.Details["health"] = inspect.State.Health.Status
			if inspect.State.Health.Status != "" {
				status.Status = inspect.State.Health.Status
			}
		}
		if inspect.State.Error != "" {
			status.Details["error"] = inspect.State.Error
		}
	}
	return status, nil
}

func (d *BaseDriver) installWithSpec(specFn func() (*container.Config, *container.HostConfig, error)) error {
	if d.Client == nil {
		return fmt.Errorf("docker client is not available")
	}
	ctx := context.Background()
	if err := d.EnsureReady(ctx); err != nil {
		return err
	}
	if err := d.pullImage(ctx, d.Image); err != nil {
		return err
	}
	cfg, hostCfg, err := specFn()
	if err != nil {
		return err
	}
	resp, err := d.Client.ContainerCreate(ctx, cfg, hostCfg, nil, nil, d.ContainerName)
	if err != nil {
		return err
	}
	createdID := resp.ID
	if err := d.Client.ContainerStart(ctx, createdID, container.StartOptions{}); err != nil {
		_ = d.Client.ContainerRemove(ctx, d.ContainerName, container.RemoveOptions{Force: true})
		return err
	}
	if err := d.waitForHealthy(ctx, d.ContainerName); err != nil {
		_ = d.Client.ContainerRemove(ctx, d.ContainerName, container.RemoveOptions{Force: true})
		return err
	}
	return nil
}

func (d *BaseDriver) recreateWithImage(targetVersion string, specFn func() (*container.Config, *container.HostConfig, error)) error {
	if d.Client == nil {
		return fmt.Errorf("docker client is not available")
	}
	if targetVersion == "" {
		return fmt.Errorf("target version is required")
	}

	ctx := context.Background()
	if err := d.EnsureReady(ctx); err != nil {
		return err
	}

	oldImage := d.Image
	newImage := replaceImageTag(oldImage, targetVersion)
	if err := d.pullImage(ctx, newImage); err != nil {
		return err
	}

	backupName := ""
	if _, err := d.Client.ContainerInspect(ctx, d.ContainerName); err == nil {
		timeout := 10
		_ = d.Client.ContainerStop(ctx, d.ContainerName, container.StopOptions{Timeout: &timeout})
		backupName = fmt.Sprintf("%s-backup-%d", d.ContainerName, time.Now().Unix())
		if err := d.Client.ContainerRename(ctx, d.ContainerName, backupName); err != nil {
			return err
		}
	}

	d.Image = newImage
	cfg, hostCfg, err := specFn()
	if err != nil {
		d.Image = oldImage
		if backupName != "" {
			_ = d.Client.ContainerRename(ctx, backupName, d.ContainerName)
		}
		return err
	}

	resp, err := d.Client.ContainerCreate(ctx, cfg, hostCfg, nil, nil, d.ContainerName)
	if err != nil {
		d.Image = oldImage
		if backupName != "" {
			_ = d.Client.ContainerRename(ctx, backupName, d.ContainerName)
		}
		return err
	}

	if err := d.Client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		_ = d.Client.ContainerRemove(ctx, d.ContainerName, container.RemoveOptions{Force: true})
		d.Image = oldImage
		return d.restoreBackup(ctx, backupName)
	}

	if err := d.waitForHealthy(ctx, d.ContainerName); err != nil {
		_ = d.Client.ContainerRemove(ctx, d.ContainerName, container.RemoveOptions{Force: true})
		d.Image = oldImage
		return d.restoreBackup(ctx, backupName, err)
	}

	if backupName != "" {
		_ = d.Client.ContainerRemove(ctx, backupName, container.RemoveOptions{Force: true})
	}
	return nil
}

func (d *BaseDriver) restoreBackup(ctx context.Context, backupName string, cause ...error) error {
	var original error
	if len(cause) > 0 {
		original = cause[0]
	}
	if backupName == "" {
		return original
	}
	if err := d.Client.ContainerRename(ctx, backupName, d.ContainerName); err != nil {
		if original != nil {
			return fmt.Errorf("%w; restore backup rename failed: %v", original, err)
		}
		return err
	}
	if err := d.Client.ContainerStart(ctx, d.ContainerName, container.StartOptions{}); err != nil {
		if original != nil {
			return fmt.Errorf("%w; restore backup start failed: %v", original, err)
		}
		return err
	}
	return original
}

func (d *BaseDriver) pullImage(ctx context.Context, ref string) error {
	if d.Client == nil {
		return fmt.Errorf("docker client is not available")
	}
	reader, err := d.Client.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return err
	}
	summary, err := collectPullProgress(reader)
	if err != nil {
		return err
	}
	if summary != "" {
		logger.Infof("docker pull %s: %s", ref, summary)
	}
	return nil
}

func collectPullProgress(reader io.ReadCloser) (string, error) {
	defer reader.Close()

	type pullStatus struct {
		Status   string `json:"status"`
		Error    string `json:"error"`
		ID       string `json:"id"`
		Progress string `json:"progress"`
	}

	decoder := json.NewDecoder(reader)
	last := ""
	for {
		var event pullStatus
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				return last, nil
			}
			return last, err
		}
		if event.Error != "" {
			return last, fmt.Errorf("image pull failed: %s", event.Error)
		}
		switch {
		case event.ID != "" && event.Status != "":
			last = fmt.Sprintf("%s: %s", event.ID, event.Status)
		case event.Progress != "" && event.Status != "":
			last = fmt.Sprintf("%s %s", event.Status, event.Progress)
		case event.Status != "":
			last = event.Status
		}
	}
}

func (d *BaseDriver) waitForHealthy(ctx context.Context, containerName string) error {
	timeout := d.StartupTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	interval := d.PollInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}

	deadline := time.Now().Add(timeout)
	for {
		inspect, err := d.Client.ContainerInspect(ctx, containerName)
		if err != nil {
			return err
		}
		if inspect.State == nil {
			return fmt.Errorf("container %s has no state", containerName)
		}
		if inspect.State.Error != "" {
			return fmt.Errorf("container %s failed: %s", containerName, inspect.State.Error)
		}
		if !inspect.State.Running && inspect.State.Status != "created" {
			return fmt.Errorf("container %s is %s", containerName, inspect.State.Status)
		}
		if inspect.State.Health == nil {
			if inspect.State.Running {
				return nil
			}
		} else {
			switch inspect.State.Health.Status {
			case "healthy":
				return nil
			case "unhealthy":
				return fmt.Errorf("container %s is unhealthy", containerName)
			}
		}
		if time.Now().After(deadline) {
			if inspect.State.Health != nil {
				return fmt.Errorf("container %s health check timed out: %s", containerName, inspect.State.Health.Status)
			}
			return fmt.Errorf("container %s startup timed out", containerName)
		}
		time.Sleep(interval)
	}
}

func (d *BaseDriver) defaultExecInContainer(containerName string, cmd []string) (string, error) {
	if d.Client == nil {
		return "", fmt.Errorf("docker client is not available")
	}

	ctx := context.Background()
	created, err := d.Client.ContainerExecCreate(ctx, containerName, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	})
	if err != nil {
		return "", err
	}

	attached, err := d.Client.ContainerExecAttach(ctx, created.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", err
	}
	defer attached.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, attached.Reader); err != nil {
		return "", err
	}

	inspect, err := d.Client.ContainerExecInspect(ctx, created.ID)
	if err != nil {
		return "", err
	}
	if inspect.ExitCode != 0 {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = fmt.Sprintf("exit code %d", inspect.ExitCode)
		}
		return "", fmt.Errorf("exec failed: %s", msg)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func replaceImageTag(image, targetVersion string) string {
	lastColon := strings.LastIndex(image, ":")
	lastSlash := strings.LastIndex(image, "/")
	if lastColon > lastSlash {
		return image[:lastColon+1] + targetVersion
	}
	return image + ":" + targetVersion
}

func (d *BaseDriver) TailLogs(lines int) (string, error) {
	if d.Client == nil {
		return "", fmt.Errorf("docker client is not available")
	}
	ctx := context.Background()
	reader, err := d.Client.ContainerLogs(ctx, d.ContainerName, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(lines),
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, reader)
	if err != nil {
		return "", err
	}
	return stdout.String() + stderr.String(), nil
}
