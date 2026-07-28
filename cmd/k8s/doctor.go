package k8s

import (
	"context"
	"fmt"

	k8sdriver "OpsVault/internal/driver/k8s"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "对 K8s 集群节点与 Pod 运行状况进行深度健康体检诊断",
	RunE: func(cmd *cobra.Command, args []string) error {
		clientset, _, err := k8sdriver.GetK8sClient(KubeConfigPath)
		if err != nil {
			fmt.Printf("❌ 连接 Kubernetes 集群失败: %v\n", err)
			return err
		}

		fmt.Println("🔍 正在扫描 K8s 集群 Node 与 Pod 健康状况...")

		health, err := k8sdriver.InspectCluster(context.Background(), clientset)
		if err != nil {
			return fmt.Errorf("集群体检失败: %w", err)
		}

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Padding(0, 1)
		passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
		failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

		fmt.Println("\n" + titleStyle.Render("🏥 K8s 集群健康体检诊断报告"))
		fmt.Printf("● 节点概况: 总计 %d 个, %s 正常就绪\n", health.TotalNodes, passStyle.Render(fmt.Sprintf("%d 个", health.ReadyNodes)))
		fmt.Printf("● Pod 概况 : 总计 %d 个, %s 正常运行\n", health.TotalPods, passStyle.Render(fmt.Sprintf("%d 个", health.RunningPods)))

		if len(health.AbnormalPods) == 0 {
			fmt.Println("\n" + passStyle.Render("✔ 集群内未检测到异常 Pod，运行状况良好！"))
		} else {
			fmt.Println("\n" + warnStyle.Render(fmt.Sprintf("⚠️ 检测到 %d 个运行异常或高频重启的 Pod:", len(health.AbnormalPods))))
			fmt.Printf("%-20s %-30s %-15s %-15s %-8s\n", "NAMESPACE", "POD 名称", "状态", "异常原因", "重启次数")
			fmt.Println("-----------------------------------------------------------------------------------------")
			for _, pod := range health.AbnormalPods {
				fmt.Printf("%-20s %-30s %-15s %-15s %-8d\n",
					pod.Namespace,
					pod.Name,
					failStyle.Render(pod.Status),
					warnStyle.Render(pod.Reason),
					pod.Restarts,
				)
			}
		}

		return nil
	},
}

func init() {
	K8sCmd.AddCommand(doctorCmd)
}
