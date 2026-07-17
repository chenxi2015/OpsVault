package ansible

import (
	"testing"
)

func TestParseDoctorOutput(t *testing.T) {
	rawOutput := `192.168.1.101 | CHANGED | rc=0 >>
===UPTIME===
 08:14:02 up 10 days,  2:30,  1 user,  load average: 0.05, 0.04, 0.01
===FREE===
              total        used        free      shared  buff/cache   available
Mem:           3789         800        1000          10        1989        2700
Swap:          2047           0        2047
===DF===
Filesystem      Size  Used Avail Use% Mounted on
/dev/sda1        40G   15G   25G  38% /
===SERVICES===
active
inactive
running
192.168.1.102 | UNREACHABLE | rc=0 >>
Connection refused
`

	results := ParseDoctorOutput(rawOutput)
	if len(results) != 2 {
		t.Fatalf("expected 2 host results, got %d", len(results))
	}

	h1 := results[0]
	if h1.IP != "192.168.1.101" {
		t.Errorf("expected IP to be 192.168.1.101, got %s", h1.IP)
	}
	if h1.Status != "CHANGED" {
		t.Errorf("expected Status to be CHANGED, got %s", h1.Status)
	}
	if h1.Uptime != "up 10 days" {
		t.Errorf("expected Uptime to be 'up 10 days', got %s", h1.Uptime)
	}
	if h1.MemTotal != "3789MB" || h1.MemUsed != "800MB" {
		t.Errorf("expected MemTotal: 3789MB, MemUsed: 800MB, got %s, %s", h1.MemTotal, h1.MemUsed)
	}
	if h1.DiskSize != "40G" || h1.DiskUsed != "15G" || h1.DiskUsePct != "38%" {
		t.Errorf("expected DiskSize: 40G, DiskUsed: 15G, DiskUsePct: 38%%, got %s, %s, %s", h1.DiskSize, h1.DiskUsed, h1.DiskUsePct)
	}
	if h1.DockerState != "active" || h1.NginxState != "inactive" || h1.MinIOState != "running" {
		t.Errorf("expected Docker: active, Nginx: inactive, MinIO: running, got %s, %s, %s", h1.DockerState, h1.NginxState, h1.MinIOState)
	}

	h2 := results[1]
	if h2.IP != "192.168.1.102" {
		t.Errorf("expected IP to be 192.168.1.102, got %s", h2.IP)
	}
	if h2.Status != "UNREACHABLE" {
		t.Errorf("expected Status to be UNREACHABLE, got %s", h2.Status)
	}
}

func TestParsePingOutput(t *testing.T) {
	rawOutput := `192.168.1.101 | SUCCESS => {
    "changed": false,
    "ping": "pong"
}
192.168.1.102 | FAILED! => {
    "changed": false,
    "msg": "The module interpreter '/usr/local/python3.9' was not found."
}
192.168.1.103 | UNREACHABLE! => {
    "changed": false,
    "msg": "Failed to connect to host via ssh"
}
`

	results := ParsePingOutput(rawOutput)
	if len(results) != 3 {
		t.Fatalf("expected 3 host results, got %d", len(results))
	}

	h1 := results[0]
	if h1.IP != "192.168.1.101" || h1.Status != "SUCCESS" || h1.Message != "pong" {
		t.Errorf("unexpected success host 1 result: %+v", h1)
	}

	h2 := results[1]
	if h2.IP != "192.168.1.102" || h2.Status != "FAILED" || h2.Message != "The module interpreter '/usr/local/python3.9' was not found." {
		t.Errorf("unexpected failed host 2 result: %+v", h2)
	}

	h3 := results[2]
	if h3.IP != "192.168.1.103" || h3.Status != "UNREACHABLE" || h3.Message != "Failed to connect to host via ssh" {
		t.Errorf("unexpected unreachable host 3 result: %+v", h3)
	}
}

func TestParsePingOutputWithNoise(t *testing.T) {
	rawOutput := `192.168.1.101 | SUCCESS => {
    "changed": false,
    "ping": "pong"
}
[ERROR]: Task failed: Action failed: some random error message here
192.168.1.102 | FAILED! => {
    "changed": false,
    "msg": "The module interpreter '/usr/local/python3.9' was not found."
}
`
	results := ParsePingOutput(rawOutput)
	if len(results) != 2 {
		t.Fatalf("expected 2 host results, got %d", len(results))
	}

	h1 := results[0]
	if h1.IP != "192.168.1.101" || h1.Status != "SUCCESS" || h1.Message != "pong" {
		t.Errorf("unexpected success host result with noise: %+v", h1)
	}

	h2 := results[1]
	if h2.IP != "192.168.1.102" || h2.Status != "FAILED" || h2.Message != "The module interpreter '/usr/local/python3.9' was not found." {
		t.Errorf("unexpected failed host result with noise: %+v", h2)
	}
}


