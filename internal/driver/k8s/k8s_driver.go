package k8s

import (
	"context"
	"fmt"
	"time"

	"OpsVault/internal/driver"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

type K8sDriver struct {
	ServiceName string
	Clientset   *kubernetes.Clientset
}

func NewK8sDriver(serviceName string, kubeconfig string) (*K8sDriver, error) {
	client, _, err := GetK8sClient(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client for %s: %w", serviceName, err)
	}
	return &K8sDriver{
		ServiceName: serviceName,
		Clientset:   client,
	}, nil
}

// RegisterK8sService creates a Kubernetes Service + Endpoints (or ExternalName) for Docker host services.
func (d *K8sDriver) RegisterK8sService(namespace string, targetIP string, targetPort int) error {
	if namespace == "" {
		namespace = "default"
	}
	ctx := context.Background()

	svcName := fmt.Sprintf("opsvault-%s", d.ServiceName)

	// 1. Create or update Headless / ClusterIP Service without selector
	svcClient := d.Clientset.CoreV1().Services(namespace)
	existingSvc, err := svcClient.Get(ctx, svcName, metav1.GetOptions{})

	svcSpec := corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name:       fmt.Sprintf("%s-port", d.ServiceName),
				Port:       int32(targetPort),
				TargetPort: intstr.FromInt(targetPort),
				Protocol:   corev1.ProtocolTCP,
			},
		},
	}

	if errors.IsNotFound(err) {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcName,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "opsvault",
					"opsvault/service":             d.ServiceName,
				},
			},
			Spec: svcSpec,
		}
		_, err = svcClient.Create(ctx, svc, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create k8s service %s: %w", svcName, err)
		}
	} else if err == nil {
		existingSvc.Spec.Ports = svcSpec.Ports
		_, err = svcClient.Update(ctx, existingSvc, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update k8s service %s: %w", svcName, err)
		}
	} else {
		return err
	}

	// 2. Create or update Endpoints mapping to targetIP:targetPort
	epClient := d.Clientset.CoreV1().Endpoints(namespace)
	existingEp, err := epClient.Get(ctx, svcName, metav1.GetOptions{})

	epSubset := []corev1.EndpointSubset{
		{
			Addresses: []corev1.EndpointAddress{
				{IP: targetIP},
			},
			Ports: []corev1.EndpointPort{
				{
					Name:     fmt.Sprintf("%s-port", d.ServiceName),
					Port:     int32(targetPort),
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	if errors.IsNotFound(err) {
		ep := &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcName,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "opsvault",
					"opsvault/service":             d.ServiceName,
				},
			},
			Subsets: epSubset,
		}
		_, err = epClient.Create(ctx, ep, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create k8s endpoints %s: %w", svcName, err)
		}
	} else if err == nil {
		existingEp.Subsets = epSubset
		_, err = epClient.Update(ctx, existingEp, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update k8s endpoints %s: %w", svcName, err)
		}
	} else {
		return err
	}

	return nil
}

// UnregisterK8sService removes the registered K8s service and endpoints.
func (d *K8sDriver) UnregisterK8sService(namespace string) error {
	if namespace == "" {
		namespace = "default"
	}
	ctx := context.Background()
	svcName := fmt.Sprintf("opsvault-%s", d.ServiceName)

	_ = d.Clientset.CoreV1().Services(namespace).Delete(ctx, svcName, metav1.DeleteOptions{})
	_ = d.Clientset.CoreV1().Endpoints(namespace).Delete(ctx, svcName, metav1.DeleteOptions{})
	return nil
}

func (d *K8sDriver) Status() (*driver.ServiceStatus, error) {
	ctx := context.Background()
	svcName := fmt.Sprintf("opsvault-%s", d.ServiceName)

	svc, err := d.Clientset.CoreV1().Services("default").Get(ctx, svcName, metav1.GetOptions{})
	if err != nil {
		return &driver.ServiceStatus{
			Name:      d.ServiceName,
			Mode:      driver.ModeK8s,
			Running:   false,
			Status:    "Not Registered / Not Found",
			UpdatedAt: time.Now(),
		}, nil
	}

	var ports []string
	for _, p := range svc.Spec.Ports {
		ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
	}

	return &driver.ServiceStatus{
		Name:      d.ServiceName,
		Mode:      driver.ModeK8s,
		Running:   true,
		Status:    "Registered in K8s",
		Ports:     ports,
		UpdatedAt: time.Now(),
	}, nil
}

func (d *K8sDriver) Install() error {
	return nil
}

func (d *K8sDriver) Start() error {
	return nil
}

func (d *K8sDriver) Stop() error {
	return nil
}

func (d *K8sDriver) Restart() error {
	return nil
}

func (d *K8sDriver) Uninstall(purgeData bool) error {
	return d.UnregisterK8sService("default")
}

func (d *K8sDriver) Upgrade(targetVersion string) error {
	return nil
}
