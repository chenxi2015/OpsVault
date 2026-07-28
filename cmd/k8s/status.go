package k8s

import (
	"context"
	"fmt"
	"os"

	k8sdriver "OpsVault/internal/driver/k8s"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看 K8s 集群控制面与 Node 节点运行概况",
	RunE: func(cmd *cobra.Command, args []string) error {
		clientset, _, err := k8sdriver.GetK8sClient(KubeConfigPath)
		if err != nil {
			fmt.Printf("❌ 无法连接到 Kubernetes 集群: %v\n", err)
			return err
		}

		ctx := context.Background()
		nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("读取节点信息失败: %w", err)
		}

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Padding(0, 1)
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
		passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))

		fmt.Println(titleStyle.Render("☸️  Kubernetes 集群运行状态概览"))
		fmt.Printf("节点总数: %d\n\n", len(nodes.Items))

		fmt.Printf("%-20s %-12s %-18s %-25s %-8s %-10s\n", "NODE 名称", "状态", "KUBELET 版本", "操作系统", "CPU", "内存")
		fmt.Println("-----------------------------------------------------------------------------------------")

		for _, node := range nodes.Items {
			isReady := false
			for _, cond := range node.Status.Conditions {
				if cond.Type == "Ready" && cond.Status == "True" {
					isReady = true
					break
				}
			}

			statusStr := warnStyle.Render("NotReady")
			if isReady {
				statusStr = passStyle.Render("Ready")
			}

			cpuStr := node.Status.Capacity.Cpu().String()
			memStr := node.Status.Capacity.Memory().String()

			fmt.Printf("%-20s %-12s %-18s %-25s %-8s %-10s\n",
				headerStyle.Render(node.Name),
				statusStr,
				node.Status.NodeInfo.KubeletVersion,
				node.Status.NodeInfo.OSImage,
				cpuStr,
				memStr,
			)
		}

		_ = os.Stdout.Sync()
		return nil
	},
}

func init() {
	K8sCmd.AddCommand(statusCmd)
}
