package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	migrateHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("208")).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color("208")).
				PaddingRight(2)

	migrateSuccessCard = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("10")).
				Padding(1, 3).
				MarginTop(1).
				MarginBottom(1)

	migrateWarnCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("208")).
			Padding(1, 3).
			MarginTop(1).
			MarginBottom(1)
)

func newMigrateCommand(cfg *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage service migrations (host-to-host or engine conversion)",
		Long:  "Migrate configurations and persistent data between hosts or convert the running engine of services.",
	}

	cmd.AddCommand(
		newMigrateHostCommand(cfg),
		newMigrateEngineCommand(cfg),
		newMigrateStatusCommand(cfg),
	)

	return cmd
}

func newMigrateHostCommand(cfg *viper.Viper) *cobra.Command {
	var service string
	var sourceGroup string
	var targetGroup string
	var syncData bool
	var purgeSource bool

	cmd := &cobra.Command{
		Use:   "host",
		Short: "Migrate service data and configs from one host group to another",
		RunE: func(cmd *cobra.Command, args []string) error {
			service = strings.ToLower(service)
			if service == "" {
				return fmt.Errorf("service is required (use --service)")
			}
			if sourceGroup == "" || targetGroup == "" {
				return fmt.Errorf("both --source-group and --target-group are required")
			}

			// 特殊安全检查：如果是 minio 且开启了数据同步，进行提示并做安全拦截
			if service == "minio" && syncData {
				warnContent := fmt.Sprintf(
					"⚠️  [WARN] 检测到迁移服务为 MinIO 对象存储！\n\n为防磁盘爆满及迁移超时，OpsVault 只会为您迁移并重建 MinIO 容器参数与配置，不默认在主机级别同步业务对象文件。\n\n请在目标主机拉起服务后，手动运行 `mc mirror` 等逻辑工具同步业务数据。",
				)
				cmd.Println(migrateWarnCard.Render(warnContent))
				// 强制将 syncData 设为 false，确保不打包数据
				syncData = false
			}

			// 占位输出，表明接口框架正常接收指令
			cardContent := fmt.Sprintf(
				"🚀 [MIGRATE HOST] 迁移任务初始化成功！\n\n服务名称: %s\n源节点组: %s\n目标节点组: %s\n同步数据: %t\n清理源端: %t\n\n系统提示: 当前迁移模块核心驱动正在实施搭建中，后续将完整执行 Ansible 管道同步。",
				service, sourceGroup, targetGroup, syncData, purgeSource,
			)
			cmd.Println(migrateSuccessCard.Render(cardContent))
			return nil
		},
	}

	cmd.Flags().StringVar(&service, "service", "", "Middleware service name to migrate (e.g. mysql, redis)")
	cmd.Flags().StringVar(&sourceGroup, "source-group", "", "Ansible host group of the source node")
	cmd.Flags().StringVar(&targetGroup, "target-group", "", "Ansible host group of the target node")
	cmd.Flags().BoolVar(&syncData, "sync-data", true, "Whether to synchronize persistent data directories")
	cmd.Flags().BoolVar(&purgeSource, "purge-source", false, "Whether to purge source service container and data after successful migration")

	_ = cmd.MarkFlagRequired("service")
	_ = cmd.MarkFlagRequired("source-group")
	_ = cmd.MarkFlagRequired("target-group")

	return cmd
}

func newMigrateEngineCommand(cfg *viper.Viper) *cobra.Command {
	var service string
	var from string
	var to string

	cmd := &cobra.Command{
		Use:   "engine",
		Short: "Convert running service from one deployment engine mode to another (e.g. binary to docker)",
		RunE: func(cmd *cobra.Command, args []string) error {
			service = strings.ToLower(service)
			if service == "" {
				return fmt.Errorf("service is required (use --service)")
			}
			if from == "" || to == "" {
				return fmt.Errorf("both --from and --to flags are required")
			}

			cardContent := fmt.Sprintf(
				"🔄 [MIGRATE ENGINE] 本地部署引擎转换初始化！\n\n服务名称: %s\n当前引擎: %s\n目标引擎: %s\n\n系统提示: 本地引擎转换暂未完全开放，核心驱动正在拼装中。",
				service, from, to,
			)
			cmd.Println(migrateSuccessCard.Render(cardContent))
			return nil
		},
	}

	cmd.Flags().StringVar(&service, "service", "", "Middleware service name (e.g. nginx, mysql)")
	cmd.Flags().StringVar(&from, "from", "", "Current deployment engine (docker or binary)")
	cmd.Flags().StringVar(&to, "to", "", "Target deployment engine (docker or binary)")

	_ = cmd.MarkFlagRequired("service")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func newMigrateStatusCommand(cfg *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the status of active migration tasks",
		Run: func(cmd *cobra.Command, args []string) {
			header := migrateHeaderStyle.Render("ACTIVE MIGRATION TASKS")
			cmd.Println(header)
			cmd.Println("No active migration tasks found.")
		},
	}

	return cmd
}
