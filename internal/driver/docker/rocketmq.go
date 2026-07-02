package docker

import (
	"fmt"
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
	base := NewBaseDriver("rocketmq", cli.Raw(), cfg, cfg.GetString("docker.images.rocketmq"), []string{"9876:9876", "10911:10911"})
	return &RocketMQDriver{BaseDriver: base}
}

func (d *RocketMQDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *RocketMQDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	portNameSrv := nat.Port("9876/tcp")
	portBroker := nat.Port("10911/tcp")
	script := `
set -e
nohup sh mqnamesrv >/home/rocketmq/logs/namesrv.log 2>&1 &
sleep 5
exec sh mqbroker -n 127.0.0.1:9876 -c /home/rocketmq/rocketmq-opsvault.conf
`
	conf := `brokerClusterName=OpsVaultCluster
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
	return &container.Config{
			Image: d.Image,
			Cmd:   []string{"sh", "-c", "cat <<'EOF' >/home/rocketmq/rocketmq-opsvault.conf\n" + conf + "EOF\n" + script},
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", "sh " + rocketMQToolsPath + " clusterList -n 127.0.0.1:9876 >/dev/null 2>&1"},
				Interval:    15 * time.Second,
				Timeout:     8 * time.Second,
				StartPeriod: 25 * time.Second,
				Retries:     12,
			},
		}, &container.HostConfig{
			Binds: []string{d.DataDir + ":/home/rocketmq/store"},
			PortBindings: nat.PortMap{
				portNameSrv: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "9876"}},
				portBroker:  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "10911"}},
			},
		}, nil
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
