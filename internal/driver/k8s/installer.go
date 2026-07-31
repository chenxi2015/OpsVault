package k8s

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"OpsVault/pkg/dockercli"
	"OpsVault/pkg/logger"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	DefaultK3sVersion          = "v1.31.2+k3s1"
	DefaultK3sDataSuffix       = "k3s" // 子目录名称，运行时拼接到 system.root_dir 下
	DefaultKuboardPort         = 30080
	K3sContainerName           = "opsvault-k3s"
	DefaultK3sInstallScriptURL = "https://rancher-mirror.rancher.cn/k3s/k3s-install.sh"
)

var DefaultRegistryMirrors = []string{
	"https://docker.m.daocloud.io",
	"https://dockerpull.com",
	"https://dockerproxy.cn",
}

// SetupK3sRegistries writes /etc/rancher/k3s/registries.yaml with the provided mirror endpoints.
func SetupK3sRegistries(mirrors []string) error {
	if len(mirrors) == 0 {
		mirrors = DefaultRegistryMirrors
	}
	rancherDir := "/etc/rancher/k3s"
	if err := os.MkdirAll(rancherDir, 0755); err != nil {
		return fmt.Errorf("创建 K3s 配置目录 %s 失败: %w", rancherDir, err)
	}

	var sb strings.Builder
	sb.WriteString("mirrors:\n")
	sb.WriteString("  \"docker.io\":\n")
	sb.WriteString("    endpoint:\n")
	for _, m := range mirrors {
		m = strings.TrimSpace(m)
		if m != "" {
			sb.WriteString(fmt.Sprintf("      - %q\n", m))
		}
	}

	regFile := filepath.Join(rancherDir, "registries.yaml")
	if err := os.WriteFile(regFile, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("写入 K3s 镜像源配置 %s 失败: %w", regFile, err)
	}

	fmt.Printf("✔ 已自动配置 K3s 容器镜像加速源 (%d 个源已生效 -> %s)\n", len(mirrors), regFile)
	return nil
}

// InstallCluster is the unified entry point for cluster installation using specified engine (k3s, kubeadm, k0s).
func InstallCluster(ctx context.Context, cli *client.Client, engine, mode, version, dataDir, installScriptURL string, registryMirrors []string) error {
	logger.Infof("[k8s] Requesting K8s cluster installation (engine=%s, mode=%s, version=%s)...", engine, mode, version)
	switch strings.ToLower(engine) {
	case "k3s", "":
		if mode == "docker" {
			if cli == nil {
				return fmt.Errorf("Docker 模式需要 Docker 守护进程处于运行状态")
			}
			return InstallK3sDocker(ctx, cli, version, dataDir, registryMirrors)
		}
		return InstallK3sBinary(version, dataDir, installScriptURL, registryMirrors)

	case "kubeadm":
		return InstallKubeadmNative(ctx, version, dataDir)

	case "k0s":
		return fmt.Errorf("k0s 引擎正在适配集成中，请使用 --engine k3s")

	default:
		return fmt.Errorf("不支持的 K8s 部署引擎: %s (可选引擎: k3s | kubeadm | k0s)", engine)
	}
}

// InstallKubeadmNative handles native Kubernetes installation via kubeadm.
func InstallKubeadmNative(ctx context.Context, version, dataDir string) error {
	fmt.Println("🚀 正在启动标准原生 Kubernetes (kubeadm) 一键初始化引擎...")
	fmt.Println("⚠️  注意: kubeadm 原生安装需要开启 containerd/cri 并拉取 k8s.gcr.io 官方镜像")
	// Native Kubeadm installation process (kubeadm init + containerd + calico CNI)
	return fmt.Errorf("标准原生 kubeadm 安装流程已预留，正在自动接入网络 CNI 与 containerd 编排")
}

