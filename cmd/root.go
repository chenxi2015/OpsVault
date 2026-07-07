package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"OpsVault/cmd/mysql"
	"OpsVault/cmd/nginx"
	"OpsVault/cmd/postgres"
	"OpsVault/cmd/rabbitmq"
	"OpsVault/cmd/redis"
	"OpsVault/cmd/rocketmq"
	"OpsVault/internal/driver"
	"OpsVault/pkg/dockercli"
	"OpsVault/pkg/logger"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	debug   bool
	config  = viper.New()

	dockerFactory = newDockerClient
)

var rootCmd = &cobra.Command{
	Use:   "opsvault",
	Short: "OpsVault 运维百宝箱",
	Long:  "OpsVault 是一站式 CentOS 中间件与 Web 服务运维 CLI/TUI 工具集。",
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := initConfig(); err != nil {
			return err
		}
		logger.Configure(debug)
		// Initialize audit log to the configured log storage path
		logDir := config.GetString("log.storage_path")
		if logDir == "" {
			logDir = "/data/opsvault/logs"
		}
		logger.ConfigureAudit(logDir)
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize()

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().String("mode", string(driver.ModeDocker), "driver mode: docker|binary")
	rootCmd.PersistentFlags().String("bind-ip", "", "Docker port bind address: 0.0.0.0 (all interfaces) or 127.0.0.1 (localhost only)")

	if err := config.BindPFlag("mode", rootCmd.PersistentFlags().Lookup("mode")); err != nil {
		panic(err)
	}
	if err := config.BindPFlag("docker.bind_ip", rootCmd.PersistentFlags().Lookup("bind-ip")); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(newTUICommand())
	rootCmd.AddCommand(newDoctorCommand(config, dockerFactory))
	rootCmd.AddCommand(newInitCommand(config, dockerFactory))
	rootCmd.AddCommand(nginx.NewCommand(config))
	rootCmd.AddCommand(mysql.NewCommand(config, dockerFactory))
	rootCmd.AddCommand(redis.NewCommand(config, dockerFactory))
	rootCmd.AddCommand(rocketmq.NewCommand(config, dockerFactory))
	rootCmd.AddCommand(rabbitmq.NewCommand(config, dockerFactory))
	rootCmd.AddCommand(postgres.NewCommand(config, dockerFactory))
	rootCmd.AddCommand(newBakCommand(config))
}

func initConfig() error {
	config.SetConfigType("yaml")
	applyDefaultConfig(config)

	if cfgFile != "" {
		config.SetConfigFile(cfgFile)
	} else {
		config.SetConfigName("default")
		config.AddConfigPath("./configs")
		config.AddConfigPath(".")
		if exePath, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exePath)
			config.AddConfigPath(filepath.Join(exeDir, "configs"))
			config.AddConfigPath(exeDir)
		}
		if home, err := os.UserHomeDir(); err == nil {
			config.AddConfigPath(filepath.Join(home, ".opsvault"))
		}
	}

	err := config.ReadInConfig()
	if err != nil {
		var configNotFound viper.ConfigFileNotFoundError
		if errors.As(err, &configNotFound) && cfgFile == "" {
			return nil
		}
		return fmt.Errorf("read config: %w", err)
	}

	return nil
}

func AppConfig() *viper.Viper {
	return config
}

func DockerClient() (*client.Client, error) {
	return dockerFactory()
}

func CurrentMode(v *viper.Viper) driver.Mode {
	mode := strings.ToLower(v.GetString("mode"))
	if mode == "" {
		return driver.ModeDocker
	}
	return driver.Mode(mode)
}

func applyDefaultConfig(v *viper.Viper) {
	v.SetDefault("docker.name_prefix", "opsvault")
	v.SetDefault("docker.network_name", "opsvault-net")
	v.SetDefault("docker.cidr", "172.28.0.0/16")
	v.SetDefault("docker.data_root", "/data/opsvault")
	v.SetDefault("docker.bind_ip", "0.0.0.0")
	v.SetDefault("docker.resource_limit.cpu_max", "2")
	v.SetDefault("docker.resource_limit.mem_max", "2g")

	v.SetDefault("mysql.image", "mysql:8.0")
	v.SetDefault("mysql.port", 3306)
	v.SetDefault("mysql.root_password", "root")

	v.SetDefault("redis.image", "redis:7-alpine")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")

	v.SetDefault("rocketmq.image", "apache/rocketmq:5.3.0")
	v.SetDefault("rocketmq.namesrv_port", 9876)
	v.SetDefault("rocketmq.broker_port", 10911)

	v.SetDefault("rabbitmq.image", "rabbitmq:3-management")
	v.SetDefault("rabbitmq.port", 5672)
	v.SetDefault("rabbitmq.ui_port", 15672)
	v.SetDefault("rabbitmq.admin_user", "admin")
	v.SetDefault("rabbitmq.admin_pwd", "password")

	v.SetDefault("postgres.image", "postgres:15")
	v.SetDefault("postgres.port", 5432)
	v.SetDefault("postgres.password", "")
	v.SetDefault("nginx.install_path", "/usr/local/nginx")
	v.SetDefault("nginx.www_root", "/data/wwwroot")
	v.SetDefault("nginx.ssl_root", "/data/ssl")
	v.SetDefault("nginx.wwwlogs_root", "/data/wwwlogs")
	v.SetDefault("nginx.source_root", "/usr/local/src/opsvault-nginx")
	v.SetDefault("nginx.version", "1.31.0")
	v.SetDefault("nginx.pcre_version", "8.45")
	v.SetDefault("nginx.openssl_version", "1.1.1w")
	v.SetDefault("nginx.run_user", "www")
	v.SetDefault("nginx.run_group", "www")
	v.SetDefault("nginx.systemd_unit_path", "/lib/systemd/system/nginx.service")
	v.SetDefault("nginx.logrotate_path", "/etc/logrotate.d/nginx")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.storage_path", "/data/opsvault/logs")
	v.SetDefault("backup.storage_path", "/data/opsvault/bak")
}

func newDockerClient() (*client.Client, error) {
	return dockercli.New()
}
