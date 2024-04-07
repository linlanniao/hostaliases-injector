package v1

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func (m *Mutate) getServiceHostsAliases() ([]corev1.HostAlias, error) {
	var services corev1.ServiceList
	if err := m.Client.List(context.TODO(), &services); err != nil {
		return nil, err
	}

	if len(services.Items) == 0 {
		return nil, nil
	}

	hostAliases := make([]corev1.HostAlias, 0)

	for _, service := range services.Items {
		if service.Spec.Type != corev1.ServiceTypeClusterIP {
			continue
		}
		if service.Spec.ClusterIP == "" || service.Spec.ClusterIP == "None" {
			continue
		}

		domain1 := fmt.Sprintf("%s.%s.svc.cluster.local", service.GetName(), service.GetNamespace())
		domain2 := fmt.Sprintf("%s.%s.svc", service.GetName(), service.GetNamespace())
		domain3 := fmt.Sprintf("%s.%s", service.GetName(), service.GetNamespace())
		hostAliases = append(hostAliases, corev1.HostAlias{
			IP:        service.Spec.ClusterIP,
			Hostnames: []string{domain1, domain2, domain3},
		})
	}

	return hostAliases, nil
}
