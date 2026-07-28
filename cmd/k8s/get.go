package k8s

import (
	"context"
	"fmt"

	k8sdriver "OpsVault/internal/driver/k8s"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var targetNamespace string

var getCmd = &cobra.Command{
	Use:   "get [pods|nodes|services]",
	Short: "格式化获取 K8s 资源列表 (pods / nodes / services)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resourceType := args[0]
		clientset, _, err := k8sdriver.GetK8sClient(KubeConfigPath)
		if err != nil {
			return fmt.Errorf("连接 K8s 集群失败: %w", err)
		}

		ctx := context.Background()

		switch resourceType {
		case "pods", "pod":
			pods, err := clientset.CoreV1().Pods(targetNamespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}
			fmt.Printf("%-20s %-35s %-12s %-10s %-15s\n", "NAMESPACE", "NAME", "STATUS", "RESTARTS", "NODE")
			fmt.Println("-------------------------------------------------------------------------------------------------")
			for _, pod := range pods.Items {
				var restarts int32
				for _, cs := range pod.Status.ContainerStatuses {
					restarts += cs.RestartCount
				}
				fmt.Printf("%-20s %-35s %-12s %-10d %-15s\n", pod.Namespace, pod.Name, pod.Status.Phase, restarts, pod.Spec.NodeName)
			}

		case "nodes", "node":
			nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}
			fmt.Printf("%-25s %-12s %-20s %-15s\n", "NAME", "STATUS", "VERSION", "INTERNAL-IP")
			fmt.Println("-----------------------------------------------------------------------------------------")
			for _, node := range nodes.Items {
				var ip string
				for _, addr := range node.Status.Addresses {
					if addr.Type == "InternalIP" {
						ip = addr.Address
						break
					}
				}
				status := "NotReady"
				for _, cond := range node.Status.Conditions {
					if cond.Type == "Ready" && cond.Status == "True" {
						status = "Ready"
					}
				}
				fmt.Printf("%-25s %-12s %-20s %-15s\n", node.Name, status, node.Status.NodeInfo.KubeletVersion, ip)
			}

		case "services", "service", "svc":
			svcs, err := clientset.CoreV1().Services(targetNamespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}
			fmt.Printf("%-20s %-30s %-15s %-20s %-15s\n", "NAMESPACE", "NAME", "TYPE", "CLUSTER-IP", "PORTS")
			fmt.Println("---------------------------------------------------------------------------------------------------------")
			for _, svc := range svcs.Items {
				var portsStr string
				for _, p := range svc.Spec.Ports {
					portsStr += fmt.Sprintf("%d/%s ", p.Port, p.Protocol)
				}
				fmt.Printf("%-20s %-30s %-15s %-20s %-15s\n", svc.Namespace, svc.Name, svc.Spec.Type, svc.Spec.ClusterIP, portsStr)
			}

		default:
			return fmt.Errorf("不支持的资源类型: %s (请使用 pods, nodes 或 services)", resourceType)
		}

		return nil
	},
}

func init() {
	getCmd.Flags().StringVarP(&targetNamespace, "namespace", "n", "", "指定命名空间 (默认为全部命名空间)")
	K8sCmd.AddCommand(getCmd)
}
