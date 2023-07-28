package client

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteAllMonitoringResources deletes all resources related to monitoring from k8s cluster.
func (c *Client) DeleteAllMonitoringResources(ctx context.Context, namespace string) error {
	cl, err := c.kubeClient()
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = c.namespace
	}

	opts := []client.DeleteAllOfOption{
		client.MatchingLabels{"everest.percona.com/type": "monitoring"},
		client.InNamespace(namespace),
	}

	for _, o := range c.monitoringResourceTypesForRemoval() {
		if err := cl.DeleteAllOf(ctx, o, opts...); err != nil {
			return err
		}
	}

	return nil
}

// monitoringResourceTypesForRemoval returns a list of object types in k8s cluster to be removed
// when deleting all monitoring resources from a k8s cluster.
func (c *Client) monitoringResourceTypesForRemoval() []client.Object {
	vmNodeScrape := &unstructured.Unstructured{}
	vmNodeScrape.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.victoriametrics.com",
		Kind:    "VMNodeScrape",
		Version: "v1beta1",
	})

	vmPodScrape := &unstructured.Unstructured{}
	vmPodScrape.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.victoriametrics.com",
		Kind:    "VMPodScrape",
		Version: "v1beta1",
	})

	vmAgent := &unstructured.Unstructured{}
	vmAgent.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.victoriametrics.com",
		Kind:    "VMAgent",
		Version: "v1beta1",
	})

	vmServiceScrape := &unstructured.Unstructured{}
	vmServiceScrape.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.victoriametrics.com",
		Kind:    "VMServiceScrape",
		Version: "v1beta1",
	})

	return []client.Object{
		&corev1.ServiceAccount{},
		&corev1.Service{},
		&appsv1.Deployment{},
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},

		vmNodeScrape,
		vmPodScrape,
		vmServiceScrape,
		vmAgent,
	}
}
