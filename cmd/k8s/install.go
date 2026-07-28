package k8s

import (
	"context"
	"fmt"

	k8sdriver "OpsVault/internal/driver/k8s"
	"OpsVault/pkg/dockercli"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	installEngine  string
	installMode    string
	installVersion string
	installDataDir string
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "一键安装 Kubernetes 集群 (支持引擎: k3s | kubeadm | k0s)",
	RunE: func(cmd *cobra.Command, args []string) error {
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
		successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
		highlightStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))

		if !cmd.Flags().Changed("engine") && viper.GetString("k8s.engine") != "" {
			installEngine = viper.GetString("k8s.engine")
		}
		if !cmd.Flags().Changed("version") && viper.GetString("k8s.version") != "" {
			installVersion = viper.GetString("k8s.version")
		}
		if !cmd.Flags().Changed("data-dir") && viper.GetString("k8s.data_dir") != "" {
			installDataDir = viper.GetString("k8s.data_dir")
		}

		engineDesc := installEngine
		switch installEngine {
		case "k3s":
			engineDesc = "K3s (轻量级 CNCF 认证版，可用 --engine kubeadm 切换原生版)"
		case "kubeadm":
			engineDesc = "Kubeadm (标准原生 Kubernetes)"
		}

		fmt.Println(titleStyle.Render("☸️  开始一键安装 Kubernetes 集群..."))
		fmt.Printf("● 部署引擎  : %s\n", highlightStyle.Render(engineDesc))
		fmt.Printf("● 运行模式  : %s\n", highlightStyle.Render(installMode))
		fmt.Printf("● 目标版本  : %s\n", installVersion)
		fmt.Printf("● 数据存储  : %s\n\n", installDataDir)

		cli, _ := dockercli.New()
		if cli != nil {
			defer cli.Close()
		}

		ctx := context.Background()

		err := k8sdriver.InstallCluster(ctx, cli, installEngine, installMode, installVersion, installDataDir)
		if err != nil {
			return err
		}

		fmt.Println("\n" + successStyle.Render("🎉 Kubernetes 集群安装并部署成功！"))
		fmt.Println("💡 全权凭据已自动挂载写入 ~/.kube/config")
		fmt.Println("👉 现在你可以直接运行以下 OpsVault 命令进行集群管理:")
		fmt.Println(highlightStyle.Render("   opsvault k8s status"))
		fmt.Println(highlightStyle.Render("   opsvault k8s doctor"))
		fmt.Println(highlightStyle.Render("   opsvault k8s get pods"))
		fmt.Println(highlightStyle.Render("   opsvault k8s dashboard"))

		return nil
	},
}

func init() {
	installCmd.Flags().StringVar(&installEngine, "engine", "k3s", "集群部署引擎: k3s (轻量级认证版) | kubeadm (标准原生版) | k0s")
	installCmd.Flags().StringVar(&installMode, "mode", "binary", "安装运行模式: binary (系统服务) | docker (容器模式)")
	installCmd.Flags().StringVar(&installVersion, "version", k8sdriver.DefaultK3sVersion, "Kubernetes / K3s 集群版本")
	installCmd.Flags().StringVar(&installDataDir, "data-dir", k8sdriver.DefaultK3sDataDir, "宿主机持久化数据根目录")

	K8sCmd.AddCommand(installCmd)
}