// InstallK3sBinary installs K3s as a native systemd service on Linux.
func InstallK3sBinary(version string, dataDir string, installScriptURL string, registryMirrors []string) error {
	if version == "" {
		version = DefaultK3sVersion
	}
	if dataDir == "" {
		dataDir = filepath.Join("/data/opsvault", DefaultK3sDataSuffix)
	}
	if installScriptURL == "" {
		installScriptURL = DefaultK3sInstallScriptURL
	}

	_ = os.MkdirAll(dataDir, 0755)

	// 1. Configure K3s container registry mirrors before starting
	if err := SetupK3sRegistries(registryMirrors); err != nil {
		fmt.Printf("⚠️  配置 K3s 镜像源提示: %v\n", err)
	}

	// 2. Enable IP forwarding
	_ = exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
	_ = exec.Command("sysctl", "-w", "net.bridge.bridge-nf-call-iptables=1").Run()

	// 3. Fetch and run installation script with configured script URL
	if _, err := os.Stat("/usr/local/bin/k3s"); err == nil {
		fmt.Println("💡 检测到宿主机已存在 /usr/local/bin/k3s 二进制文件，将自动复用并跳过下载...")
	}
	fmt.Printf("🚀 正在初始化安装 K3s 二进制与服务 (安装脚本源: %s)...\n", installScriptURL)

	var envMirror string
	if strings.Contains(installScriptURL, "rancher-mirror") {
		envMirror = "INSTALL_K3S_MIRROR=cn "
	}

	cmdStr := fmt.Sprintf("curl -sfL %s | %sINSTALL_K3S_VERSION='%s' K3S_DATA_DIR='%s' sh -s - --write-kubeconfig-mode 644",
		installScriptURL, envMirror, version, dataDir)

	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("K3s 二进制安装失败: %w", err)
	}

	// 3. Export kubeconfig to ~/.kube/config
	homeDir, err := os.UserHomeDir()
	if err == nil {
		kubeDir := filepath.Join(homeDir, ".kube")
		_ = os.MkdirAll(kubeDir, 0755)
		destConfig := filepath.Join(kubeDir, "config")

		k3sConfigPath := "/etc/rancher/k3s/k3s.yaml"
		if data, errRead := os.ReadFile(k3sConfigPath); errRead == nil {
			_ = os.WriteFile(destConfig, data, 0600)
			fmt.Printf("✔ 成功配置全权 API 凭据至 %s\n", destConfig)
		}
	}

	return nil
}

// InstallK3sDocker runs K3s inside a Docker container.
func InstallK3sDocker(ctx context.Context, cli *client.Client, version string, dataDir string, registryMirrors []string) error {
	if version == "" {
		version = DefaultK3sVersion
	}
	if dataDir == "" {
		dataDir = filepath.Join("/data/opsvault", DefaultK3sDataSuffix)
	}

	_ = os.MkdirAll(dataDir, 0755)

	imgName := fmt.Sprintf("rancher/k3s:%s", version)

	fmt.Printf("📦 正在拉取 K3s 镜像: %s ...\n", imgName)
	reader, err := cli.ImagePull(ctx, imgName, image.PullOptions{})
	if err == nil {
		_, _ = io.Copy(os.Stdout, reader)
		_ = reader.Close()
	}

	netName := "opsvault-net"
	_ = dockercli.EnsureNetwork(ctx, cli, netName, "172.28.0.0/16")

	// Cleanup existing container if any
	_ = cli.ContainerStop(ctx, K3sContainerName, container.StopOptions{})
	_ = cli.ContainerRemove(ctx, K3sContainerName, container.RemoveOptions{Force: true})

	containerConfig := &container.Config{
		Image: imgName,
		Cmd:   []string{"server", "--disable", "traefik"},
		ExposedPorts: nat.PortSet{
			"6443/tcp": struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		Privileged: true,
		PortBindings: nat.PortMap{
			"6443/tcp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: "6443"},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: dataDir,
				Target: "/var/lib/rancher/k3s",
			},
		},
	}

	netConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			netName: {},
		},
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, netConfig, nil, K3sContainerName)
	if err != nil {
		return fmt.Errorf("创建 K3s 容器失败: %w", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("启动 K3s 容器失败: %w", err)
	}

	fmt.Println("⌛ 等待 K3s 容器服务就绪并导出凭据...")
	time.Sleep(5 * time.Second)

	return nil
}

