package ansiblecmd

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"OpsVault/internal/driver/ansible"
	"OpsVault/pkg/credutil"
	"OpsVault/pkg/fileutil"
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
			case "docker", "mysql", "redis", "rabbitmq", "nginx", "minio", "nacos":
				// valid
			default:
				return fmt.Errorf("unsupported service: %s. Supported: docker, mysql, redis, rabbitmq, nginx, minio, nacos", service)
			}

			exec, cleanup, err := c.getExecutor()
			if err != nil {
				return err
			}
			defer cleanup()

			// Prepare playbook variables from config
			v := c.config
			vars := ansible.PlaybookVars{
				TargetGroup:     group,
				DataRoot:        v.GetString("docker.data_root"),
				NetworkName:     v.GetString("docker.network_name"),
				CIDR:            v.GetString("docker.cidr"),
				NamePrefix:      v.GetString("docker.name_prefix"),
				RegistryMirrors: v.GetStringSlice("docker.registry_mirrors"),
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
			if len(vars.RegistryMirrors) == 0 {
				vars.RegistryMirrors = []string{
					"https://docker.1panel.dev",
					"https://docker.foreverlink.love",
					"https://docker.xn--6oq72ry9d5zx.cn",
					"https://docker.zhai.cm",
					"https://docker.5z5f.com",
					"https://a.ussh.net",
					"https://docker.cloudlayer.icu",
					"https://hub.littlediary.cn",
					"https://hub.crdz.gq",
					"https://docker.unsee.tech",
					"https://docker.kejilion.pro",
					"https://registry.dockermirror.com",
					"https://hub.rat.dev",
					"https://dhub.kubesre.xyz",
					"https://docker.nastool.de",
					"https://docker.udayun.com",
					"https://docker.rainbond.cc",
					"https://docker.1panelproxy.com",
					"https://atomhub.openatom.cn",
					"https://docker.m.daocloud.io",
					"https://docker.1ms.run",
					"https://docker.linkedbus.com",
					"https://dytt.online",
					"https://func.ink",
					"https://lispy.org",
					"https://docker.xiaogenban1993.com",
				}
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
					v.Set("mysql.root_password", vars.MySQLRootPassword)
					cfgPath := v.ConfigFileUsed()
					if cfgPath == "" {
						cfgPath = fileutil.GetDefaultWriteConfigPath()
					}
					_ = fileutil.UpdateYAMLValue(cfgPath, "mysql", "root_password", vars.MySQLRootPassword)
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
					v.Set("redis.password", vars.RedisPassword)
					cfgPath := v.ConfigFileUsed()
					if cfgPath == "" {
						cfgPath = fileutil.GetDefaultWriteConfigPath()
					}
					_ = fileutil.UpdateYAMLValue(cfgPath, "redis", "password", vars.RedisPassword)
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
					v.Set("rabbitmq.admin_pwd", vars.RabbitMQPwd)
					cfgPath := v.ConfigFileUsed()
					if cfgPath == "" {
						cfgPath = fileutil.GetDefaultWriteConfigPath()
					}
					_ = fileutil.UpdateYAMLValue(cfgPath, "rabbitmq", "admin_pwd", vars.RabbitMQPwd)
				}
			case "nginx":
				vars.NginxVersion = versionutil.ResolveNginxVersion(
					v.GetString("nginx.version"),
					"1.26.2",
				)
				vars.NginxPCREVersion = v.GetString("nginx.pcre_version")
				vars.NginxOpenSSLVersion = versionutil.ResolveOpenSSLVersion(
					v.GetString("nginx.openssl_version"),
					"3.0.15",
				)
				vars.NginxOpenSSLURLs = versionutil.GetOpenSSLDownloadURLs(vars.NginxOpenSSLVersion)
				if len(vars.NginxOpenSSLURLs) > 0 {
					vars.NginxOpenSSLURL = vars.NginxOpenSSLURLs[0]
				}
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
					vars.NginxOpenSSLVersion = "3.0.15"
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
			case "minio":
				vars.MinIOImage = v.GetString("minio.image")
				vars.MinIOPort = v.GetInt("minio.port")
				vars.MinIOConsolePort = v.GetInt("minio.console_port")
				vars.MinIORootUser = v.GetString("minio.root_user")
				vars.MinIORootPassword = v.GetString("minio.root_password")
				if vars.MinIOImage == "" {
					vars.MinIOImage = "minio/minio:RELEASE.2024-05-10T01-39-39Z"
				}
				if vars.MinIOPort == 0 {
					vars.MinIOPort = 9000
				}
				if vars.MinIOConsolePort == 0 {
					vars.MinIOConsolePort = 9001
				}
				if vars.MinIORootUser == "" {
					vars.MinIORootUser = "minioadmin"
				}
				if vars.MinIORootPassword == "" {
					vars.MinIORootPassword = generateRandomPassword()
					v.Set("minio.root_password", vars.MinIORootPassword)
					cfgPath := v.ConfigFileUsed()
					if cfgPath == "" {
						cfgPath = fileutil.GetDefaultWriteConfigPath()
					}
					_ = fileutil.UpdateYAMLValue(cfgPath, "minio", "root_password", vars.MinIORootPassword)
				}
			case "nacos":
				vars.NacosImage = v.GetString("nacos.image")
				vars.NacosPort = v.GetInt("nacos.port")
				vars.NacosGrpcPort1 = v.GetInt("nacos.grpc_port_1")
				vars.NacosGrpcPort2 = v.GetInt("nacos.grpc_port_2")
				vars.NacosAuthEnable = true
				if v.IsSet("nacos.auth_enable") {
					vars.NacosAuthEnable = v.GetBool("nacos.auth_enable")
				}
				vars.NacosAuthToken = v.GetString("nacos.auth_token")

				if vars.NacosImage == "" {
					vars.NacosImage = "nacos/nacos-server:v2.3.2"
				}
				if vars.NacosPort == 0 {
					vars.NacosPort = 8848
				}
				if vars.NacosGrpcPort1 == 0 {
					vars.NacosGrpcPort1 = 9848
				}
				if vars.NacosGrpcPort2 == 0 {
					vars.NacosGrpcPort2 = 9849
				}
				if vars.NacosAuthToken == "" && vars.NacosAuthEnable {
					b := make([]byte, 32)
					_, _ = rand.Read(b)
					vars.NacosAuthToken = base64.StdEncoding.EncodeToString(b)
					v.Set("nacos.auth_token", vars.NacosAuthToken)
					cfgPath := v.ConfigFileUsed()
					if cfgPath == "" {
						cfgPath = fileutil.GetDefaultWriteConfigPath()
					}
					_ = fileutil.UpdateYAMLValue(cfgPath, "nacos", "auth_token", vars.NacosAuthToken)
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
			err = exec.RunPlaybook(cmd.Context(), playbookFile, group, nil, os.Stdout, os.Stderr)
			if err != nil {
				return fmt.Errorf("playbook execution failed: %w", err)
			}

			fmt.Printf("Deployment of %s completed successfully.\n", service)

			var creds []credutil.Credential
			switch service {
			case "mysql":
				creds = []credutil.Credential{
					{Label: "目标分组", Value: group},
					{Label: "端口", Value: fmt.Sprintf("%d", vars.MySQLPort)},
					{Label: "用户名", Value: "root"},
					{Label: "密  码", Value: vars.MySQLRootPassword},
				}
			case "redis":
				pwd := vars.RedisPassword
				if pwd == "" {
					pwd = "(无认证)"
				}
				creds = []credutil.Credential{
					{Label: "目标分组", Value: group},
					{Label: "端口", Value: fmt.Sprintf("%d", vars.RedisPort)},
					{Label: "密  码", Value: pwd},
				}
			case "rabbitmq":
				creds = []credutil.Credential{
					{Label: "目标分组", Value: group},
					{Label: "管理界面", Value: fmt.Sprintf("http://<目标主机>:%d", vars.RabbitMQUIPort)},
					{Label: "AMQP 端口", Value: fmt.Sprintf("%d", vars.RabbitMQPort)},
					{Label: "用户名", Value: vars.RabbitMQUser},
					{Label: "密  码", Value: vars.RabbitMQPwd},
				}
			case "minio":
				creds = []credutil.Credential{
					{Label: "目标分组", Value: group},
					{Label: "API 端口", Value: fmt.Sprintf("%d", vars.MinIOPort)},
					{Label: "控制台端口", Value: fmt.Sprintf("%d", vars.MinIOConsolePort)},
					{Label: "用户名", Value: vars.MinIORootUser},
					{Label: "密  码", Value: vars.MinIORootPassword},
				}
			case "nacos":
				authVal := "Enabled"
				if !vars.NacosAuthEnable {
					authVal = "Disabled"
				}
				creds = []credutil.Credential{
					{Label: "目标分组", Value: group},
					{Label: "控制台端口", Value: fmt.Sprintf("%d", vars.NacosPort)},
					{Label: "认证状态", Value: authVal},
				}
				if vars.NacosAuthEnable {
					creds = append(creds,
						credutil.Credential{Label: "默认用户名", Value: "nacos"},
						credutil.Credential{Label: "默认密码", Value: "nacos (可在控制台修改)"},
						credutil.Credential{Label: "Token Secret", Value: vars.NacosAuthToken},
					)
				}
			}
			if len(creds) > 0 {
				credutil.PrintCredentials(strings.ToUpper(service), creds)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&service, "service", "s", "", "middleware service to deploy (docker, mysql, redis, rabbitmq, nginx, minio, nacos)")
	cmd.Flags().StringVarP(&group, "group", "g", "all", "target host group for deployment")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func generateRandomPassword() string {
	return credutil.GenPassword(20)
}
