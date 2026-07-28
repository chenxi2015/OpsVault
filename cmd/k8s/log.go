package k8s

import (
	"context"
	"fmt"
	"io"
	"os"

	k8sdriver "OpsVault/internal/driver/k8s"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

var (
	logNamespace string
	logFollow    bool
	logTailLines int64
)

var logCmd = &cobra.Command{
	Use:   "log <pod-name>",
	Short: "查看或追踪指定 K8s Pod 的运行日志",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		podName := args[0]
		clientset, _, err := k8sdriver.GetK8sClient(KubeConfigPath)
		if err != nil {
			return fmt.Errorf("连接 K8s 集群失败: %w", err)
		}

		if logNamespace == "" {
			logNamespace = "default"
		}

		req := clientset.CoreV1().Pods(logNamespace).GetLogs(podName, &corev1.PodLogOptions{
			Follow:    logFollow,
			TailLines: &logTailLines,
		})

		stream, err := req.Stream(context.Background())
		if err != nil {
			return fmt.Errorf("拉取 Pod 日志流失败: %w", err)
		}
		defer stream.Close()

		_, err = io.Copy(os.Stdout, stream)
		return err
	},
}

func init() {
	logCmd.Flags().StringVarP(&logNamespace, "namespace", "n", "default", "指定命名空间")
	logCmd.Flags().BoolVarP(&logFollow, "follow", "f", false, "流式持续输出日志")
	logCmd.Flags().Int64VarP(&logTailLines, "lines", "l", 100, "查看末尾行数")
	K8sCmd.AddCommand(logCmd)
}
