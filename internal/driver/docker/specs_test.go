package docker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/networks"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/networks/create"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"opsvault-net-id","Warning":""}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/stop"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/containers/opsvault-mysql"):
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
	if len(requests) != 6 {
		t.Fatalf("requests = %#v", requests)
	}
	wantRequests := []string{
		"GET /v1.47/networks",
		"POST /v1.47/networks/create",
		"POST /v1.47/containers/opsvault-mysql/stop",
		"DELETE /v1.47/containers/opsvault-mysql",
		"POST /v1.47/containers/create",
		"POST /v1.47/containers/new-container-id/start",
	}
	for i := range wantRequests {
		if requests[i] != wantRequests[i] {
			t.Fatalf("requests[%d] = %q, want %q; all requests=%#v", i, requests[i], wantRequests[i], requests)
		}
	}
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
