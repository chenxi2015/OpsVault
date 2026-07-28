package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ClusterHealth struct {
	TotalNodes     int
	ReadyNodes     int
	TotalPods      int
	RunningPods    int
	AbnormalPods   []PodCheckResult
	NodeSummary    []NodeCheckResult
	CheckTimestamp time.Time
}

type NodeCheckResult struct {
	Name       string
	Status     string
	KubeletVer string
	OSImage    string
	CPU        string
	Memory     string
}

type PodCheckResult struct {
	Namespace string
	Name      string
	Status    string
	Reason    string
	Restarts  int32
	NodeName  string
}

// InspectCluster performs a health check across Nodes and Pods in the cluster.
func InspectCluster(ctx context.Context, clientset *kubernetes.Clientset) (*ClusterHealth, error) {
	health := &ClusterHealth{
		CheckTimestamp: time.Now(),
	}

	// 1. Inspect Nodes
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	health.TotalNodes = len(nodes.Items)

	for _, node := range nodes.Items {
		isReady := false
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				isReady = true
				break
			}
		}

		statusStr := "NotReady"
		if isReady {
			isReadyCount := 1
			health.ReadyNodes += isReadyCount
			statusStr = "Ready"
		}

		cpuQuantity := node.Status.Capacity[corev1.ResourceCPU]
		memQuantity := node.Status.Capacity[corev1.ResourceMemory]

		health.NodeSummary = append(health.NodeSummary, NodeCheckResult{
			Name:       node.Name,
			Status:     statusStr,
			KubeletVer: node.Status.NodeInfo.KubeletVersion,
			OSImage:    node.Status.NodeInfo.OSImage,
			CPU:        cpuQuantity.String(),
			Memory:     memQuantity.String(),
		})
	}

	// 2. Inspect Pods across all namespaces
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	health.TotalPods = len(pods.Items)

	for _, pod := range pods.Items {
		var totalRestarts int32
		var abnormalReason []string

		for _, cs := range pod.Status.ContainerStatuses {
			totalRestarts += cs.RestartCount
			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				abnormalReason = append(abnormalReason, cs.State.Waiting.Reason)
			}
			if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" && cs.State.Terminated.Reason != "Completed" {
				abnormalReason = append(abnormalReason, cs.State.Terminated.Reason)
			}
		}

		isAbnormal := false
		if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded {
			isAbnormal = true
		} else if len(abnormalReason) > 0 {
			isAbnormal = true
		} else if totalRestarts > 5 {
			isAbnormal = true
		}

		if pod.Status.Phase == corev1.PodRunning && !isAbnormal {
			health.RunningPods++
		}

		if isAbnormal {
			reasonStr := pod.Status.Reason
			if len(abnormalReason) > 0 {
				reasonStr = strings.Join(abnormalReason, ",")
			}
			if reasonStr == "" {
				reasonStr = string(pod.Status.Phase)
			}

			health.AbnormalPods = append(health.AbnormalPods, PodCheckResult{
				Namespace: pod.Namespace,
				Name:      pod.Name,
				Status:    string(pod.Status.Phase),
				Reason:    reasonStr,
				Restarts:  totalRestarts,
				NodeName:  pod.Spec.NodeName,
			})
		}
	}

	return health, nil
}
