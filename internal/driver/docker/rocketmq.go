package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

const rocketMQToolsPath = "/home/rocketmq/rocketmq/bin/tools.sh"

type RocketMQDriver struct {
	*BaseDriver
}

func NewRocketMQDriver(cli DockerClient, cfg *viper.Viper) *RocketMQDriver {
	namesrvPort := cfg.GetInt("rocketmq.namesrv_port")
	if namesrvPort == 0 {
		namesrvPort = 9876
	}
	brokerPort := cfg.GetInt("rocketmq.broker_port")
	if brokerPort == 0 {
		brokerPort = 10911
	}
	image := cfg.GetString("rocketmq.image")
	if image == "" {
		image = "apache/rocketmq:5.3.0"
	}
	base := NewBaseDriver("rocketmq", cli.Raw(), cfg, image, []string{fmt.Sprintf("%d:9876", namesrvPort), fmt.Sprintf("%d:10911", brokerPort)})
	drv := &RocketMQDriver{BaseDriver: base}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

func (d *RocketMQDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *RocketMQDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	portNameSrv := nat.Port("9876/tcp")
	portBroker := nat.Port("10911/tcp")
	namesrvPort := d.Config.GetString("rocketmq.namesrv_port")
	if namesrvPort == "" {
		namesrvPort = "9876"
	}
	brokerPort := d.Config.GetString("rocketmq.broker_port")
	if brokerPort == "" {
		brokerPort = "10911"
	}

	script := `
set -e
nohup sh mqnamesrv >/home/rocketmq/logs/namesrv.log 2>&1 &
sleep 5
exec sh mqbroker -n 127.0.0.1:9876 -c /home/rocketmq/rocketmq-opsvault.conf
`
	return &container.Config{
			Image: d.Image,
			Cmd:   []string{"sh", "-c", script},
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", "sh " + rocketMQToolsPath + " clusterList -n 127.0.0.1:9876 >/dev/null 2>&1"},
				Interval:    15 * time.Second,
				Timeout:     8 * time.Second,
				StartPeriod: 25 * time.Second,
				Retries:     12,
			},
		}, &container.HostConfig{
			Binds: []string{
				filepath.Join(d.DataDir, "data") + ":/home/rocketmq/store",
				filepath.Join(d.DataDir, "conf", "broker.conf") + ":/home/rocketmq/rocketmq-opsvault.conf",
			},
			PortBindings: nat.PortMap{
				portNameSrv: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: namesrvPort}},
				portBroker:  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: brokerPort}},
			},
		}, nil
}

func (d *RocketMQDriver) prepareConfig(confDir string) error {
	filePath := filepath.Join(confDir, "broker.conf")
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}
	content := `brokerClusterName=OpsVaultCluster
brokerName=broker-a
brokerId=0
deleteWhen=04
fileReservedTime=48
brokerRole=ASYNC_MASTER
flushDiskType=ASYNC_FLUSH
namesrvAddr=127.0.0.1:9876
listenPort=10911
autoCreateTopicEnable=true
storePathRootDir=/home/rocketmq/store
storePathCommitLog=/home/rocketmq/store/commitlog
storePathConsumeQueue=/home/rocketmq/store/consumequeue
storePathIndex=/home/rocketmq/store/index
`
	return os.WriteFile(filePath, []byte(content), 0o644)
}

func (d *RocketMQDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

func (d *RocketMQDriver) Version() string {
	return d.Image
}

func (d *RocketMQDriver) DLQStat() (map[string]string, error) {
	topicsOutput, err := d.execInContainer(d.ContainerName, []string{
		"sh", rocketMQToolsPath, "topicList", "-n", "127.0.0.1:9876",
	})
	if err != nil {
		return nil, err
	}

	topics := extractDLQTopics(topicsOutput)
	sort.Strings(topics)

	stats := map[string]string{
		"dlq_topics":      strconv.Itoa(len(topics)),
		"total_messages":  "0",
		"queried_broker":  d.ContainerName,
		"nameserver_addr": "127.0.0.1:9876",
	}
	total := 0
	for _, topic := range topics {
		topicOutput, err := d.execInContainer(d.ContainerName, []string{
			"sh", rocketMQToolsPath, "topicStatus", "-n", "127.0.0.1:9876", "-t", topic,
		})
		if err != nil {
			return nil, fmt.Errorf("query dlq topic %s: %w", topic, err)
		}
		count := parseRocketMQTopicBacklog(topicOutput)
		stats[topic] = strconv.Itoa(count)
		total += count
	}
	stats["total_messages"] = strconv.Itoa(total)
	return stats, nil
}

func extractDLQTopics(output string) []string {
	seen := map[string]struct{}{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "Topic:")
		if strings.HasPrefix(line, "%DLQ%") {
			seen[line] = struct{}{}
		}
	}
	topics := make([]string, 0, len(seen))
	for topic := range seen {
		topics = append(topics, topic)
	}
	return topics
}

func parseRocketMQTopicBacklog(output string) int {
	var total int
	re := regexp.MustCompile(`\s+`)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Broker") || strings.Contains(line, "#QueueId") {
			continue
		}
		fields := re.Split(line, -1)
		if len(fields) < 4 {
			continue
		}
		minOffset, errMin := strconv.ParseInt(fields[2], 10, 64)
		maxOffset, errMax := strconv.ParseInt(fields[3], 10, 64)
		if errMin != nil || errMax != nil || maxOffset < minOffset {
			continue
		}
		total += int(maxOffset - minOffset)
	}
	return total
}
