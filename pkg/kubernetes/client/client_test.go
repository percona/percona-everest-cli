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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	fake "k8s.io/client-go/kubernetes/fake"
)

func TestGetSecretsForServiceAccount(t *testing.T) {
	t.Parallel()
	clientset := fake.NewSimpleClientset(
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pmm-service-account",
				Namespace: "default",
			},
			Secrets: []corev1.ObjectReference{
				{
					Name: "pmm-service-account-token",
				},
				{
					Name: "pmm-service-account-token-ktgqd",
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pmm-service-account-token",
				Namespace: "default",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pmm-service-account-token-ktgqd",
				Namespace: "default",
			},
		})
	client := &Client{clientset: clientset, restConfig: nil, namespace: "default"}

	ctx := context.Background()
	secret, err := client.GetSecretsForServiceAccount(ctx, "pmm-service-account")
	assert.NotNil(t, secret, "secret is nil")
	require.NoError(t, err)
}

func TestGetSecretsForServiceAccountNoSecrets(t *testing.T) {
	t.Parallel()
	clientset := fake.NewSimpleClientset(
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pmm-service-account",
				Namespace: "default",
			},
		})
	client := &Client{clientset: clientset, restConfig: nil, namespace: "default"}

	ctx := context.Background()
	secret, err := client.GetSecretsForServiceAccount(ctx, "pmm-service-account")
	assert.Nil(t, secret, "secret is not nil")
	require.Error(t, err)
}

func TestGetServerVersion(t *testing.T) {
	t.Parallel()
	clientset := fake.NewSimpleClientset()
	client := &Client{clientset: clientset, namespace: "default"}
	ver, err := client.GetServerVersion()
	expectedVersion := &version.Info{}
	require.NoError(t, err)
	assert.Equal(t, expectedVersion.Minor, ver.Minor)
}

func TestGetPods(t *testing.T) {
	t.Parallel()

	data := []struct {
		clientset         kubernetes.Interface
		countExpectedPods int
		inputNamespace    string
		err               error
	}{
		// there are no pods in the specified namespace
		{
			clientset: fake.NewSimpleClientset(&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "awesome-pod",
					Namespace: "my-safe-space",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			}, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cool-pod",
					Namespace: "get-me-outta-here",
				},
			}),
			inputNamespace:    "default",
			countExpectedPods: 0,
		},
		// there is a pod in the specified namespace
		{
			clientset: fake.NewSimpleClientset(&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pmm-0",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			}, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cool-pod",
					Namespace: "get-me-outta-here",
				},
			}),
			inputNamespace:    "default",
			countExpectedPods: 1,
		},
	}

	//nolint:paralleltest
	for _, test := range data {
		t.Run("", func(test struct {
			clientset         kubernetes.Interface
			countExpectedPods int
			inputNamespace    string
			err               error
		},
		) func(t *testing.T) {
			return func(t *testing.T) {
				t.Parallel()
				clientset := test.clientset
				client := &Client{clientset: clientset, namespace: "default"}

				pods, err := client.GetPods(context.Background(), test.inputNamespace, nil)
				if test.err == nil {
					require.NoError(t, err)
					assert.Len(t, test.countExpectedPods, len(pods.Items))
				} else {
					require.Error(t, err)
					assert.Equal(t, test.err, err)
				}
			}
		}(test))
	}
}

