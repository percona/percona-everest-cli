// percona-everest-cli
// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package client ...
package client

import (
	"context"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
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
			var discoveryError *discovery.ErrGroupDiscoveryFailed
			if ok := errors.As(err, &discoveryError); !ok {
				return err
			}
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
