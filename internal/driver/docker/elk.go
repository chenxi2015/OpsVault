package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"OpsVault/internal/driver"
	"OpsVault/pkg/credutil"
	"OpsVault/pkg/dockercli"
	"OpsVault/pkg/fileutil"
	"OpsVault/pkg/logger"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

type ELKDriver struct {
	*BaseDriver
}

func NewELKDriver(cli DockerClient, cfg *viper.Viper) *ELKDriver {
	esImage := cfg.GetString("elk.elasticsearch_image")
	if esImage == "" {
		esImage = "elasticsearch:8.12.0"
	}
	esPort := cfg.GetInt("elk.elasticsearch_port")
	if esPort == 0 {
		esPort = 9200
	}
	kbPort := cfg.GetInt("elk.kibana_port")
	if kbPort == 0 {
		kbPort = 5601
	}
	lsPort := cfg.GetInt("elk.logstash_port")
	if lsPort == 0 {
		lsPort = 5044
	}

	base := NewBaseDriver("elk", cli.Raw(), cfg, esImage, []string{
		fmt.Sprintf("%d:9200", esPort),
		fmt.Sprintf("%d:5601", kbPort),
		fmt.Sprintf("%d:5044", lsPort),
	})
	drv := &ELKDriver{BaseDriver: base}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

func (d *ELKDriver) containerNames() (string, string, string) {
	es := dockercli.ResolveContainerName(d.Config, "elasticsearch")
	kb := dockercli.ResolveContainerName(d.Config, "kibana")
	ls := dockercli.ResolveContainerName(d.Config, "logstash")
	return es, kb, ls
}

func (d *ELKDriver) images() (string, string, string) {
	es := d.Config.GetString("elk.elasticsearch_image")
	if es == "" {
		es = "elasticsearch:8.12.0"
	}
	kb := d.Config.GetString("elk.kibana_image")
	if kb == "" {
		kb = "kibana:8.12.0"
	}
	ls := d.Config.GetString("elk.logstash_image")
	if ls == "" {
		ls = "logstash:8.12.0"
	}
	return es, kb, ls
}

func (d *ELKDriver) prepareConfig(confDir string) error {
	pipelineDir := filepath.Join(confDir, "pipeline")
	if err := fileutil.EnsureDir(pipelineDir, 0o755); err != nil {
		return err
	}

	esCfgPath := filepath.Join(confDir, "elasticsearch.yml")
	if _, err := os.Stat(esCfgPath); err != nil {
		esCfgContent := `cluster.name: "opsvault-elk"
network.host: 0.0.0.0
discovery.type: single-node
xpack.security.enabled: false
`
		if err := os.WriteFile(esCfgPath, []byte(esCfgContent), 0o644); err != nil {
			return err
		}
	}

	kbCfgPath := filepath.Join(confDir, "kibana.yml")
	if _, err := os.Stat(kbCfgPath); err != nil {
		kbCfgContent := `server.name: kibana
server.host: "0.0.0.0"
elasticsearch.hosts: [ "http://opsvault-elasticsearch:9200" ]
`
		esContainerName := dockercli.ResolveContainerName(d.Config, "elasticsearch")
		kbCfgContent = strings.ReplaceAll(kbCfgContent, "opsvault-elasticsearch", esContainerName)

		if err := os.WriteFile(kbCfgPath, []byte(kbCfgContent), 0o644); err != nil {
			return err
		}
	}

	lsCfgPath := filepath.Join(confDir, "logstash.yml")
	if _, err := os.Stat(lsCfgPath); err != nil {
		lsCfgContent := `http.host: "0.0.0.0"
pipeline.ordered: auto
`
		if err := os.WriteFile(lsCfgPath, []byte(lsCfgContent), 0o644); err != nil {
			return err
		}
	}

	lsPipelinePath := filepath.Join(pipelineDir, "logstash.conf")
	if _, err := os.Stat(lsPipelinePath); err != nil {
		lsPipelineContent := `input {
  beats {
    port => 5044
  }
  tcp {
    port => 5000
    codec => json
  }
}

output {
  elasticsearch {
    hosts => ["http://opsvault-elasticsearch:9200"]
    index => "opsvault-%{+YYYY.MM.dd}"
  }
}
`
		esContainerName := dockercli.ResolveContainerName(d.Config, "elasticsearch")
		lsPipelineContent = strings.ReplaceAll(lsPipelineContent, "opsvault-elasticsearch", esContainerName)

		if err := os.WriteFile(lsPipelinePath, []byte(lsPipelineContent), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func (d *ELKDriver) Install() error {
	ctx := context.Background()
	if err := d.EnsureReady(ctx); err != nil {
		return err
	}

	esDataDir := filepath.Join(d.DataDir, "data", "elasticsearch")
	lsDataDir := filepath.Join(d.DataDir, "data", "logstash")
	if err := fileutil.EnsureDir(esDataDir, 0o777); err != nil {
		return err
	}
	if err := os.Chmod(esDataDir, 0o777); err != nil {
		logger.Errorf("failed to chmod 777 elasticsearch data dir: %v", err)
	}
	if err := fileutil.EnsureDir(lsDataDir, 0o777); err != nil {
		return err
	}
	if err := os.Chmod(lsDataDir, 0o777); err != nil {
		logger.Errorf("failed to chmod 777 logstash data dir: %v", err)
	}

	esImg, kbImg, lsImg := d.images()

	logger.Infof("Pulling Elasticsearch image: %s", esImg)
	if err := d.pullImage(ctx, esImg); err != nil {
		return err
	}
	logger.Infof("Pulling Kibana image: %s", kbImg)
	if err := d.pullImage(ctx, kbImg); err != nil {
		return err
	}
	logger.Infof("Pulling Logstash image: %s", lsImg)
	if err := d.pullImage(ctx, lsImg); err != nil {
		return err
	}

	if err := d.installES(ctx); err != nil {
		return err
	}

	if err := d.installKB(ctx); err != nil {
		return err
	}

	if err := d.installLS(ctx); err != nil {
		return err
	}

	return nil
}

func (d *ELKDriver) installES(ctx context.Context) error {
	esName, _, _ := d.containerNames()
	esImg, _, _ := d.images()
	esPort := d.Config.GetString("elk.elasticsearch_port")
	if esPort == "" {
		esPort = "9200"
	}
	javaOpts := d.Config.GetString("elk.es_java_opts")
	if javaOpts == "" {
		javaOpts = "-Xms512m -Xmx512m"
	}

	cfg := &container.Config{
		Image: esImg,
		Env: []string{
			"discovery.type=single-node",
			"ES_JAVA_OPTS=" + javaOpts,
			"xpack.security.enabled=false",
		},
		Healthcheck: &container.HealthConfig{
			Test:        []string{"CMD-SHELL", "curl -s http://localhost:9200/_cat/health | grep -q green || curl -s http://localhost:9200/_cat/health | grep -q yellow"},
			Interval:    10 * time.Second,
			Timeout:     5 * time.Second,
			StartPeriod: 30 * time.Second,
			Retries:     10,
		},
	}

	hostCfg := &container.HostConfig{
		Binds: []string{
			filepath.Join(d.DataDir, "conf", "elasticsearch.yml") + ":/usr/share/elasticsearch/config/elasticsearch.yml",
			filepath.Join(d.DataDir, "data", "elasticsearch") + ":/usr/share/elasticsearch/data",
		},
		PortBindings: nat.PortMap{
			nat.Port("9200/tcp"): []nat.PortBinding{{HostIP: d.BindIP, HostPort: esPort}},
		},
	}
	d.applyResources(hostCfg)

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			d.NetworkName: {},
		},
	}

	_ = d.Client.ContainerRemove(ctx, esName, container.RemoveOptions{Force: true})

	resp, err := d.Client.ContainerCreate(ctx, cfg, hostCfg, networkingConfig, nil, esName)
	if err != nil {
		return fmt.Errorf("failed to create Elasticsearch container: %w", err)
	}

	if err := d.Client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start Elasticsearch container: %w", err)
	}

	logger.Infof("Waiting for Elasticsearch to become healthy...")
	if err := d.waitForHealthy(ctx, esName); err != nil {
		return fmt.Errorf("Elasticsearch healthcheck failed: %w", err)
	}

	return nil
}

func (d *ELKDriver) installKB(ctx context.Context) error {
	_, kbName, _ := d.containerNames()
	_, kbImg, _ := d.images()
	kbPort := d.Config.GetString("elk.kibana_port")
	if kbPort == "" {
		kbPort = "5601"
	}

	cfg := &container.Config{
		Image: kbImg,
		Healthcheck: &container.HealthConfig{
			Test:        []string{"CMD-SHELL", "curl -I -s http://localhost:5601 | grep -q 'HTTP/1.1'"},
			Interval:    10 * time.Second,
			Timeout:     5 * time.Second,
			StartPeriod: 30 * time.Second,
			Retries:     10,
		},
	}

	hostCfg := &container.HostConfig{
		Binds: []string{
			filepath.Join(d.DataDir, "conf", "kibana.yml") + ":/usr/share/kibana/config/kibana.yml",
		},
		PortBindings: nat.PortMap{
			nat.Port("5601/tcp"): []nat.PortBinding{{HostIP: d.BindIP, HostPort: kbPort}},
		},
	}
	d.applyResources(hostCfg)

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			d.NetworkName: {},
		},
	}

	_ = d.Client.ContainerRemove(ctx, kbName, container.RemoveOptions{Force: true})

	resp, err := d.Client.ContainerCreate(ctx, cfg, hostCfg, networkingConfig, nil, kbName)
	if err != nil {
		return fmt.Errorf("failed to create Kibana container: %w", err)
	}

	if err := d.Client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start Kibana container: %w", err)
	}

	return nil
}

