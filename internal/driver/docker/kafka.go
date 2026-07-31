package docker

import (
	"fmt"
	"path/filepath"
	"time"

	"OpsVault/pkg/credutil"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

type KafkaDriver struct {
	*BaseDriver
}

func NewKafkaDriver(cli DockerClient, cfg *viper.Viper) *KafkaDriver {
	port := cfg.GetInt("kafka.port")
	if port == 0 {
		port = 9092
	}
	ctrlPort := cfg.GetInt("kafka.controller_port")
	if ctrlPort == 0 {
		ctrlPort = 9093
	}
	image := cfg.GetString("kafka.image")
	if image == "" {
		image = "apache/kafka:3.7.0"
	}
	base := NewBaseDriver("kafka", cli.Raw(), cfg, image, []string{
		fmt.Sprintf("%d:9092", port),
		fmt.Sprintf("%d:9093", ctrlPort),
	})
	drv := &KafkaDriver{BaseDriver: base}
	return drv
}

func (d *KafkaDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *KafkaDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	portClient := nat.Port("9092/tcp")
	portCtrl := nat.Port("9093/tcp")

	hostPort := d.Config.GetString("kafka.port")
	if hostPort == "" {
		hostPort = "9092"
	}
	hostCtrlPort := d.Config.GetString("kafka.controller_port")
	if hostCtrlPort == "" {
		hostCtrlPort = "9093"
	}

	bindIP := d.BindIP
	if bindIP == "" || bindIP == "0.0.0.0" {
		bindIP = "127.0.0.1"
	}

	env := []string{
		"KAFKA_NODE_ID=1",
		"KAFKA_PROCESS_ROLES=broker,controller",
		"KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093",
		fmt.Sprintf("KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://%s:%s", bindIP, hostPort),
		"KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER",
		"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT",
		"KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093",
		"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1",
		"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1",
		"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1",
		"KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=0",
		"KAFKA_LOG_DIRS=/var/lib/kafka/data",
	}

	healthCmd := "/opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server 127.0.0.1:9092 >/dev/null 2>&1"

	return &container.Config{
			Image: d.Image,
			Env:   env,
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", healthCmd},
				Interval:    15 * time.Second,
				Timeout:     8 * time.Second,
				StartPeriod: 25 * time.Second,
				Retries:     12,
			},
		}, &container.HostConfig{
			Binds: []string{
				toDockerBind(filepath.Join(d.DataDir, "data"), "/var/lib/kafka/data"),
			},
			PortBindings: nat.PortMap{
				portClient: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostPort}},
				portCtrl:   []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostCtrlPort}},
			},
		}, nil
}

func (d *KafkaDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

func (d *KafkaDriver) Version() string {
	return d.Image
}

func (d *KafkaDriver) GetCredentials() []credutil.Credential {
	port := d.Config.GetString("kafka.port")
	if port == "" {
		port = "9092"
	}
	ctrlPort := d.Config.GetString("kafka.controller_port")
	if ctrlPort == "" {
		ctrlPort = "9093"
	}
	return []credutil.Credential{
		{Label: "Bootstrap Server", Value: fmt.Sprintf("localhost:%s", port)},
		{Label: "Controller Server", Value: fmt.Sprintf("localhost:%s", ctrlPort)},
		{Label: "部署模式", Value: "KRaft Single Node"},
	}
}