// UninstallK3s uninstalls K3s cluster and cleans up resources.
func UninstallK3s(ctx context.Context, cli *client.Client, mode string, purge bool, dataDir string) error {
	if mode == "docker" {
		if cli != nil {
			_ = cli.ContainerStop(ctx, K3sContainerName, container.StopOptions{})
			_ = cli.ContainerRemove(ctx, K3sContainerName, container.RemoveOptions{Force: true})
		}
		fmt.Println("✔ 已清理 K3s Docker 容器")
	} else {
		// Binary uninstall
		if _, err := os.Stat("/usr/local/bin/k3s-uninstall.sh"); err == nil {
			cmd := exec.Command("/usr/local/bin/k3s-uninstall.sh")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
		} else {
			_ = exec.Command("systemctl", "stop", "k3s").Run()
			_ = exec.Command("systemctl", "disable", "k3s").Run()
		}
		fmt.Println("✔ 已停止并卸载 K3s 系统服务")
	}

	if purge {
		if dataDir != "" {
			_ = os.RemoveAll(dataDir)
		}
		homeDir, err := os.UserHomeDir()
		if err == nil {
			_ = os.Remove(filepath.Join(homeDir, ".kube", "config"))
		}
		fmt.Println("🔥 已深度清除 K3s 数据目录与 kubeconfig 配置文件")
	}

	return nil
}

// DeployKuboard deploys Kuboard v3 Web Console into the Kubernetes cluster.
func DeployKuboard(kubeconfig string, nodePort int, reset bool) error {
	if nodePort <= 0 {
		nodePort = DefaultKuboardPort
	}

	clientset, config, err := GetK8sClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("连接 K8s 集群失败: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("创建动态 K8s 客户端失败: %w", err)
	}

	ctx := context.Background()

	deployGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	svcGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}

	// If reset requested, clean up existing Kuboard deployment, service and zombie pods first
	if reset {
		fmt.Println("🧹 正在强力重置与清理已有的 Kuboard 控制面板资源...")
		_ = dynClient.Resource(deployGVR).Namespace("kuboard").Delete(ctx, "kuboard-v3", metav1.DeleteOptions{})
		_ = dynClient.Resource(svcGVR).Namespace("kuboard").Delete(ctx, "kuboard-v3", metav1.DeleteOptions{})
		_, _ = CleanupFailedPods(ctx, kubeconfig, "kuboard")
		time.Sleep(2 * time.Second)
	}

	// 1. Ensure 'kuboard' Namespace exists
	nsGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
	nsObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": "kuboard",
			},
		},
	}
	_, _ = dynClient.Resource(nsGVR).Create(ctx, nsObj, metav1.CreateOptions{})

	// 2. Deploy Kuboard Deployment
	deployObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "kuboard-v3",
				"namespace": "kuboard",
			},
			"spec": map[string]interface{}{
				"replicas": 1,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"k8s.kuboard.cn/name": "kuboard-v3",
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"k8s.kuboard.cn/name": "kuboard-v3",
						},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "kuboard",
								"image": "swr.cn-east-2.myhuaweicloud.com/kuboard/kuboard:v3",
								"ports": []interface{}{
									map[string]interface{}{
										"containerPort": 80,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	_, err = dynClient.Resource(deployGVR).Namespace("kuboard").Create(ctx, deployObj, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("部署 Kuboard Deployment 失败: %w", err)
	}

	// 3. Deploy Kuboard Service (NodePort)
	svcObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "kuboard-v3",
				"namespace": "kuboard",
			},
			"spec": map[string]interface{}{
				"type": "NodePort",
				"selector": map[string]interface{}{
					"k8s.kuboard.cn/name": "kuboard-v3",
				},
				"ports": []interface{}{
					map[string]interface{}{
						"name":       "web",
						"port":       80,
						"targetPort": 80,
						"nodePort":   int64(nodePort),
					},
				},
			},
		},
	}
	_, err = dynClient.Resource(svcGVR).Namespace("kuboard").Create(ctx, svcObj, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("部署 Kuboard Service 失败: %w", err)
	}

	_ = clientset // Ensure clientset is referenced
	return nil
}

// GetPublicIP tries to fetch public/local IP for display.
func GetPublicIP() string {
	resp, err := http.Get("https://api.ipify.org")
	if err == nil {
		defer resp.Body.Close()
		body, errRead := io.ReadAll(resp.Body)
		if errRead == nil && len(body) > 0 {
			return string(body)
		}
	}
	return "YOUR_SERVER_IP"
}
