// Code generated by ifacemaker; DO NOT EDIT.

package client

import (
	"context"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// KubeClientConnector ...
type KubeClientConnector interface {
	// ClusterName returns the name of the k8s cluster.
	ClusterName() string
	// GetSecretsForServiceAccount returns secret by given service account name.
	GetSecretsForServiceAccount(ctx context.Context, accountName string) (*corev1.Secret, error)
	// GenerateKubeConfigWithToken generates kubeconfig with a user and token provided as a secret.
	GenerateKubeConfigWithToken(user string, secret *corev1.Secret) ([]byte, error)
	// GetServerVersion returns server version.
	GetServerVersion() (*version.Info, error)
	// ListDatabaseClusters returns list of managed PCX clusters.
	ListDatabaseClusters(ctx context.Context) (*dbaasv1.DatabaseClusterList, error)
	// GetDatabaseCluster returns PXC clusters by provided name.
	GetDatabaseCluster(ctx context.Context, name string) (*dbaasv1.DatabaseCluster, error)
	// GetStorageClasses returns all storage classes available in the cluster.
	GetStorageClasses(ctx context.Context) (*storagev1.StorageClassList, error)
	// GetDeployment returns deployment by name.
	GetDeployment(ctx context.Context, name string, namespace string) (*appsv1.Deployment, error)
	// GetSecret returns secret by name.
	GetSecret(ctx context.Context, name string) (*corev1.Secret, error)
	// ListSecrets returns secrets.
	ListSecrets(ctx context.Context) (*corev1.SecretList, error)
	// DeleteObject deletes object from the k8s cluster.
	DeleteObject(obj runtime.Object) error
	// ApplyObject applies object.
	ApplyObject(obj runtime.Object) error
	// GetPersistentVolumes returns Persistent Volumes available in the cluster.
	GetPersistentVolumes(ctx context.Context) (*corev1.PersistentVolumeList, error)
	// GetPods returns list of pods.
	GetPods(ctx context.Context, namespace string, labelSelector *metav1.LabelSelector) (*corev1.PodList, error)
	// GetNodes returns list of nodes.
	GetNodes(ctx context.Context) (*corev1.NodeList, error)
	// GetLogs returns logs for pod.
	GetLogs(ctx context.Context, pod, container string) (string, error)
	// GetEvents retrieves events from a pod by a name.
	GetEvents(ctx context.Context, name string) (string, error)
	// ApplyFile accepts manifest file contents, parses into []runtime.Object
	// and applies them against the cluster.
	ApplyFile(fileBytes []byte) error
	// DoCSVWait waits until for a CSV to be applied.
	DoCSVWait(ctx context.Context, key types.NamespacedName) error
	// GetSubscriptionCSV retrieves a subscription CSV.
	GetSubscriptionCSV(ctx context.Context, subKey types.NamespacedName) (types.NamespacedName, error)
	// DoRolloutWait waits until a deployment has been rolled out susccessfully or there is an error.
	DoRolloutWait(ctx context.Context, key types.NamespacedName) error
	// GetOperatorGroup retrieves an operator group details by namespace and name.
	GetOperatorGroup(ctx context.Context, namespace, name string) (*v1.OperatorGroup, error)
	// CreateOperatorGroup creates an operator group to be used as part of a subscription.
	CreateOperatorGroup(ctx context.Context, namespace, name string) (*v1.OperatorGroup, error)
	// CreateSubscriptionForCatalog creates an OLM subscription.
	CreateSubscriptionForCatalog(ctx context.Context, namespace, name, catalogNamespace, catalog, packageName, channel, startingCSV string, approval v1alpha1.Approval) (*v1alpha1.Subscription, error)
	// GetSubscription retrieves an OLM subscription by namespace and name.
	GetSubscription(ctx context.Context, namespace, name string) (*v1alpha1.Subscription, error)
	// ListSubscriptions all the subscriptions in the namespace.
	ListSubscriptions(ctx context.Context, namespace string) (*v1alpha1.SubscriptionList, error)
	// GetInstallPlan retrieves an OLM install plan by namespace and name.
	GetInstallPlan(ctx context.Context, namespace string, name string) (*v1alpha1.InstallPlan, error)
	// UpdateInstallPlan updates the existing install plan in the specified namespace.
	UpdateInstallPlan(ctx context.Context, namespace string, installPlan *v1alpha1.InstallPlan) (*v1alpha1.InstallPlan, error)
	// ListCRDs returns a list of CRDs.
	ListCRDs(ctx context.Context, labelSelector *metav1.LabelSelector) (*apiextv1.CustomResourceDefinitionList, error)
	// ListCRs returns a list of CRs.
	ListCRs(ctx context.Context, namespace string, gvr schema.GroupVersionResource, labelSelector *metav1.LabelSelector) (*unstructured.UnstructuredList, error)
	// GetClusterServiceVersion retrieve a CSV by namespaced name.
	GetClusterServiceVersion(ctx context.Context, key types.NamespacedName) (*v1alpha1.ClusterServiceVersion, error)
	// ListClusterServiceVersion list all CSVs for the given namespace.
	ListClusterServiceVersion(ctx context.Context, namespace string) (*v1alpha1.ClusterServiceVersionList, error)
	// DeleteFile accepts manifest file contents parses into []runtime.Object
	// and deletes them from the cluster.
	DeleteFile(fileBytes []byte) error
}
