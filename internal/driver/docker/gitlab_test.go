package docker

import (
	"path/filepath"
	"testing"

	"github.com/docker/go-connections/nat"
)

func TestGitLabContainerSpec(t *testing.T) {
	cfg := testConfigWithRoot("/data/opsvault")
	cfg.Set("gitlab.image", "gitlab/gitlab-ce:latest")
	cfg.Set("gitlab.port", 8082)
	cfg.Set("gitlab.ssh_port", 2222)
	cfg.Set("gitlab.https_port", 8443)

	drv := NewGitLabDriver(WrapClient(nil), cfg)
	containerCfg, hostCfg, err := drv.containerSpec()
	if err != nil {
		t.Fatalf("containerSpec: %v", err)
	}

	if containerCfg.Image != "gitlab/gitlab-ce:latest" {
		t.Errorf("expected image gitlab/gitlab-ce:latest, got %s", containerCfg.Image)
	}

	expectedBinds := map[string]string{
		filepath.Join("/data/opsvault", "gitlab", "config"): "/etc/gitlab",
		filepath.Join("/data/opsvault", "gitlab", "logs"):   "/var/log/gitlab",
		filepath.Join("/data/opsvault", "gitlab", "data"):   "/var/opt/gitlab",
	}

	for hostPath, containerPath := range expectedBinds {
		expectedBind := hostPath + ":" + containerPath
		foundBind := false
		for _, bind := range hostCfg.Binds {
			if bind == expectedBind {
				foundBind = true
				break
			}
		}
		if !foundBind {
			t.Errorf("expected bind %s, not found in %v", expectedBind, hostCfg.Binds)
		}
	}

	port80 := nat.Port("80/tcp")
	if hostCfg.PortBindings[port80][0].HostPort != "8082" {
		t.Errorf("expected port 80 mapping to 8082, got %s", hostCfg.PortBindings[port80][0].HostPort)
	}

	port22 := nat.Port("22/tcp")
	if hostCfg.PortBindings[port22][0].HostPort != "2222" {
		t.Errorf("expected SSH port 22 mapping to 2222, got %s", hostCfg.PortBindings[port22][0].HostPort)
	}

	port443 := nat.Port("443/tcp")
	if hostCfg.PortBindings[port443][0].HostPort != "8443" {
		t.Errorf("expected HTTPS port 443 mapping to 8443, got %s", hostCfg.PortBindings[port443][0].HostPort)
	}
}
