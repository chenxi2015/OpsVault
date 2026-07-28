package k8s

import (
	"fmt"

	k8sdriver "OpsVault/internal/driver/k8s"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	regService   string
	regNamespace string
	regTargetIP  string
	regPort      int
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "将宿主机/Docker 现有中间件注册并关联映射为 K8s 外部 Service",
	RunE: func(cmd *cobra.Command, args []string) error {
		if regService == "" || regTargetIP == "" || regPort <= 0 {
			return fmt.Errorf("必须提供 --service, --target-ip 和 --port 参数")
		}

		drv, err := k8sdriver.NewK8sDriver(regService, KubeConfigPath)
		if err != nil {
			return fmt.Errorf("创建 K8s 驱动失败: %w", err)
		}

		err = drv.RegisterK8sService(regNamespace, regTargetIP, regPort)
		if err != nil {
			return fmt.Errorf("关联 K8s 外部服务失败: %w", err)
		}

		successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
		highlightStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))

		fmt.Println(successStyle.Render("✔ 成功将本地/Docker 服务注册并关联到 K8s 集群！"))
		fmt.Printf("● 服务名称: %s\n", highlightStyle.Render(fmt.Sprintf("opsvault-%s", regService)))
		fmt.Printf("● 命名空间: %s\n", regNamespace)
		fmt.Printf("● 映射目标: %s:%d\n", regTargetIP, regPort)
		fmt.Printf("💡 集群内 Pod 现在可以通过域名 %s 访问该服务\n", highlightStyle.Render(fmt.Sprintf("opsvault-%s.%s.svc.cluster.local", regService, regNamespace)))

		return nil
	},
}

func init() {
	registerCmd.Flags().StringVar(&regService, "service", "", "服务名称 (例如: mysql, redis, minio)")
	registerCmd.Flags().StringVarP(&regNamespace, "namespace", "n", "default", "关联注册的 K8s 命名空间")
	registerCmd.Flags().StringVar(&regTargetIP, "target-ip", "", "宿主机或外部 Docker 服务的 IP 地址")
	registerCmd.Flags().IntVar(&regPort, "port", 0, "外部服务暴露的端口号")

	_ = registerCmd.MarkFlagRequired("service")
	_ = registerCmd.MarkFlagRequired("target-ip")
	_ = registerCmd.MarkFlagRequired("port")

	K8sCmd.AddCommand(registerCmd)
}
