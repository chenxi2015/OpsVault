package ansiblecmd

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"OpsVault/internal/driver/ansible"
	"OpsVault/pkg/versionutil"

	"github.com/spf13/cobra"
)

func (c *commandSet) newDeployCommand() *cobra.Command {
	var service string
	var group string

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy specified middleware or docker onto remote hosts via Ansible Playbook",
		RunE: func(cmd *cobra.Command, args []string) error {
			if service == "" {
				return errors.New("service name must be specified, use --service")
			}

			// Validate service support
			switch service {
			case "docker", "mysql", "redis", "rabbitmq", "nginx":
				// valid
			default:
				return fmt.Errorf("unsupported service: %s. Supported: docker, mysql, redis, rabbitmq, nginx", service)
			}

			exec, cleanup, err := c.getExecutor()
			if err != nil {
				return err
			}
			defer cleanup()

			// Prepare playbook variables from config
			v := c.config
			vars := ansible.PlaybookVars{
				DataRoot:    v.GetString("docker.data_root"),
				NetworkName: v.GetString("docker.network_name"),
				CIDR:        v.GetString("docker.cidr"),
				NamePrefix:  v.GetString("docker.name_prefix"),
			}

			if vars.DataRoot == "" {
				vars.DataRoot = "/data/opsvault"
			}
			if vars.NetworkName == "" {
				vars.NetworkName = "opsvault-net"
			}
			if vars.CIDR == "" {
				vars.CIDR = "172.28.0.0/16"
			}
			if vars.NamePrefix == "" {
				vars.NamePrefix = "opsvault"
			}

			// Service specific configuration injection
			switch service {
			case "mysql":
				vars.MySQLImage = v.GetString("mysql.image")
				vars.MySQLPort = v.GetInt("mysql.port")
				vars.MySQLRootPassword = v.GetString("mysql.root_password")
				if vars.MySQLImage == "" {
					vars.MySQLImage = "mysql:8.0"
				}
				if vars.MySQLPort == 0 {
					vars.MySQLPort = 3306
				}
				if vars.MySQLRootPassword == "" {
					vars.MySQLRootPassword = generateRandomPassword()
				}
			case "redis":
				vars.RedisImage = v.GetString("redis.image")
				vars.RedisPort = v.GetInt("redis.port")
				vars.RedisPassword = v.GetString("redis.password")
				if vars.RedisImage == "" {
					vars.RedisImage = "redis:7-alpine"
				}
				if vars.RedisPort == 0 {
					vars.RedisPort = 6379
				}
				if vars.RedisPassword == "" {
					vars.RedisPassword = generateRandomPassword()
				}
			case "rabbitmq":
				vars.RabbitMQImage = v.GetString("rabbitmq.image")
				vars.RabbitMQPort = v.GetInt("rabbitmq.port")
				vars.RabbitMQUIPort = v.GetInt("rabbitmq.ui_port")
				vars.RabbitMQUser = v.GetString("rabbitmq.admin_user")
				vars.RabbitMQPwd = v.GetString("rabbitmq.admin_pwd")
				if vars.RabbitMQImage == "" {
					vars.RabbitMQImage = "rabbitmq:3-management"
				}
				if vars.RabbitMQPort == 0 {
					vars.RabbitMQPort = 5672
				}
				if vars.RabbitMQUIPort == 0 {
					vars.RabbitMQUIPort = 15672
				}
				if vars.RabbitMQUser == "" {
					vars.RabbitMQUser = "admin"
				}
				if vars.RabbitMQPwd == "" {
					vars.RabbitMQPwd = generateRandomPassword()
				}
			case "nginx":
				vars.NginxVersion = versionutil.ResolveNginxVersion(
					v.GetString("nginx.version"),
					"1.26.2",
				)
				vars.NginxPCREVersion = v.GetString("nginx.pcre_version")
				vars.NginxOpenSSLVersion = v.GetString("nginx.openssl_version")
				vars.NginxInstallPath = v.GetString("nginx.install_path")
				vars.NginxSourceRoot = v.GetString("nginx.source_root")
				vars.NginxWWWRoot = v.GetString("nginx.www_root")
				vars.NginxSSLRoot = v.GetString("nginx.ssl_root")
				vars.NginxWWWLogsRoot = v.GetString("nginx.wwwlogs_root")
				vars.NginxRunUser = v.GetString("nginx.run_user")
				vars.NginxRunGroup = v.GetString("nginx.run_group")
				vars.NginxSystemdUnitPath = v.GetString("nginx.systemd_unit_path")
				// Apply defaults for fields not set in config
				if vars.NginxPCREVersion == "" {
					vars.NginxPCREVersion = "8.45"
				}
				if vars.NginxOpenSSLVersion == "" {
					vars.NginxOpenSSLVersion = "1.1.1w"
				}
				if vars.NginxInstallPath == "" {
					vars.NginxInstallPath = "/usr/local/nginx"
				}
				if vars.NginxSourceRoot == "" {
					vars.NginxSourceRoot = "/usr/local/src/opsvault-nginx"
				}
				if vars.NginxWWWRoot == "" {
					vars.NginxWWWRoot = "/data/wwwroot"
				}
				if vars.NginxSSLRoot == "" {
					vars.NginxSSLRoot = "/data/ssl"
				}
				if vars.NginxWWWLogsRoot == "" {
					vars.NginxWWWLogsRoot = "/data/wwwlogs"
				}
				if vars.NginxRunUser == "" {
					vars.NginxRunUser = "www"
				}
				if vars.NginxRunGroup == "" {
					vars.NginxRunGroup = "www"
				}
				if vars.NginxSystemdUnitPath == "" {
					vars.NginxSystemdUnitPath = "/lib/systemd/system/nginx.service"
				}
			}

			tempDir := v.GetString("ansible.temp_dir")
			if tempDir == "" {
				tempDir = "/data/opsvault/ansible/tmp"
			}

			fmt.Printf("Generating deployment playbook for service: %s...\n", service)
			playbookFile, err := ansible.GeneratePlaybookFile(tempDir, service, vars)
			if err != nil {
				return fmt.Errorf("failed to generate playbook: %w", err)
			}
			defer func() {
				_ = os.Remove(playbookFile)
			}()

			fmt.Printf("Executing playbook deployment on group: %s...\n", group)
			// extraVars can be passed dynamically if needed, keeping it empty for now
			err = exec.RunPlaybook(cmd.Context(), playbookFile, nil, os.Stdout, os.Stderr)
			if err != nil {
				return fmt.Errorf("playbook execution failed: %w", err)
			}

			fmt.Printf("Deployment of %s completed successfully.\n", service)
			return nil
		},
	}

	cmd.Flags().StringVarP(&service, "service", "s", "", "middleware service to deploy (docker, mysql, redis, rabbitmq, nginx)")
	cmd.Flags().StringVarP(&group, "group", "g", "all", "target host group for deployment")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func generateRandomPassword() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "DefaultPassword123"
	}
	return hex.EncodeToString(bytes)
}