func (d *ELKDriver) installLS(ctx context.Context) error {
	_, _, lsName := d.containerNames()
	_, _, lsImg := d.images()
	lsPort := d.Config.GetString("elk.logstash_port")
	if lsPort == "" {
		lsPort = "5044"
	}

	cfg := &container.Config{
		Image: lsImg,
		ExposedPorts: nat.PortSet{
			nat.Port("5044/tcp"): {},
		},
		Healthcheck: &container.HealthConfig{
			Test:        []string{"CMD-SHELL", "curl -s http://localhost:9600/_node | grep -q logstash"},
			Interval:    10 * time.Second,
			Timeout:     5 * time.Second,
			StartPeriod: 30 * time.Second,
			Retries:     10,
		},
	}

	hostCfg := &container.HostConfig{
		Binds: []string{
			filepath.Join(d.DataDir, "conf", "logstash.yml") + ":/usr/share/logstash/config/logstash.yml",
			filepath.Join(d.DataDir, "conf", "pipeline", "logstash.conf") + ":/usr/share/logstash/pipeline/logstash.conf",
			filepath.Join(d.DataDir, "data", "logstash") + ":/usr/share/logstash/data",
		},
		PortBindings: nat.PortMap{
			nat.Port("5044/tcp"): []nat.PortBinding{{HostIP: d.BindIP, HostPort: lsPort}},
		},
	}
	d.applyResources(hostCfg)

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			d.NetworkName: {},
		},
	}

	_ = d.Client.ContainerRemove(ctx, lsName, container.RemoveOptions{Force: true})

	resp, err := d.Client.ContainerCreate(ctx, cfg, hostCfg, networkingConfig, nil, lsName)
	if err != nil {
		return fmt.Errorf("failed to create Logstash container: %w", err)
	}

	if err := d.Client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start Logstash container: %w", err)
	}

	return nil
}

