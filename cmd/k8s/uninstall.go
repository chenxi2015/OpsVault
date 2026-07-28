package k8s

import (
	"context"
	"fmt"

	k8sdriver "OpsVault/internal/driver/k8s"
	"OpsVault/pkg/dockercli"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	uninstallMode  string
	uninstallPurge bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "停止并卸载 K3s Kubernetes 集群 (支持清理持久化数据)",
	RunE: func(cmd *cobra.Command, args []string) error {
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))

		fmt.Println(titleStyle.Render("🧹 正在清理与卸载 K3s Kubernetes 集群..."))

		cli, _ := dockercli.New()
		if cli != nil {
			defer cli.Close()
		}

		err := k8sdriver.UninstallK3s(context.Background(), cli, uninstallMode, uninstallPurge)
		if err != nil {
			return fmt.Errorf("卸载 K3s 失败: %w", err)
		}

		if uninstallPurge {
			fmt.Println(warnStyle.Render("🔥 物理数据目录与 API 凭据已深度清空！"))
		} else {
			fmt.Println("💡 宿主机持久化数据目录已保留 (若需物理删除请添加 --purge 参数)")
		}

		return nil
	},
}

func init() {
	uninstallCmd.Flags().StringVar(&uninstallMode, "mode", "binary", "卸载运行模式: binary | docker")
	uninstallCmd.Flags().BoolVar(&uninstallPurge, "purge", false, "深度擦除持久化数据目录与配置文件")

	K8sCmd.AddCommand(uninstallCmd)
}
