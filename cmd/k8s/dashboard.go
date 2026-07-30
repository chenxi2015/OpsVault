package k8s

import (
	"fmt"

	k8sdriver "OpsVault/internal/driver/k8s"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	dashboardType string
	dashboardPort int
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "一键向 K8s 集群部署图形化 Web 控制面板 (例如: Kuboard v3)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := getConfig()
		if !cmd.Flags().Changed("type") && cfg.GetString("k8s.dashboard.type") != "" {
			dashboardType = cfg.GetString("k8s.dashboard.type")
		}
		if !cmd.Flags().Changed("port") && cfg.GetInt("k8s.dashboard.port") > 0 {
			dashboardPort = cfg.GetInt("k8s.dashboard.port")
		}

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
		successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
		highlightStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))

		fmt.Println(titleStyle.Render(fmt.Sprintf("🖥️  正在向集群部署 %s Web 控制面板...", dashboardType)))

		if dashboardType != "kuboard" {
			return fmt.Errorf("暂不支持的面板类型: %s (目前支持: kuboard)", dashboardType)
		}

		err := k8sdriver.DeployKuboard(KubeConfigPath, dashboardPort)
		if err != nil {
			return fmt.Errorf("部署 %s 面板失败: %w", dashboardType, err)
		}

		ip := k8sdriver.GetPublicIP()

		fmt.Println("\n" + successStyle.Render("🎉 Kuboard v3 Web 控制面板已成功部署到集群！"))
		fmt.Printf("● 访问 URL    : %s\n", highlightStyle.Render(fmt.Sprintf("http://%s:%d", ip, dashboardPort)))
		fmt.Println("● 默认初始账号: admin")
		fmt.Println("● 默认初始密码: Kuboard123")
		fmt.Println("\n💡 提示: 部署后镜像拉取与服务启动可能需要 15-30 秒，请在浏览器刷新访问。")

		return nil
	},
}

func init() {
	dashboardCmd.Flags().StringVar(&dashboardType, "type", "kuboard", "控制面板类型 (默认 kuboard)")
	dashboardCmd.Flags().IntVar(&dashboardPort, "port", k8sdriver.DefaultKuboardPort, "映射暴露的 NodePort 端口号")

	K8sCmd.AddCommand(dashboardCmd)
}