func (d *ELKDriver) Start() error {
	if err := d.checkAndInstallDocker(); err != nil {
		return err
	}
	if d.Client == nil {
		return fmt.Errorf("docker client is not available")
	}
	ctx := context.Background()
	es, kb, ls := d.containerNames()

	logger.Infof("Starting Elasticsearch...")
	if err := d.Client.ContainerStart(ctx, es, container.StartOptions{}); err != nil {
		return fmt.Errorf("start elasticsearch: %w", err)
	}

	logger.Infof("Starting Kibana...")
	if err := d.Client.ContainerStart(ctx, kb, container.StartOptions{}); err != nil {
		return fmt.Errorf("start kibana: %w", err)
	}

	logger.Infof("Starting Logstash...")
	if err := d.Client.ContainerStart(ctx, ls, container.StartOptions{}); err != nil {
		return fmt.Errorf("start logstash: %w", err)
	}

	return nil
}

func (d *ELKDriver) Stop() error {
	if err := d.checkAndInstallDocker(); err != nil {
		return err
	}
	if d.Client == nil {
		return fmt.Errorf("docker client is not available")
	}
	ctx := context.Background()
	es, kb, ls := d.containerNames()
	timeout := 10

	logger.Infof("Stopping Logstash...")
	_ = d.Client.ContainerStop(ctx, ls, container.StopOptions{Timeout: &timeout})

	logger.Infof("Stopping Kibana...")
	_ = d.Client.ContainerStop(ctx, kb, container.StopOptions{Timeout: &timeout})

	logger.Infof("Stopping Elasticsearch...")
	_ = d.Client.ContainerStop(ctx, es, container.StopOptions{Timeout: &timeout})

	return nil
}

