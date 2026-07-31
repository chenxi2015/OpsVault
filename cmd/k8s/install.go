package k8s

import (
	"context"
	"fmt"
	"path/filepath"

	k8sdriver "OpsVault/internal/driver/k8s"
	"OpsVault/pkg/dockercli"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	installEngine          string
	installMode            string
	installVersion         string
	installDataDir         string
	installScriptURL       string
	installRegistryMirrors []string
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "一键安装 Kubernetes 集群 (支持引擎: k3s | kubeadm | k0s)",
	RunE: func(cmd *cobra.Command, args []string) error {
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
		successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
		highlightStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))

		cfg := getConfig()
		if !cmd.Flags().Changed("engine") && cfg.GetString("k8s.engine") != "" {
			installEngine = cfg.GetString("k8s.engine")
		}
		if !cmd.Flags().Changed("version") && cfg.GetString("k8s.version") != "" {
			installVersion = cfg.GetString("k8s.version")
		}
		if !cmd.Flags().Changed("install-script-url") && cfg.GetString("k8s.install_script_url") != "" {
			installScriptURL = cfg.GetString("k8s.install_script_url")
		}
		if !cmd.Flags().Changed("registry-mirror") {
			if mirrors := cfg.GetStringSlice("k8s.registry_mirrors"); len(mirrors) > 0 {
				installRegistryMirrors = mirrors
			} else {
				installRegistryMirrors = k8sdriver.DefaultRegistryMirrors
			}
		}
		if !cmd.Flags().Changed("data-dir") {
			if cfg.GetString("k8s.data_dir") != "" {
				installDataDir = cfg.GetString("k8s.data_dir")
			} else {
				rootDir := cfg.GetString("system.root_dir")
				if rootDir == "" {
					rootDir = "/data/opsvault"
				}
				installDataDir = filepath.Join(rootDir, k8sdriver.DefaultK3sDataSuffix)
			}
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
		fmt.Printf("● 数据存储  : %s\n", installDataDir)
		fmt.Printf("● 镜像源加速: %v\n\n", installRegistryMirrors)

		cli, _ := dockercli.New()
		if cli != nil {
			defer cli.Close()
		}

		ctx := context.Background()

		err := k8sdriver.InstallCluster(ctx, cli, installEngine, installMode, installVersion, installDataDir, installScriptURL, installRegistryMirrors)
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
	installCmd.Flags().StringVar(&installDataDir, "data-dir", "", "宿主机持久化数据根目录（留空则自动使用 {system.root_dir}/k3s）")
	installCmd.Flags().StringVar(&installScriptURL, "install-script-url", "", "K3s 二进制安装脚本下载 URL（留空则优先使用 k8s.install_script_url 配置）")
	installCmd.Flags().StringSliceVar(&installRegistryMirrors, "registry-mirror", nil, "K3s / Containerd 容器镜像加速源 (支持多次指定或逗号分隔，默认使用国内主流镜像源)")

	K8sCmd.AddCommand(installCmd)
}