func TestListCRDs(t *testing.T) {
	t.Parallel()

	data := []struct {
		clientset          apiextv1clientset.Interface
		inputLabelSelector *metav1.LabelSelector
		countExpectedCRDs  int
		err                error
	}{
		// no label selector
		{
			clientset: apiextfake.NewSimpleClientset(&apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "awesome-crd",
					Labels: map[string]string{
						"custom_label_key_1": "custom_label_value_1",
					},
				},
			}, &apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cool-crd",
					Labels: map[string]string{
						"custom_label_key_2": "custom_label_value_2",
					},
				},
			}),
			inputLabelSelector: nil,
			countExpectedCRDs:  2,
		},
		// one CRD matches label selector
		{
			clientset: apiextfake.NewSimpleClientset(&apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "awesome-crd",
					Labels: map[string]string{
						"custom_label_key_1": "custom_label_value_1",
					},
				},
			}, &apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cool-crd",
					Labels: map[string]string{
						"custom_label_key_2": "custom_label_value_2",
					},
				},
			}),
			inputLabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"custom_label_key_1": "custom_label_value_1",
				},
			},
			countExpectedCRDs: 1,
		},
		// two CRDs match label selector
		{
			clientset: apiextfake.NewSimpleClientset(&apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "awesome-crd",
					Labels: map[string]string{
						"custom_label_key_1": "custom_label_value_1",
					},
				},
			}, &apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cool-crd",
					Labels: map[string]string{
						"custom_label_key_1": "custom_label_value_1",
						"custom_label_key_2": "custom_label_value_2",
					},
				},
			}, &apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "another-crd",
					Labels: map[string]string{
						"custom_label_key_3": "custom_label_value_1",
					},
				},
			}),
			inputLabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"custom_label_key_1": "custom_label_value_1",
				},
			},
			countExpectedCRDs: 2,
		},
		// one CRD matches label selector with multiple labels
		{
			clientset: apiextfake.NewSimpleClientset(&apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "awesome-crd",
					Labels: map[string]string{
						"custom_label_key_1": "custom_label_value_1",
					},
				},
			}, &apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cool-crd",
					Labels: map[string]string{
						"custom_label_key_1": "custom_label_value_1",
						"custom_label_key_2": "custom_label_value_2",
					},
				},
			}, &apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "another-crd",
					Labels: map[string]string{
						"custom_label_key_3": "custom_label_value_1",
					},
				},
			}),
			inputLabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"custom_label_key_1": "custom_label_value_1",
					"custom_label_key_2": "custom_label_value_2",
				},
			},
			countExpectedCRDs: 1,
		},
	}

	//nolint:paralleltest
	for _, test := range data {
		t.Run("", func(test struct {
			clientset          apiextv1clientset.Interface
			inputLabelSelector *metav1.LabelSelector
			countExpectedCRDs  int
			err                error
		},
		) func(t *testing.T) {
			return func(t *testing.T) {
				t.Parallel()
				clientset := test.clientset
				client := &Client{apiextClientset: clientset, namespace: "default"}

				crds, err := client.ListCRDs(context.Background(), test.inputLabelSelector)
				if test.err == nil {
					require.NoError(t, err)
					assert.Len(t, test.countExpectedCRDs, len(crds.Items))
				} else {
					require.Error(t, err)
					assert.Equal(t, test.err, err)
				}
			}
		}(test))
	}
}

