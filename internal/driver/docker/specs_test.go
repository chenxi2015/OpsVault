package docker

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"OpsVault/internal/driver"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

func testConfigWithRoot(dataRoot string) *viper.Viper {
	cfg := viper.New()
	cfg.Set("docker.data_root", dataRoot)
	cfg.Set("docker.network_name", "opsvault-net")
	cfg.Set("docker.images.mysql", "mysql:8.0")
	cfg.Set("docker.images.redis", "redis:7-alpine")
	cfg.Set("docker.images.rabbitmq", "rabbitmq:3-management")
	cfg.Set("docker.images.rocketmq", "apache/rocketmq:5.3.0")
	cfg.Set("docker.images.postgres", "postgres:15")
	return cfg
}

func TestMySQLContainerSpec(t *testing.T) {
	drv := NewMySQLDriver(WrapClient(nil), testConfigWithRoot("/data/opsvault"), "secret")
	cfg, host, err := drv.containerSpec()
	if err != nil {
		t.Fatalf("containerSpec: %v", err)
	}
	if cfg.Image != "mysql:8.0" {
		t.Fatalf("image = %q", cfg.Image)
	}
	if len(cfg.Env) != 1 || cfg.Env[0] != "MYSQL_ROOT_PASSWORD=secret" {
		t.Fatalf("env = %#v", cfg.Env)
	}
	if len(host.Binds) != 1 || host.Binds[0] != filepath.Join("/data/opsvault", "mysql")+":/var/lib/mysql" {
		t.Fatalf("binds = %#v", host.Binds)
	}
	port := nat.Port("3306/tcp")
	if host.PortBindings[port][0].HostPort != "3306" {
		t.Fatalf("port binding = %#v", host.PortBindings[port])
	}
}

func TestRedisContainerSpecWithPassword(t *testing.T) {
	drv := NewRedisDriver(WrapClient(nil), testConfigWithRoot("/data/opsvault"), "redispass")
	cfg, host, err := drv.containerSpec()
	if err != nil {
		t.Fatalf("containerSpec: %v", err)
	}
	wantCmd := []string{"redis-server", "--appendonly", "yes", "--requirepass", "redispass"}
	if len(cfg.Cmd) != len(wantCmd) {
		t.Fatalf("cmd = %#v", cfg.Cmd)
	}
	for i := range wantCmd {
		if cfg.Cmd[i] != wantCmd[i] {
			t.Fatalf("cmd[%d] = %q, want %q", i, cfg.Cmd[i], wantCmd[i])
		}
	}
	if host.Binds[0] != filepath.Join("/data/opsvault", "redis")+":/data" {
		t.Fatalf("binds = %#v", host.Binds)
	}
}

func TestRabbitMQContainerSpec(t *testing.T) {
	drv := NewRabbitMQDriver(WrapClient(nil), testConfigWithRoot("/data/opsvault"), "admin", "123456")
	cfg, host, err := drv.containerSpec()
	if err != nil {
		t.Fatalf("containerSpec: %v", err)
	}
	if cfg.Env[0] != "RABBITMQ_DEFAULT_USER=admin" || cfg.Env[1] != "RABBITMQ_DEFAULT_PASS=123456" {
		t.Fatalf("env = %#v", cfg.Env)
	}
	if host.PortBindings[nat.Port("15672/tcp")][0].HostPort != "15672" {
		t.Fatalf("port binding = %#v", host.PortBindings[nat.Port("15672/tcp")])
	}
}