func (d *ELKDriver) Restart() error {
	if err := d.Stop(); err != nil {
		return err
	}
	return d.Start()
}

func (d *ELKDriver) Uninstall(purgeData bool) error {
	_ = d.checkAndInstallDocker()
	if d.Client != nil {
		ctx := context.Background()
		es, kb, ls := d.containerNames()
		logger.Infof("Removing Logstash...")
		_ = d.Client.ContainerRemove(ctx, ls, container.RemoveOptions{Force: true})
		logger.Infof("Removing Kibana...")
		_ = d.Client.ContainerRemove(ctx, kb, container.RemoveOptions{Force: true})
		logger.Infof("Removing Elasticsearch...")
		_ = d.Client.ContainerRemove(ctx, es, container.RemoveOptions{Force: true})
	}
	if purgeData {
		return os.RemoveAll(d.DataDir)
	}
	return nil
}

func (d *ELKDriver) Upgrade(targetVersion string) error {
	if targetVersion == "" {
		return fmt.Errorf("target version is required")
	}
	ctx := context.Background()
	esImg := "elasticsearch:" + targetVersion
	kbImg := "kibana:" + targetVersion
	lsImg := "logstash:" + targetVersion

	logger.Infof("Pulling Elasticsearch upgraded image...")
	if err := d.pullImage(ctx, esImg); err != nil {
		return err
	}
	logger.Infof("Pulling Kibana upgraded image...")
	if err := d.pullImage(ctx, kbImg); err != nil {
		return err
	}
	logger.Infof("Pulling Logstash upgraded image...")
	if err := d.pullImage(ctx, lsImg); err != nil {
		return err
	}

	d.Config.Set("elk.elasticsearch_image", esImg)
	d.Config.Set("elk.kibana_image", kbImg)
	d.Config.Set("elk.logstash_image", lsImg)
	_ = d.Config.WriteConfig()

	_ = d.Stop()

	if err := d.installES(ctx); err != nil {
		return err
	}
	if err := d.installKB(ctx); err != nil {
		return err
	}
	if err := d.installLS(ctx); err != nil {
		return err
	}

	return nil
}

