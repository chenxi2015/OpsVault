package k8s

import (
	"context"
	"fmt"

	k8sdriver "OpsVault/internal/driver/k8s"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	deleteNamespace string
	forceDelete     bool
	cleanupFailed   bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "从 K8s 集群中删除特定资源或批量清理僵尸/异常 Pod",
	Long:  `opsvault k8s delete pod [pod-name] 提供 Pod 删除与一键批量清理异常僵尸 Pod 功能。`,
}

var deletePodCmd = &cobra.Command{
	Use:     "pod [pod-name]",
	Aliases: []string{"pods"},
	Short:   "删除指定的 Pod 或一键清理全部僵尸/异常 Pod",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))

		// 1. Batch cleanup mode
		if cleanupFailed {
			fmt.Println("🔍 正在扫描并强力清理集群中的异常与僵尸 Pod...")
			count, err := k8sdriver.CleanupFailedPods(ctx, KubeConfigPath, deleteNamespace)
			if err != nil {
				return fmt.Errorf("清理异常 Pod 失败: %w", err)
			}
			if count == 0 {
				fmt.Println(passStyle.Render("✔ 未发现需要清理的异常或僵尸 Pod。"))
			} else {
				fmt.Println(passStyle.Render(fmt.Sprintf("🎉 成功强力清理 %d 个僵尸/异常 Pod！", count)))
			}
			return nil
		}

		// 2. Single Pod deletion mode
		if len(args) == 0 {
			return fmt.Errorf("缺少 Pod 名称！示例: opsvault k8s delete pod <pod-name> 或使用 --cleanup-failed 清理全部异常 Pod")
		}

		podName := args[0]
		ns := deleteNamespace
		if ns == "" {
			ns = "default"
		}

		if forceDelete {
			fmt.Println(warnStyle.Render(fmt.Sprintf("⚠️  正在对 Pod %s/%s 执行 0-GracePeriod 强制强行删除...", ns, podName)))
		} else {
			fmt.Printf("🗑️  正在删除 Pod %s/%s ...\n", ns, podName)
		}

		err := k8sdriver.DeletePod(ctx, KubeConfigPath, ns, podName, forceDelete)
		if err != nil {
			return fmt.Errorf("删除 Pod %s/%s 失败: %w", ns, podName, err)
		}

		fmt.Println(passStyle.Render(fmt.Sprintf("✔ Pod %s/%s 已成功删除！", ns, podName)))
		return nil
	},
}

func init() {
	deletePodCmd.Flags().StringVarP(&deleteNamespace, "namespace", "n", "", "指定命名空间 (默认为 default；使用 --cleanup-failed 时留空可扫描所有命名空间)")
	deletePodCmd.Flags().BoolVar(&forceDelete, "force", false, "是否强制删除 (设置 0-GracePeriod 绕过优雅退出直接抹除卡死 Pod)")
	deletePodCmd.Flags().BoolVar(&cleanupFailed, "cleanup-failed", false, "自动扫描并批量强力清理所有处于 Evicted/Terminating/CrashLoop 等异常状态的僵尸 Pod")

	deleteCmd.AddCommand(deletePodCmd)
	K8sCmd.AddCommand(deleteCmd)
}