//nolint:maintidx
func TestListCRs(t *testing.T) {
	t.Parallel()

	data := []struct {
		clientset          dynamic.Interface
		inputNamespace     string
		inputGVR           schema.GroupVersionResource
		inputLabelSelector *metav1.LabelSelector
		countExpectedCRs   int
		err                error
	}{
		// one CR matches namespace
		{
			clientset: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
				map[schema.GroupVersionResource]string{
					{Group: "everest.percona.com", Version: "v1alpha1", Resource: "mycoolkinds"}: "MyCoolKindList",
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "awesome-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_1": "custom_label_value_1",
							},
						},
					},
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "another-cr",
							"namespace": "get-me-outta-here",
							"labels": map[string]interface{}{
								"custom_label_key_1": "custom_label_value_1",
							},
						},
					},
				}),
			inputNamespace: "my-safe-space",
			inputGVR: schema.GroupVersionResource{
				Group:    "everest.percona.com",
				Version:  "v1alpha1",
				Resource: "mycoolkinds",
			},
			inputLabelSelector: nil,
			countExpectedCRs:   1,
		},
		// one CR matches GVR
		{
			clientset: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
				map[schema.GroupVersionResource]string{
					{Group: "everest.percona.com", Version: "v1alpha1", Resource: "mycoolkinds"}:    "MyCoolKindList",
					{Group: "everest.percona.com", Version: "v1alpha1", Resource: "othercoolkinds"}: "OtherKindList",
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "awesome-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_1": "custom_label_value_1",
							},
						},
					},
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "OtherKind",
						"metadata": map[string]interface{}{
							"name":      "another-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_1": "custom_label_value_1",
							},
						},
					},
				}),
			inputNamespace: "my-safe-space",
			inputGVR: schema.GroupVersionResource{
				Group:    "everest.percona.com",
				Version:  "v1alpha1",
				Resource: "mycoolkinds",
			},
			inputLabelSelector: nil,
			countExpectedCRs:   1,
		},
		// no label selector
		{
			clientset: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
				map[schema.GroupVersionResource]string{
					{Group: "everest.percona.com", Version: "v1alpha1", Resource: "mycoolkinds"}: "MyCoolKindList",
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "awesome-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_1": "custom_label_value_1",
							},
						},
					},
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "cool-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_2": "custom_label_value_2",
							},
						},
					},
				}),
			inputNamespace: "my-safe-space",
			inputGVR: schema.GroupVersionResource{
				Group:    "everest.percona.com",
				Version:  "v1alpha1",
				Resource: "mycoolkinds",
			},
			inputLabelSelector: nil,
			countExpectedCRs:   2,
		},
		// one CR matches label selector
		{
			clientset: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
				map[schema.GroupVersionResource]string{
					{Group: "everest.percona.com", Version: "v1alpha1", Resource: "mycoolkinds"}: "MyCoolKindList",
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "awesome-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_1": "custom_label_value_1",
							},
						},
					},
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "cool-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_2": "custom_label_value_2",
							},
						},
					},
				}),
			inputNamespace: "my-safe-space",
			inputGVR: schema.GroupVersionResource{
				Group:    "everest.percona.com",
				Version:  "v1alpha1",
				Resource: "mycoolkinds",
			},
			inputLabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"custom_label_key_1": "custom_label_value_1",
				},
			},
			countExpectedCRs: 1,
		},
		// two CRs match label selector
		{
			//nolint:dupl
			clientset: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
				map[schema.GroupVersionResource]string{
					{Group: "everest.percona.com", Version: "v1alpha1", Resource: "mycoolkinds"}: "MyCoolKindList",
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "awesome-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_1": "custom_label_value_1",
							},
						},
					},
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "cool-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_1": "custom_label_value_1",
								"custom_label_key_2": "custom_label_value_2",
							},
						},
					},
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "another-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_3": "custom_label_value_1",
							},
						},
					},
				}),
			inputNamespace: "my-safe-space",
			inputGVR: schema.GroupVersionResource{
				Group:    "everest.percona.com",
				Version:  "v1alpha1",
				Resource: "mycoolkinds",
			},
			inputLabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"custom_label_key_1": "custom_label_value_1",
				},
			},
			countExpectedCRs: 2,
		},
		// one CR matches label selector with multiple labels
		{
			//nolint:dupl
			clientset: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
				map[schema.GroupVersionResource]string{
					{Group: "everest.percona.com", Version: "v1alpha1", Resource: "mycoolkinds"}: "MyCoolKindList",
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "awesome-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_1": "custom_label_value_1",
							},
						},
					},
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "cool-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_1": "custom_label_value_1",
								"custom_label_key_2": "custom_label_value_2",
							},
						},
					},
				}, &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "everest.percona.com/v1alpha1",
						"kind":       "MyCoolKind",
						"metadata": map[string]interface{}{
							"name":      "another-cr",
							"namespace": "my-safe-space",
							"labels": map[string]interface{}{
								"custom_label_key_3": "custom_label_value_1",
							},
						},
					},
				}),
			inputNamespace: "my-safe-space",
			inputGVR: schema.GroupVersionResource{
				Group:    "everest.percona.com",
				Version:  "v1alpha1",
				Resource: "mycoolkinds",
			},
			inputLabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"custom_label_key_1": "custom_label_value_1",
					"custom_label_key_2": "custom_label_value_2",
				},
			},
			countExpectedCRs: 1,
		},
	}

	//nolint:paralleltest
	for _, test := range data {
		t.Run("", func(test struct {
			clientset          dynamic.Interface
			inputNamespace     string
			inputGVR           schema.GroupVersionResource
			inputLabelSelector *metav1.LabelSelector
			countExpectedCRs   int
			err                error
		},
		) func(t *testing.T) {
			return func(t *testing.T) {
				t.Parallel()
				clientset := test.clientset
				client := &Client{dynamicClientset: clientset, namespace: "default"}

				crds, err := client.ListCRs(context.Background(), test.inputNamespace, test.inputGVR, test.inputLabelSelector)
				if test.err == nil {
					require.NoError(t, err)
					assert.Len(t, test.countExpectedCRs, len(crds.Items))
				} else {
					require.Error(t, err)
					assert.Equal(t, test.err, err)
				}
			}
		}(test))
	}
}