func (d *ELKDriver) Status() (*driver.ServiceStatus, error) {
	status := &driver.ServiceStatus{
		Name:      "elk",
		Mode:      driver.ModeDocker,
		Status:    "unknown",
		DataPath:  d.DataDir,
		Ports:     append([]string(nil), d.Ports...),
		Network:   d.NetworkName,
		UpdatedAt: time.Now(),
		Details:   make(map[string]string),
	}
	if d.Client == nil {
		status.Status = "docker client unavailable"
		return status, nil
	}
	ctx := context.Background()
	es, kb, ls := d.containerNames()

	inspectES, errES := d.Client.ContainerInspect(ctx, es)
	inspectKB, errKB := d.Client.ContainerInspect(ctx, kb)
	inspectLS, errLS := d.Client.ContainerInspect(ctx, ls)

	if errES != nil && errKB != nil && errLS != nil {
		status.Status = "not installed"
		return status, nil
	}

	esState := "not created"
	kbState := "not created"
	lsState := "not created"

	esRunning := false
	kbRunning := false
	lsRunning := false

	if errES == nil {
		esState = inspectES.State.Status
		esRunning = inspectES.State.Running
		if inspectES.State.Health != nil {
			status.Details["es_health"] = inspectES.State.Health.Status
		}
	}
	if errKB == nil {
		kbState = inspectKB.State.Status
		kbRunning = inspectKB.State.Running
		if inspectKB.State.Health != nil {
			status.Details["kibana_health"] = inspectKB.State.Health.Status
		}
	}
	if errLS == nil {
		lsState = inspectLS.State.Status
		lsRunning = inspectLS.State.Running
		if inspectLS.State.Health != nil {
			status.Details["logstash_health"] = inspectLS.State.Health.Status
		}
	}

	status.Details["elasticsearch"] = esState
	status.Details["kibana"] = kbState
	status.Details["logstash"] = lsState

	if esRunning && kbRunning && lsRunning {
		status.Running = true
		status.Status = "running"
	} else if esRunning || kbRunning || lsRunning {
		status.Running = true
		status.Status = "degraded"
	} else {
		status.Running = false
		status.Status = "stopped"
	}

	return status, nil
}

func (d *ELKDriver) GetCredentials() []credutil.Credential {
	esPort := d.Config.GetString("elk.elasticsearch_port")
	if esPort == "" {
		esPort = "9200"
	}
	kbPort := d.Config.GetString("elk.kibana_port")
	if kbPort == "" {
		kbPort = "5601"
	}
	lsPort := d.Config.GetString("elk.logstash_port")
	if lsPort == "" {
		lsPort = "5044"
	}
	return []credutil.Credential{
		{Label: "Elasticsearch REST API", Value: fmt.Sprintf("http://localhost:%s", esPort)},
		{Label: "Kibana Web UI", Value: fmt.Sprintf("http://localhost:%s", kbPort)},
		{Label: "Logstash Beats Port", Value: fmt.Sprintf("localhost:%s", lsPort)},
		{Label: "认证方式", Value: "无认证 (Security Disabled)"},
	}
}

func (d *ELKDriver) TailLogs(lines int) (string, error) {
	return d.TailComponentLogs("elasticsearch", lines)
}

func (d *ELKDriver) TailComponentLogs(component string, lines int) (string, error) {
	if err := d.checkAndInstallDocker(); err != nil {
		return "", err
	}
	if d.Client == nil {
		return "", fmt.Errorf("docker client is not available")
	}

	var containerName string
	es, kb, ls := d.containerNames()
	switch strings.ToLower(component) {
	case "elasticsearch", "es":
		containerName = es
	case "kibana", "kb":
		containerName = kb
	case "logstash", "ls":
		containerName = ls
	default:
		return "", fmt.Errorf("unknown ELK component: %s", component)
	}

	ctx := context.Background()
	reader, err := d.Client.ContainerLogs(ctx, containerName, container.LogsOptions{
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
