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

	if err := config.BindPFlag("mode", rootCmd.PersistentFlags().Lookup("mode")); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(newTUICommand())
	rootCmd.AddCommand(nginx.NewCommand(config))
	rootCmd.AddCommand(mysql.NewCommand(config, dockerFactory))
	rootCmd.AddCommand(redis.NewCommand(config, dockerFactory))
	rootCmd.AddCommand(rocketmq.NewCommand(config, dockerFactory))
	rootCmd.AddCommand(rabbitmq.NewCommand(config, dockerFactory))
	rootCmd.AddCommand(postgres.NewCommand(config, dockerFactory))
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
	v.SetDefault("docker.network_name", "opsvault-net")
	v.SetDefault("docker.cidr", "172.28.0.0/16")
	v.SetDefault("docker.data_root", "/data/opsvault")
	v.SetDefault("docker.images.mysql", "mysql:8.0")
	v.SetDefault("docker.images.redis", "redis:7-alpine")
	v.SetDefault("docker.images.rocketmq", "apache/rocketmq:5.3.0")
	v.SetDefault("docker.images.rabbitmq", "rabbitmq:3-management")
	v.SetDefault("docker.images.postgres", "postgres:15")
	v.SetDefault("docker.resource_limit.cpu_max", "2")
	v.SetDefault("docker.resource_limit.mem_max", "2g")
	v.SetDefault("oneinstack.auto_script_url", "https://oneinstack.com/auto/")
	v.SetDefault("oneinstack.nginx_install_path", "/usr/local/nginx")
	v.SetDefault("oneinstack.www_root", "/data/wwwroot")
	v.SetDefault("oneinstack.ssl_root", "/data/ssl")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.storage_path", "/data/opsvault/logs")
}

func newDockerClient() (*client.Client, error) {
	return dockercli.New()
}