func TestMySQLUpgradeRecreatesContainerWithNewTag(t *testing.T) {
	var requests []string
	var createImage string
	inspectCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/networks"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/networks/create"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"opsvault-net-id","Warning":""}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/images/create"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, "{\"status\":\"Pulling\"}\n")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/containers/opsvault-mysql/json"):
			inspectCount++
			w.Header().Set("Content-Type", "application/json")
			if inspectCount == 1 {
				_, _ = io.WriteString(w, `{"Id":"old-container","State":{"Status":"running","Running":true,"Health":{"Status":"healthy"}}}`)
			} else {
				_, _ = io.WriteString(w, `{"Id":"new-container-id","State":{"Status":"running","Running":true,"Health":{"Status":"healthy"}}}`)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/stop"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/rename"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/create"):
			var cfg container.Config
			if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			createImage = cfg.Image
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"new-container-id","Warnings":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/containers/opsvault-mysql-backup-"):
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Transport = &rewriteTransport{baseURL: server.URL, rt: httpClient.Transport}
	cli, err := client.NewClientWithOpts(
		client.WithHost(server.URL),
		client.WithVersion("1.47"),
		client.WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	drv := NewMySQLDriver(WrapClient(cli), testConfigWithRoot(t.TempDir()), "secret")
	if err := drv.Upgrade("8.4"); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	if createImage != "mysql:8.4" {
		t.Fatalf("create image = %q, want %q", createImage, "mysql:8.4")
	}
	if len(requests) != 10 {
		t.Fatalf("requests = %#v", requests)
	}
	wantRequests := []string{
		"GET /v1.47/networks",
		"POST /v1.47/networks/create",
		"POST /v1.47/images/create",
		"GET /v1.47/containers/opsvault-mysql/json",
		"POST /v1.47/containers/opsvault-mysql/stop",
		"POST /v1.47/containers/opsvault-mysql/rename",
		"POST /v1.47/containers/create",
		"POST /v1.47/containers/new-container-id/start",
	}
	for i := range wantRequests {
		if requests[i] != wantRequests[i] {
			t.Fatalf("requests[%d] = %q, want %q; all requests=%#v", i, requests[i], wantRequests[i], requests)
		}
	}
	if !strings.Contains(requests[8], "/containers/opsvault-mysql/json") {
		t.Fatalf("requests[8] = %q, want health inspect", requests[8])
	}
	if !strings.Contains(requests[9], "/containers/opsvault-mysql-backup-") {
		t.Fatalf("requests[9] = %q, want backup cleanup", requests[9])
	}
}

func TestMySQLUpgradeRestoresBackupWhenNewContainerFails(t *testing.T) {
	var requests []string
	inspectCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/networks"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/networks/create"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"opsvault-net-id","Warning":""}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/images/create"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, "{\"status\":\"Pulling\"}\n")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/containers/opsvault-mysql/json"):
			w.Header().Set("Content-Type", "application/json")
			inspectCount++
			if inspectCount == 1 {
				_, _ = io.WriteString(w, `{"Id":"old-container","State":{"Status":"running","Running":true,"Health":{"Status":"healthy"}}}`)
			} else {
				_, _ = io.WriteString(w, `{"Id":"new-container-id","State":{"Status":"exited","Running":false,"Error":"new image failed"}}`)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/stop"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/rename"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/create"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"new-container-id","Warnings":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/new-container-id/start"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/containers/opsvault-mysql"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/opsvault-mysql/start"):
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	cli := testDockerClient(t, server)
	drv := NewMySQLDriver(WrapClient(cli), testConfigWithRoot(t.TempDir()), "secret")
	drv.PollInterval = time.Millisecond
	drv.StartupTimeout = 25 * time.Millisecond

	err := drv.Upgrade("8.4")
	if err == nil {
		t.Fatal("Upgrade succeeded, want error")
	}
	if !strings.Contains(err.Error(), "new image failed") {
		t.Fatalf("Upgrade error = %v, want failure from new image", err)
	}

	foundRestoreRename := false
	foundRestoreStart := false
	for _, request := range requests {
		if strings.Contains(request, "/containers/opsvault-mysql-backup-") && strings.Contains(request, "/rename") {
			foundRestoreRename = true
		}
		if strings.Contains(request, "/containers/opsvault-mysql/start") {
			foundRestoreStart = true
		}
	}
	if !foundRestoreRename {
		t.Fatalf("requests missing restore rename: %#v", requests)
	}
	if !foundRestoreStart {
		t.Fatalf("requests missing restore start: %#v", requests)
	}
}

func TestMySQLInstallPullsImageStartsContainerAndWaitsHealthy(t *testing.T) {
	var requests []string
	var createImage string
	var bindValue string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/networks"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/networks/create"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"opsvault-net-id","Warning":""}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/images/create"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, "{\"status\":\"Pulling from library/mysql\"}\n{\"status\":\"Download complete\"}\n")
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/create"):
			var payload struct {
				*container.Config
				HostConfig *container.HostConfig `json:"HostConfig"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			createImage = payload.Config.Image
			if len(payload.HostConfig.Binds) > 0 {
				bindValue = payload.HostConfig.Binds[0]
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"new-container-id","Warnings":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/json"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"Id":"new-container-id","State":{"Status":"running","Running":true,"Health":{"Status":"healthy"}}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	cli := testDockerClient(t, server)
	drv := NewMySQLDriver(WrapClient(cli), testConfigWithRoot(t.TempDir()), "secret")
	drv.PollInterval = time.Millisecond
	drv.StartupTimeout = 25 * time.Millisecond

	if err := drv.Install(); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if createImage != "mysql:8.0" {
		t.Fatalf("create image = %q, want %q", createImage, "mysql:8.0")
	}
	if !strings.HasSuffix(bindValue, ":/var/lib/mysql") {
		t.Fatalf("bind value = %q", bindValue)
	}
	wantRequests := []string{
		"GET /v1.47/networks",
		"POST /v1.47/networks/create",
		"POST /v1.47/images/create",
		"POST /v1.47/containers/create",
		"POST /v1.47/containers/new-container-id/start",
		"GET /v1.47/containers/opsvault-mysql/json",
	}
	for i := range wantRequests {
		if requests[i] != wantRequests[i] {
			t.Fatalf("requests[%d] = %q, want %q; all=%#v", i, requests[i], wantRequests[i], requests)
		}
	}
}

func TestMySQLInstallRollsBackContainerWhenHealthCheckFails(t *testing.T) {
	var removed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/networks"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/networks/create"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"opsvault-net-id","Warning":""}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/images/create"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, "{\"status\":\"Pulling\"}\n")
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/create"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"new-container-id","Warnings":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/json"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"Id":"new-container-id","State":{"Status":"exited","Running":false,"Error":"boot failed"}}`)
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/containers/opsvault-mysql"):
			removed = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	cli := testDockerClient(t, server)
	drv := NewMySQLDriver(WrapClient(cli), testConfigWithRoot(t.TempDir()), "secret")
	drv.PollInterval = time.Millisecond
	drv.StartupTimeout = 25 * time.Millisecond

	err := drv.Install()
	if err == nil {
		t.Fatal("Install succeeded, want error")
	}
	if !strings.Contains(err.Error(), "boot failed") {
		t.Fatalf("Install error = %v, want boot failure", err)
	}
	if !removed {
		t.Fatal("expected failed container to be removed")
	}
}

