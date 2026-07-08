package docker

import (
	"path/filepath"
	"testing"

	"github.com/docker/go-connections/nat"
)

func TestJenkinsContainerSpec(t *testing.T) {
	cfg := testConfigWithRoot("/data/opsvault")
	cfg.Set("jenkins.image", "jenkins/jenkins:lts")
	cfg.Set("jenkins.port", 8080)
	cfg.Set("jenkins.agent_port", 50000)

	drv := NewJenkinsDriver(WrapClient(nil), cfg)
	containerCfg, hostCfg, err := drv.containerSpec()
	if err != nil {
		t.Fatalf("containerSpec: %v", err)
	}

	if containerCfg.Image != "jenkins/jenkins:lts" {
		t.Errorf("expected image jenkins/jenkins:lts, got %s", containerCfg.Image)
	}

	expectedBind := filepath.Join("/data/opsvault", "jenkins", "data") + ":/var/jenkins_home"
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

	port8080 := nat.Port("8080/tcp")
	if hostCfg.PortBindings[port8080][0].HostPort != "8080" {
		t.Errorf("expected port 8080 mapping, got %s", hostCfg.PortBindings[port8080][0].HostPort)
	}

	port50000 := nat.Port("50000/tcp")
	if hostCfg.PortBindings[port50000][0].HostPort != "50000" {
		t.Errorf("expected agent port 50000 mapping, got %s", hostCfg.PortBindings[port50000][0].HostPort)
	}
}
