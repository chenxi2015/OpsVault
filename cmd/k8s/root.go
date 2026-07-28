package k8s

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var KubeConfigPath string

// K8sCmd represents the k8s parent command
var K8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Kubernetes 集群观测、体检与服务关联诊断工具",
	Long:  `opsvault k8s 命令集提供 K8s 集群状态检查、Pods/Nodes 诊断、日志流调取以及外部 Docker 服务注册绑定功能。`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if KubeConfigPath == "" {
			KubeConfigPath = viper.GetString("k8s.kubeconfig")
		}
	},
}

func init() {
	K8sCmd.PersistentFlags().StringVar(&KubeConfigPath, "kubeconfig", "", "指定 kubeconfig 配置文件路径 (默认读取 ~/.kube/config 或 $KUBECONFIG)")
}
