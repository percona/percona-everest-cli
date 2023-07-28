// Code generated by mockery v1.0.0. DO NOT EDIT.

package client

import (
	context "context"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	v1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	apiv1alpha1 "github.com/percona/everest-operator/api/v1alpha1"
	mock "github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	version "k8s.io/apimachinery/pkg/version"
)

// MockKubeClientConnector is an autogenerated mock type for the KubeClientConnector type
type MockKubeClientConnector struct {
	mock.Mock
}

// ApplyFile provides a mock function with given fields: fileBytes
func (_m *MockKubeClientConnector) ApplyFile(fileBytes []byte) error {
	ret := _m.Called(fileBytes)

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte) error); ok {
		r0 = rf(fileBytes)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ApplyObject provides a mock function with given fields: obj
func (_m *MockKubeClientConnector) ApplyObject(obj runtime.Object) error {
	ret := _m.Called(obj)

	var r0 error
	if rf, ok := ret.Get(0).(func(runtime.Object) error); ok {
		r0 = rf(obj)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ClusterName provides a mock function with given fields:
func (_m *MockKubeClientConnector) ClusterName() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// CreateNamespace provides a mock function with given fields: name
func (_m *MockKubeClientConnector) CreateNamespace(name string) error {
	ret := _m.Called(name)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateOperatorGroup provides a mock function with given fields: ctx, namespace, name
func (_m *MockKubeClientConnector) CreateOperatorGroup(ctx context.Context, namespace string, name string) (*v1.OperatorGroup, error) {
	ret := _m.Called(ctx, namespace, name)

	var r0 *v1.OperatorGroup
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *v1.OperatorGroup); ok {
		r0 = rf(ctx, namespace, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.OperatorGroup)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, namespace, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateSubscriptionForCatalog provides a mock function with given fields: ctx, namespace, name, catalogNamespace, catalog, packageName, channel, startingCSV, approval
func (_m *MockKubeClientConnector) CreateSubscriptionForCatalog(ctx context.Context, namespace string, name string, catalogNamespace string, catalog string, packageName string, channel string, startingCSV string, approval v1alpha1.Approval) (*v1alpha1.Subscription, error) {
	ret := _m.Called(ctx, namespace, name, catalogNamespace, catalog, packageName, channel, startingCSV, approval)

	var r0 *v1alpha1.Subscription
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, string, string, string, string, v1alpha1.Approval) *v1alpha1.Subscription); ok {
		r0 = rf(ctx, namespace, name, catalogNamespace, catalog, packageName, channel, startingCSV, approval)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.Subscription)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, string, string, string, string, v1alpha1.Approval) error); ok {
		r1 = rf(ctx, namespace, name, catalogNamespace, catalog, packageName, channel, startingCSV, approval)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteAllMonitoringResources provides a mock function with given fields: ctx, namespace
func (_m *MockKubeClientConnector) DeleteAllMonitoringResources(ctx context.Context, namespace string) error {
	ret := _m.Called(ctx, namespace)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, namespace)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteFile provides a mock function with given fields: fileBytes
func (_m *MockKubeClientConnector) DeleteFile(fileBytes []byte) error {
	ret := _m.Called(fileBytes)

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte) error); ok {
		r0 = rf(fileBytes)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteObject provides a mock function with given fields: obj
func (_m *MockKubeClientConnector) DeleteObject(obj runtime.Object) error {
	ret := _m.Called(obj)

	var r0 error
	if rf, ok := ret.Get(0).(func(runtime.Object) error); ok {
		r0 = rf(obj)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoCSVWait provides a mock function with given fields: ctx, key
func (_m *MockKubeClientConnector) DoCSVWait(ctx context.Context, key types.NamespacedName) error {
	ret := _m.Called(ctx, key)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.NamespacedName) error); ok {
		r0 = rf(ctx, key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoPackageWait provides a mock function with given fields: ctx, name
func (_m *MockKubeClientConnector) DoPackageWait(ctx context.Context, name string) error {
	ret := _m.Called(ctx, name)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoRolloutWait provides a mock function with given fields: ctx, key
func (_m *MockKubeClientConnector) DoRolloutWait(ctx context.Context, key types.NamespacedName) error {
	ret := _m.Called(ctx, key)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.NamespacedName) error); ok {
		r0 = rf(ctx, key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GenerateKubeConfigWithToken provides a mock function with given fields: user, secret
func (_m *MockKubeClientConnector) GenerateKubeConfigWithToken(user string, secret *corev1.Secret) ([]byte, error) {
	ret := _m.Called(user, secret)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(string, *corev1.Secret) []byte); ok {
		r0 = rf(user, secret)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, *corev1.Secret) error); ok {
		r1 = rf(user, secret)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetClusterServiceVersion provides a mock function with given fields: ctx, key
func (_m *MockKubeClientConnector) GetClusterServiceVersion(ctx context.Context, key types.NamespacedName) (*v1alpha1.ClusterServiceVersion, error) {
	ret := _m.Called(ctx, key)

	var r0 *v1alpha1.ClusterServiceVersion
	if rf, ok := ret.Get(0).(func(context.Context, types.NamespacedName) *v1alpha1.ClusterServiceVersion); ok {
		r0 = rf(ctx, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.ClusterServiceVersion)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, types.NamespacedName) error); ok {
		r1 = rf(ctx, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDatabaseCluster provides a mock function with given fields: ctx, name
func (_m *MockKubeClientConnector) GetDatabaseCluster(ctx context.Context, name string) (*apiv1alpha1.DatabaseCluster, error) {
	ret := _m.Called(ctx, name)

	var r0 *apiv1alpha1.DatabaseCluster
	if rf, ok := ret.Get(0).(func(context.Context, string) *apiv1alpha1.DatabaseCluster); ok {
		r0 = rf(ctx, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*apiv1alpha1.DatabaseCluster)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDeployment provides a mock function with given fields: ctx, name, namespace
func (_m *MockKubeClientConnector) GetDeployment(ctx context.Context, name string, namespace string) (*appsv1.Deployment, error) {
	ret := _m.Called(ctx, name, namespace)

	var r0 *appsv1.Deployment
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *appsv1.Deployment); ok {
		r0 = rf(ctx, name, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.Deployment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, name, namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetEvents provides a mock function with given fields: ctx, name
func (_m *MockKubeClientConnector) GetEvents(ctx context.Context, name string) (string, error) {
	ret := _m.Called(ctx, name)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, string) string); ok {
		r0 = rf(ctx, name)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetInstallPlan provides a mock function with given fields: ctx, namespace, name
func (_m *MockKubeClientConnector) GetInstallPlan(ctx context.Context, namespace string, name string) (*v1alpha1.InstallPlan, error) {
	ret := _m.Called(ctx, namespace, name)

	var r0 *v1alpha1.InstallPlan
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *v1alpha1.InstallPlan); ok {
		r0 = rf(ctx, namespace, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.InstallPlan)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, namespace, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLogs provides a mock function with given fields: ctx, pod, container
func (_m *MockKubeClientConnector) GetLogs(ctx context.Context, pod string, container string) (string, error) {
	ret := _m.Called(ctx, pod, container)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, string, string) string); ok {
		r0 = rf(ctx, pod, container)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, pod, container)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNodes provides a mock function with given fields: ctx
func (_m *MockKubeClientConnector) GetNodes(ctx context.Context) (*corev1.NodeList, error) {
	ret := _m.Called(ctx)

	var r0 *corev1.NodeList
	if rf, ok := ret.Get(0).(func(context.Context) *corev1.NodeList); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.NodeList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetOperatorGroup provides a mock function with given fields: ctx, namespace, name
func (_m *MockKubeClientConnector) GetOperatorGroup(ctx context.Context, namespace string, name string) (*v1.OperatorGroup, error) {
	ret := _m.Called(ctx, namespace, name)

	var r0 *v1.OperatorGroup
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *v1.OperatorGroup); ok {
		r0 = rf(ctx, namespace, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.OperatorGroup)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, namespace, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPackageManifest provides a mock function with given fields: ctx, name
func (_m *MockKubeClientConnector) GetPackageManifest(ctx context.Context, name string) (*operatorsv1.PackageManifest, error) {
	ret := _m.Called(ctx, name)

	var r0 *operatorsv1.PackageManifest
	if rf, ok := ret.Get(0).(func(context.Context, string) *operatorsv1.PackageManifest); ok {
		r0 = rf(ctx, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*operatorsv1.PackageManifest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPersistentVolumes provides a mock function with given fields: ctx
func (_m *MockKubeClientConnector) GetPersistentVolumes(ctx context.Context) (*corev1.PersistentVolumeList, error) {
	ret := _m.Called(ctx)

	var r0 *corev1.PersistentVolumeList
	if rf, ok := ret.Get(0).(func(context.Context) *corev1.PersistentVolumeList); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.PersistentVolumeList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPods provides a mock function with given fields: ctx, namespace, labelSelector
func (_m *MockKubeClientConnector) GetPods(ctx context.Context, namespace string, labelSelector *metav1.LabelSelector) (*corev1.PodList, error) {
	ret := _m.Called(ctx, namespace, labelSelector)

	var r0 *corev1.PodList
	if rf, ok := ret.Get(0).(func(context.Context, string, *metav1.LabelSelector) *corev1.PodList); ok {
		r0 = rf(ctx, namespace, labelSelector)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.PodList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *metav1.LabelSelector) error); ok {
		r1 = rf(ctx, namespace, labelSelector)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSecret provides a mock function with given fields: ctx, name, namespace
func (_m *MockKubeClientConnector) GetSecret(ctx context.Context, name string, namespace string) (*corev1.Secret, error) {
	ret := _m.Called(ctx, name, namespace)

	var r0 *corev1.Secret
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *corev1.Secret); ok {
		r0 = rf(ctx, name, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Secret)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, name, namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSecretsForServiceAccount provides a mock function with given fields: ctx, accountName
func (_m *MockKubeClientConnector) GetSecretsForServiceAccount(ctx context.Context, accountName string) (*corev1.Secret, error) {
	ret := _m.Called(ctx, accountName)

	var r0 *corev1.Secret
	if rf, ok := ret.Get(0).(func(context.Context, string) *corev1.Secret); ok {
		r0 = rf(ctx, accountName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Secret)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, accountName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetServerVersion provides a mock function with given fields:
func (_m *MockKubeClientConnector) GetServerVersion() (*version.Info, error) {
	ret := _m.Called()

	var r0 *version.Info
	if rf, ok := ret.Get(0).(func() *version.Info); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*version.Info)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStorageClasses provides a mock function with given fields: ctx
func (_m *MockKubeClientConnector) GetStorageClasses(ctx context.Context) (*storagev1.StorageClassList, error) {
	ret := _m.Called(ctx)

	var r0 *storagev1.StorageClassList
	if rf, ok := ret.Get(0).(func(context.Context) *storagev1.StorageClassList); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*storagev1.StorageClassList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSubscription provides a mock function with given fields: ctx, namespace, name
func (_m *MockKubeClientConnector) GetSubscription(ctx context.Context, namespace string, name string) (*v1alpha1.Subscription, error) {
	ret := _m.Called(ctx, namespace, name)

	var r0 *v1alpha1.Subscription
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *v1alpha1.Subscription); ok {
		r0 = rf(ctx, namespace, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.Subscription)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, namespace, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSubscriptionCSV provides a mock function with given fields: ctx, subKey
func (_m *MockKubeClientConnector) GetSubscriptionCSV(ctx context.Context, subKey types.NamespacedName) (types.NamespacedName, error) {
	ret := _m.Called(ctx, subKey)

	var r0 types.NamespacedName
	if rf, ok := ret.Get(0).(func(context.Context, types.NamespacedName) types.NamespacedName); ok {
		r0 = rf(ctx, subKey)
	} else {
		r0 = ret.Get(0).(types.NamespacedName)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, types.NamespacedName) error); ok {
		r1 = rf(ctx, subKey)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListCRDs provides a mock function with given fields: ctx, labelSelector
func (_m *MockKubeClientConnector) ListCRDs(ctx context.Context, labelSelector *metav1.LabelSelector) (*apiextensionsv1.CustomResourceDefinitionList, error) {
	ret := _m.Called(ctx, labelSelector)

	var r0 *apiextensionsv1.CustomResourceDefinitionList
	if rf, ok := ret.Get(0).(func(context.Context, *metav1.LabelSelector) *apiextensionsv1.CustomResourceDefinitionList); ok {
		r0 = rf(ctx, labelSelector)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*apiextensionsv1.CustomResourceDefinitionList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *metav1.LabelSelector) error); ok {
		r1 = rf(ctx, labelSelector)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListCRs provides a mock function with given fields: ctx, namespace, gvr, labelSelector
func (_m *MockKubeClientConnector) ListCRs(ctx context.Context, namespace string, gvr schema.GroupVersionResource, labelSelector *metav1.LabelSelector) (*unstructured.UnstructuredList, error) {
	ret := _m.Called(ctx, namespace, gvr, labelSelector)

	var r0 *unstructured.UnstructuredList
	if rf, ok := ret.Get(0).(func(context.Context, string, schema.GroupVersionResource, *metav1.LabelSelector) *unstructured.UnstructuredList); ok {
		r0 = rf(ctx, namespace, gvr, labelSelector)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*unstructured.UnstructuredList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, schema.GroupVersionResource, *metav1.LabelSelector) error); ok {
		r1 = rf(ctx, namespace, gvr, labelSelector)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListClusterServiceVersion provides a mock function with given fields: ctx, namespace
func (_m *MockKubeClientConnector) ListClusterServiceVersion(ctx context.Context, namespace string) (*v1alpha1.ClusterServiceVersionList, error) {
	ret := _m.Called(ctx, namespace)

	var r0 *v1alpha1.ClusterServiceVersionList
	if rf, ok := ret.Get(0).(func(context.Context, string) *v1alpha1.ClusterServiceVersionList); ok {
		r0 = rf(ctx, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.ClusterServiceVersionList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListDatabaseClusters provides a mock function with given fields: ctx
func (_m *MockKubeClientConnector) ListDatabaseClusters(ctx context.Context) (*apiv1alpha1.DatabaseClusterList, error) {
	ret := _m.Called(ctx)

	var r0 *apiv1alpha1.DatabaseClusterList
	if rf, ok := ret.Get(0).(func(context.Context) *apiv1alpha1.DatabaseClusterList); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*apiv1alpha1.DatabaseClusterList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListSecrets provides a mock function with given fields: ctx
func (_m *MockKubeClientConnector) ListSecrets(ctx context.Context) (*corev1.SecretList, error) {
	ret := _m.Called(ctx)

	var r0 *corev1.SecretList
	if rf, ok := ret.Get(0).(func(context.Context) *corev1.SecretList); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.SecretList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListSubscriptions provides a mock function with given fields: ctx, namespace
func (_m *MockKubeClientConnector) ListSubscriptions(ctx context.Context, namespace string) (*v1alpha1.SubscriptionList, error) {
	ret := _m.Called(ctx, namespace)

	var r0 *v1alpha1.SubscriptionList
	if rf, ok := ret.Get(0).(func(context.Context, string) *v1alpha1.SubscriptionList); ok {
		r0 = rf(ctx, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.SubscriptionList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateInstallPlan provides a mock function with given fields: ctx, namespace, installPlan
func (_m *MockKubeClientConnector) UpdateInstallPlan(ctx context.Context, namespace string, installPlan *v1alpha1.InstallPlan) (*v1alpha1.InstallPlan, error) {
	ret := _m.Called(ctx, namespace, installPlan)

	var r0 *v1alpha1.InstallPlan
	if rf, ok := ret.Get(0).(func(context.Context, string, *v1alpha1.InstallPlan) *v1alpha1.InstallPlan); ok {
		r0 = rf(ctx, namespace, installPlan)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.InstallPlan)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *v1alpha1.InstallPlan) error); ok {
		r1 = rf(ctx, namespace, installPlan)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
