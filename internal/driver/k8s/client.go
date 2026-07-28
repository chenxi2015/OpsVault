package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetK8sClient initializes and returns a Kubernetes clientset.
// It searches in order: specified path, KUBECONFIG env, ~/.kube/config, in-cluster config.
func GetK8sClient(customKubeConfig string) (*kubernetes.Clientset, *rest.Config, error) {
	var config *rest.Config
	var err error

	kubeconfigPath := customKubeConfig
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}
	if kubeconfigPath == "" {
		homeDir, errHome := os.UserHomeDir()
		if errHome == nil {
			defaultPath := filepath.Join(homeDir, ".kube", "config")
			if _, errStat := os.Stat(defaultPath); errStat == nil {
				kubeconfigPath = defaultPath
			}
		}
	}

	if kubeconfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build kubeconfig from %s: %w", kubeconfigPath, err)
		}
	} else {
		// Fallback to in-cluster configuration
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("kubeconfig not found and in-cluster config failed: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return clientset, config, nil
}