func TestRocketMQDLQStatParsesExecOutput(t *testing.T) {
	execCalls := 0
	drv := NewRocketMQDriver(WrapClient(nil), testConfigWithRoot(t.TempDir()))
	drv.execInContainer = func(_ string, cmd []string) (string, error) {
		execCalls++
		joined := strings.Join(cmd, " ")
		switch {
		case strings.Contains(joined, "topicList"):
			return "Topic:%DLQ%groupA\nTopic:%DLQ%groupB\nTopic:NormalTopic\n", nil
		case strings.Contains(joined, "topicStatus") && strings.Contains(joined, "%DLQ%groupA"):
			return "Broker Name                      #QueueId  #Min Offset           #Max Offset             #Last Updated\nbroker-a                         0         0                     3                       2026-07-02 12:00:00\nbroker-a                         1         5                     9                       2026-07-02 12:00:00\n", nil
		case strings.Contains(joined, "topicStatus") && strings.Contains(joined, "%DLQ%groupB"):
			return "Broker Name                      #QueueId  #Min Offset           #Max Offset             #Last Updated\nbroker-b                         0         7                     7                       2026-07-02 12:00:00\n", nil
		default:
			t.Fatalf("unexpected command: %s", joined)
			return "", nil
		}
	}

	stats, err := drv.DLQStat()
	if err != nil {
		t.Fatalf("DLQStat: %v", err)
	}
	if execCalls != 3 {
		t.Fatalf("execCalls = %d, want 3", execCalls)
	}
	if got := stats["dlq_topics"]; got != "2" {
		t.Fatalf("dlq_topics = %q, want 2", got)
	}
	if got := stats["total_messages"]; got != "7" {
		t.Fatalf("total_messages = %q, want 7", got)
	}
	if got := stats["%DLQ%groupA"]; got != "7" {
		t.Fatalf("groupA count = %q, want 7", got)
	}
	if got := stats["%DLQ%groupB"]; got != "0" {
		t.Fatalf("groupB count = %q, want 0", got)
	}
}

func TestCollectPullProgressSummarizesLastStatus(t *testing.T) {
	progress := bytes.NewBufferString("{\"status\":\"Pulling fs layer\"}\n{\"status\":\"Downloading\",\"progressDetail\":{\"current\":50,\"total\":100}}\n{\"status\":\"Download complete\"}\n")
	summary, err := collectPullProgress(io.NopCloser(progress))
	if err != nil {
		t.Fatalf("collectPullProgress: %v", err)
	}
	if summary != "Download complete" {
		t.Fatalf("summary = %q, want %q", summary, "Download complete")
	}
}

func testDockerClient(t *testing.T, server *httptest.Server) *client.Client {
	t.Helper()
	httpClient := server.Client()
	httpClient.Transport = &rewriteTransport{baseURL: server.URL, rt: httpClient.Transport}
	cli, err := client.NewClientWithOpts(
		client.WithHost(server.URL),
		client.WithVersion("1.47"),
		client.WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return cli
}

type rewriteTransport struct {
	baseURL string
	rt      http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.baseURL, "http://")
	return t.rt.RoundTrip(req)
}

func TestDockerLogReaderCapability(t *testing.T) {
	drv := NewMySQLDriver(WrapClient(nil), testConfigWithRoot("/data/opsvault"), "secret")
	if _, ok := interface{}(drv).(driver.LogReader); !ok {
		t.Fatalf("MySQLDriver does not implement driver.LogReader")
	}
}

