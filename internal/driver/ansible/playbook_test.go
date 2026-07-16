package ansible

import (
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestGeneratePlaybookFile(t *testing.T) {
	tempDir := "./test_playbook_tmp"
	defer os.RemoveAll(tempDir)

	vars := PlaybookVars{
		DataRoot:             "/data/opsvault",
		NetworkName:          "opsvault-net",
		CIDR:                 "172.28.0.0/16",
		NamePrefix:           "opsvault",
		MySQLImage:           "mysql:8.0",
		MySQLPort:            3306,
		MySQLRootPassword:    "rootpassword",
		RedisImage:           "redis:7-alpine",
		RedisPort:            6379,
		RedisPassword:        "redispass",
		RabbitMQImage:        "rabbitmq:3-management",
		RabbitMQPort:         5672,
		RabbitMQUIPort:       15672,
		RabbitMQUser:         "admin",
		RabbitMQPwd:          "adminpass",
		NginxVersion:         "1.26.2",
		NginxPCREVersion:     "8.45",
		NginxOpenSSLVersion:  "1.1.1w",
		NginxInstallPath:     "/usr/local/nginx",
		NginxSourceRoot:      "/usr/local/src/opsvault-nginx",
		NginxWWWRoot:         "/data/wwwroot",
		NginxSSLRoot:         "/data/ssl",
		NginxWWWLogsRoot:     "/data/wwwlogs",
		NginxRunUser:         "www",
		NginxRunGroup:        "www",
		NginxSystemdUnitPath: "/lib/systemd/system/nginx.service",
	}

	services := []string{"docker", "mysql", "redis", "rabbitmq", "nginx"}
	for _, svc := range services {
		t.Run(svc, func(t *testing.T) {
			playbookPath, err := GeneratePlaybookFile(tempDir, svc, vars)
			if err != nil {
				t.Fatalf("failed to generate playbook for %s: %v", svc, err)
			}

			contentBytes, err := os.ReadFile(playbookPath)
			if err != nil {
				t.Fatalf("failed to read generated playbook for %s: %v", svc, err)
			}
			content := string(contentBytes)

			// Validate that the generated playbook is valid YAML
			var parsed []map[string]interface{}
			if err := yaml.Unmarshal(contentBytes, &parsed); err != nil {
				t.Errorf("generated playbook for %s is not valid YAML: %v\nContent:\n%s", svc, err, content)
			}

			if svc == "mysql" {
				if !strings.Contains(content, "/data/opsvault/mysql") {
					t.Errorf("expected data root to be rendered in mysql playbook")
				}
				if !strings.Contains(content, "opsvault-mysql") {
					t.Errorf("expected mysql container name prefix to be rendered")
				}
				if !strings.Contains(content, "MYSQL_ROOT_PASSWORD='rootpassword'") {
					t.Errorf("expected mysql root password to be rendered")
				}
			} else if svc == "nginx" {
				if !strings.Contains(content, "worker_processes auto;") {
					t.Errorf("expected nginx base config to be rendered in nginx playbook")
				}
			}
		})
	}
}

