package kubernetes

import "context"

// DeleteAllMonitoringResources deletes all resources related to monitoring from k8s cluster.
// If namespace is empty, a default namespace is used.
func (k *Kubernetes) DeleteAllMonitoringResources(ctx context.Context, namespace string) error {
	return k.client.DeleteAllMonitoringResources(ctx, namespace)
}
