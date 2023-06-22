// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package kubernetes provides functionality for kubernetes.
package kubernetes

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	dbaasv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/version"

	"github.com/percona/percona-everest-cli/data"
	"github.com/percona/percona-everest-cli/pkg/kubernetes/client"
)

// ClusterType defines type of cluster.
type ClusterType string

const (
	// ClusterTypeUnknown is for unknown type.
	ClusterTypeUnknown ClusterType = "unknown"
	// ClusterTypeMinikube is for minikube.
	ClusterTypeMinikube ClusterType = "minikube"
	// ClusterTypeEKS is for EKS.
	ClusterTypeEKS ClusterType = "eks"
	// ClusterTypeGeneric is a generic type.
	ClusterTypeGeneric ClusterType = "generic"

	pxcDeploymentName          = "percona-xtradb-cluster-operator"
	psmdbDeploymentName        = "percona-server-mongodb-operator"
	dbaasDeploymentName        = "dbaas-operator-controller-manager"
	psmdbOperatorContainerName = "percona-server-mongodb-operator"
	pxcOperatorContainerName   = "percona-xtradb-cluster-operator"
	dbaasOperatorContainerName = "manager"
	databaseClusterKind        = "DatabaseCluster"
	databaseClusterAPIVersion  = "dbaas.percona.com/v1"
	restartAnnotationKey       = "dbaas.percona.com/restart"
	managedByKey               = "dbaas.percona.com/managed-by"

	// ContainerStateWaiting represents a state when container requires some
	// operations being done in order to complete start up.
	ContainerStateWaiting ContainerState = "waiting"
	// ContainerStateTerminated indicates that container began execution and
	// then either ran to completion or failed for some reason.
	ContainerStateTerminated ContainerState = "terminated"

	olmNamespace = "olm"

	// APIVersionCoreosV1 constant for some API requests.
	APIVersionCoreosV1 = "operators.coreos.com/v1"

	pollInterval = 1 * time.Second
	pollDuration = 150 * time.Second
)

// ErrEmptyVersionTag Got an empty version tag from GitHub API.
var ErrEmptyVersionTag error = errors.New("got an empty version tag from Github")

// Kubernetes is a client for Kubernetes.
type Kubernetes struct {
	client     client.KubeClientConnector
	l          *logrus.Entry
	httpClient *http.Client
	kubeconfig string
}

// ContainerState describes container's state - waiting, running, terminated.
type ContainerState string

// NodeSummaryNode holds information about Node inside Node's summary.
type NodeSummaryNode struct {
	FileSystem NodeFileSystemSummary `json:"fs,omitempty"`
}

// NodeSummary holds summary of the Node.
// One gets this by requesting Kubernetes API endpoint:
// /v1/nodes/<node-name>/proxy/stats/summary.
type NodeSummary struct {
	Node NodeSummaryNode `json:"node,omitempty"`
}

// NodeFileSystemSummary holds a summary of Node's filesystem.
type NodeFileSystemSummary struct {
	UsedBytes uint64 `json:"usedBytes,omitempty"`
}

// New returns new Kubernetes object.
func New(kubeconfig string, l *logrus.Entry) (*Kubernetes, error) {
	client, err := client.NewFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		client: client,
		l:      l.WithField("component", "kubernetes"),
		httpClient: &http.Client{
			Timeout: time.Second * 5,
			Transport: &http.Transport{
				MaxIdleConns:    1,
				IdleConnTimeout: 10 * time.Second,
			},
		},
		kubeconfig: kubeconfig,
	}, nil
}

// NewEmpty returns new Kubernetes object.
func NewEmpty() *Kubernetes {
	return &Kubernetes{
		client: &client.Client{},
		l:      logrus.WithField("component", "kubernetes"),
		httpClient: &http.Client{
			Timeout: time.Second * 5,
			Transport: &http.Transport{
				MaxIdleConns:    1,
				IdleConnTimeout: 10 * time.Second,
			},
		},
	}
}

// ClusterName returns the name of the k8s cluster.
func (k *Kubernetes) ClusterName() string {
	return k.client.ClusterName()
}

// ListDatabaseClusters returns list of managed PCX clusters.
func (k *Kubernetes) ListDatabaseClusters(ctx context.Context) (*dbaasv1.DatabaseClusterList, error) {
	return k.client.ListDatabaseClusters(ctx)
}

// GetDatabaseCluster returns PXC clusters by provided name.
func (k *Kubernetes) GetDatabaseCluster(ctx context.Context, name string) (*dbaasv1.DatabaseCluster, error) {
	return k.client.GetDatabaseCluster(ctx, name)
}

// RestartDatabaseCluster restarts database cluster.
func (k *Kubernetes) RestartDatabaseCluster(ctx context.Context, name string) error {
	cluster, err := k.client.GetDatabaseCluster(ctx, name)
	if err != nil {
		return err
	}
	cluster.TypeMeta.APIVersion = databaseClusterAPIVersion
	cluster.TypeMeta.Kind = databaseClusterKind
	if cluster.ObjectMeta.Annotations == nil {
		cluster.ObjectMeta.Annotations = make(map[string]string)
	}
	cluster.ObjectMeta.Annotations[restartAnnotationKey] = "true"
	return k.client.ApplyObject(cluster)
}

// PatchDatabaseCluster patches CR of managed Database cluster.
func (k *Kubernetes) PatchDatabaseCluster(cluster *dbaasv1.DatabaseCluster) error {
	return k.client.ApplyObject(cluster)
}

// CreateDatabaseCluster creates database cluster.
func (k *Kubernetes) CreateDatabaseCluster(cluster *dbaasv1.DatabaseCluster) error {
	if cluster.ObjectMeta.Annotations == nil {
		cluster.ObjectMeta.Annotations = make(map[string]string)
	}
	cluster.ObjectMeta.Annotations[managedByKey] = "pmm"
	return k.client.ApplyObject(cluster)
}

// DeleteDatabaseCluster deletes database cluster.
func (k *Kubernetes) DeleteDatabaseCluster(ctx context.Context, name string) error {
	cluster, err := k.client.GetDatabaseCluster(ctx, name)
	if err != nil {
		return err
	}
	cluster.TypeMeta.APIVersion = databaseClusterAPIVersion
	cluster.TypeMeta.Kind = databaseClusterKind
	return k.client.DeleteObject(cluster)
}

// GetDefaultStorageClassName returns first storageClassName from kubernetes cluster.
func (k *Kubernetes) GetDefaultStorageClassName(ctx context.Context) (string, error) {
	storageClasses, err := k.client.GetStorageClasses(ctx)
	if err != nil {
		return "", err
	}
	if len(storageClasses.Items) != 0 {
		return storageClasses.Items[0].Name, nil
	}
	return "", errors.New("no storage classes available")
}

// GetClusterType tries to guess the underlying kubernetes cluster based on storage class.
func (k *Kubernetes) GetClusterType(ctx context.Context) (ClusterType, error) {
	storageClasses, err := k.client.GetStorageClasses(ctx)
	if err != nil {
		return ClusterTypeUnknown, err
	}
	for _, storageClass := range storageClasses.Items {
		if strings.Contains(storageClass.Provisioner, "aws") {
			return ClusterTypeEKS, nil
		}
		if strings.Contains(storageClass.Provisioner, "minikube") ||
			strings.Contains(storageClass.Provisioner, "kubevirt.io/hostpath-provisioner") ||
			strings.Contains(storageClass.Provisioner, "standard") {
			return ClusterTypeMinikube, nil
		}
	}
	return ClusterTypeGeneric, nil
}

// getOperatorVersion parses operator version from operator deployment.
func (k *Kubernetes) getOperatorVersion(ctx context.Context, deploymentName, containerName string) (string, error) {
	deployment, err := k.client.GetDeployment(ctx, deploymentName, "")
	if err != nil {
		return "", err
	}
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			return strings.Split(container.Image, ":")[1], nil
		}
	}
	return "", errors.New("unknown version of operator")
}

// GetPSMDBOperatorVersion parses PSMDB operator version from operator deployment.
func (k *Kubernetes) GetPSMDBOperatorVersion(ctx context.Context) (string, error) {
	return k.getOperatorVersion(ctx, psmdbDeploymentName, psmdbOperatorContainerName)
}

// GetPXCOperatorVersion parses PXC operator version from operator deployment.
func (k *Kubernetes) GetPXCOperatorVersion(ctx context.Context) (string, error) {
	return k.getOperatorVersion(ctx, pxcDeploymentName, pxcOperatorContainerName)
}

// GetDBaaSOperatorVersion parses DBaaS operator version from operator deployment.
func (k *Kubernetes) GetDBaaSOperatorVersion(ctx context.Context) (string, error) {
	return k.getOperatorVersion(ctx, dbaasDeploymentName, dbaasOperatorContainerName)
}

// GetSecret returns secret by name.
func (k *Kubernetes) GetSecret(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	return k.client.GetSecret(ctx, name, namespace)
}

// ListSecrets returns secret by name.
func (k *Kubernetes) ListSecrets(ctx context.Context) (*corev1.SecretList, error) {
	return k.client.ListSecrets(ctx)
}

// CreatePMMSecret creates pmm secret in kubernetes.
func (k *Kubernetes) CreatePMMSecret(secretName string, secrets map[string][]byte) error {
	secret := &corev1.Secret{ //nolint: exhaustruct
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: secrets,
	}
	return k.client.ApplyObject(secret)
}

// CreateRestore creates a restore.
func (k *Kubernetes) CreateRestore(restore *dbaasv1.DatabaseClusterRestore) error {
	return k.client.ApplyObject(restore)
}

// GetPods returns list of pods.
func (k *Kubernetes) GetPods(
	ctx context.Context,
	namespace string,
	labelSelector *metav1.LabelSelector,
) (*corev1.PodList, error) {
	return k.client.GetPods(ctx, namespace, labelSelector)
}

// GetLogs returns logs as slice of log lines - strings - for given pod's container.
func (k *Kubernetes) GetLogs(
	ctx context.Context,
	containerStatuses []corev1.ContainerStatus,
	pod,
	container string,
) ([]string, error) {
	if IsContainerInState(containerStatuses, ContainerStateWaiting) {
		return []string{}, nil
	}

	stdout, err := k.client.GetLogs(ctx, pod, container)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get logs")
	}

	if stdout == "" {
		return []string{}, nil
	}

	return strings.Split(stdout, "\n"), nil
}

// GetEvents returns pod's events as a slice of strings.
func (k *Kubernetes) GetEvents(ctx context.Context, pod string) ([]string, error) {
	stdout, err := k.client.GetEvents(ctx, pod)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't describe pod")
	}

	lines := strings.Split(stdout, "\n")

	return lines, nil
}

// IsContainerInState returns true if container is in give state, otherwise false.
func IsContainerInState(containerStatuses []corev1.ContainerStatus, state ContainerState) bool {
	containerState := make(map[string]interface{})
	for _, status := range containerStatuses {
		data, err := json.Marshal(status.State)
		if err != nil {
			return false
		}

		if err := json.Unmarshal(data, &containerState); err != nil {
			return false
		}

		if _, ok := containerState[string(state)]; ok {
			return true
		}
	}

	return false
}

// IsNodeInCondition returns true if node's condition given as an argument has
// status "True". Otherwise it returns false.
func IsNodeInCondition(node corev1.Node, conditionType corev1.NodeConditionType) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Status == corev1.ConditionTrue && condition.Type == conditionType {
			return true
		}
	}
	return false
}

// GetWorkerNodes returns list of cluster workers nodes.
func (k *Kubernetes) GetWorkerNodes(ctx context.Context) ([]corev1.Node, error) {
	nodes, err := k.client.GetNodes(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get nodes of Kubernetes cluster")
	}
	forbidenTaints := map[string]corev1.TaintEffect{
		"node.cloudprovider.kubernetes.io/uninitialized": corev1.TaintEffectNoSchedule,
		"node.kubernetes.io/unschedulable":               corev1.TaintEffectNoSchedule,
		"node-role.kubernetes.io/master":                 corev1.TaintEffectNoSchedule,
	}
	workers := make([]corev1.Node, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		if len(node.Spec.Taints) == 0 {
			workers = append(workers, node)
			continue
		}
		for _, taint := range node.Spec.Taints {
			effect, keyFound := forbidenTaints[taint.Key]
			if !keyFound || effect != taint.Effect {
				workers = append(workers, node)
			}
		}
	}
	return workers, nil
}

// GetPersistentVolumes returns list of persistent volumes.
func (k *Kubernetes) GetPersistentVolumes(ctx context.Context) (*corev1.PersistentVolumeList, error) {
	return k.client.GetPersistentVolumes(ctx)
}

// GetStorageClasses returns all storage classes available in the cluster.
func (k *Kubernetes) GetStorageClasses(ctx context.Context) (*storagev1.StorageClassList, error) {
	return k.client.GetStorageClasses(ctx)
}

// InstallOLMOperator installs the OLM in the Kubernetes cluster.
func (k *Kubernetes) InstallOLMOperator(ctx context.Context) error {
	deployment, err := k.client.GetDeployment(ctx, "olm-operator", "olm")
	if err == nil && deployment != nil && deployment.ObjectMeta.Name != "" {
		k.l.Info("OLM operator is already installed")
		return nil // already installed
	}

	resources, err := k.applyResources()
	if err != nil {
		return err
	}

	if err := k.waitForDeploymentRollout(ctx); err != nil {
		return err
	}

	if err := k.applyCSVs(ctx, resources); err != nil {
		return err
	}

	if err := k.client.DoRolloutWait(ctx, types.NamespacedName{Namespace: "olm", Name: "packageserver"}); err != nil {
		return errors.Wrap(err, "error while waiting for deployment rollout")
	}

	return nil
}

func (k *Kubernetes) applyCSVs(ctx context.Context, resources []unstructured.Unstructured) error {
	subscriptions := filterResources(resources, func(r unstructured.Unstructured) bool {
		return r.GroupVersionKind() == schema.GroupVersionKind{
			Group:   v1alpha1.GroupName,
			Version: v1alpha1.GroupVersion,
			Kind:    v1alpha1.SubscriptionKind,
		}
	})

	for _, sub := range subscriptions {
		subscriptionKey := types.NamespacedName{Namespace: sub.GetNamespace(), Name: sub.GetName()}
		log.Printf("Waiting for subscription/%s to install CSV", subscriptionKey.Name)
		csvKey, err := k.client.GetSubscriptionCSV(ctx, subscriptionKey)
		if err != nil {
			return errors.Errorf("subscription/%s failed to install CSV: %v", subscriptionKey.Name, err)
		}
		log.Printf("Waiting for clusterserviceversion/%s to reach 'Succeeded' phase", csvKey.Name)
		if err := k.client.DoCSVWait(ctx, csvKey); err != nil {
			return errors.Errorf("clusterserviceversion/%s failed to reach 'Succeeded' phase", csvKey.Name)
		}
	}

	return nil
}

// InstallPerconaCatalog installs percona catalog and ensures that packages are available
func (k *Kubernetes) InstallPerconaCatalog(ctx context.Context) error {
	data, err := fs.ReadFile(data.OLMCRDs, "crds/olm/percona-dbaas-catalog.yaml")
	if err != nil {
		return errors.Wrapf(err, "failed to read percona catalog file")
	}

	if err := k.client.ApplyFile(data); err != nil {
		return errors.Wrapf(err, "cannot apply percona catalog file")
	}
	if err := k.client.DoPackageWait(ctx, "dbaas-operator"); err != nil {
		return errors.Wrapf(err, "timeout waiting for package")
	}
	return nil
}

func (k *Kubernetes) waitForPackageService() error {
	return nil
}

func (k *Kubernetes) applyResources() ([]unstructured.Unstructured, error) {
	files := []string{
		"crds/olm/crds.yaml",
		"crds/olm/olm.yaml",
	}

	resources := []unstructured.Unstructured{}
	for _, f := range files {
		data, err := fs.ReadFile(data.OLMCRDs, f)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read %q file", f)
		}

		if err := k.client.ApplyFile(data); err != nil {
			return nil, errors.Wrapf(err, "cannot apply %q file", f)
		}

		r, err := decodeResources(data)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot decode resources in %q", f)
		}
		resources = append(resources, r...)
	}

	return resources, nil
}

func (k *Kubernetes) waitForDeploymentRollout(ctx context.Context) error {
	if err := k.client.DoRolloutWait(ctx, types.NamespacedName{
		Namespace: olmNamespace,
		Name:      "olm-operator",
	}); err != nil {
		return errors.Wrap(err, "error while waiting for deployment rollout")
	}
	if err := k.client.DoRolloutWait(ctx, types.NamespacedName{Namespace: "olm", Name: "catalog-operator"}); err != nil {
		return errors.Wrap(err, "error while waiting for deployment rollout")
	}

	return nil
}

func decodeResources(f []byte) ([]unstructured.Unstructured, error) {
	var err error
	objs := []unstructured.Unstructured{}
	dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(f), 8)
	for {
		var u unstructured.Unstructured
		err = dec.Decode(&u)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
		objs = append(objs, u)
	}

	return objs, nil
}

func filterResources(resources []unstructured.Unstructured, filter func(unstructured.
	Unstructured) bool,
) []unstructured.Unstructured {
	filtered := make([]unstructured.Unstructured, 0, len(resources))
	for _, r := range resources {
		if filter(r) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// CreateNamespace creates a new namespace.
func (k *Kubernetes) CreateNamespace(name string) error {
	return k.client.CreateNamespace(name)
}

// InstallOperatorRequest holds the fields to make an operator install request.
type InstallOperatorRequest struct {
	Namespace              string
	Name                   string
	OperatorGroup          string
	CatalogSource          string
	CatalogSourceNamespace string
	Channel                string
	InstallPlanApproval    v1alpha1.Approval
	StartingCSV            string
}

// InstallOperator installs an operator via OLM.
func (k *Kubernetes) InstallOperator(ctx context.Context, req InstallOperatorRequest) error {
	if err := createOperatorGroupIfNeeded(ctx, k.client, req.OperatorGroup, req.Namespace); err != nil {
		return err
	}

	subs, err := k.client.CreateSubscriptionForCatalog(
		ctx, req.Namespace, req.Name, "olm", req.CatalogSource,
		req.Name, req.Channel, req.StartingCSV, v1alpha1.ApprovalManual,
	)
	if err != nil {
		return errors.Wrap(err, "cannot create a subscription to install the operator")
	}

	err = wait.PollUntilContextTimeout(ctx, pollInterval, pollDuration, false, func(ctx context.Context) (bool, error) {
		k.l.Debugf("Polling subscription %s/%s", req.Namespace, req.Name)
		subs, err = k.client.GetSubscription(ctx, req.Namespace, req.Name)
		if err != nil || subs == nil || (subs != nil && subs.Status.InstallPlanRef == nil) {
			return false, errors.Wrapf(err, "cannot get an install plan for the operator subscription: %q", req.Name)
		}

		return k.approveInstallPlan(ctx, req.Namespace, subs.Status.InstallPlanRef.Name)
	})

	return err
}

func (k *Kubernetes) approveInstallPlan(ctx context.Context, namespace, installPlanName string) (bool, error) {
	ip, err := k.client.GetInstallPlan(ctx, namespace, installPlanName)
	if err != nil {
		return false, err
	}

	ip.Spec.Approved = true
	_, err = k.client.UpdateInstallPlan(ctx, namespace, ip)
	if err != nil {
		var sErr *apierrors.StatusError
		if ok := errors.As(err, sErr); ok && sErr.Status().Reason == metav1.StatusReasonConflict {
			// The install plan has changed. We retry to get an updated install plan.
			k.l.Debugf("Retrying install plan update due to a version conflict. Error: %s", err)
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func createOperatorGroupIfNeeded(
	ctx context.Context,
	client client.KubeClientConnector,
	name, namespace string,
) error {
	_, err := client.GetOperatorGroup(ctx, namespace, name)
	if err == nil {
		return nil
	}

	_, err = client.CreateOperatorGroup(ctx, namespace, name)

	return err
}

// ListSubscriptions all the subscriptions in the namespace.
func (k *Kubernetes) ListSubscriptions(ctx context.Context, namespace string) (*v1alpha1.SubscriptionList, error) {
	return k.client.ListSubscriptions(ctx, namespace)
}

// UpgradeOperator upgrades an operator to the next available version.
func (k *Kubernetes) UpgradeOperator(ctx context.Context, namespace, name string) error {
	ip, err := k.getInstallPlan(ctx, namespace, name)
	if err != nil {
		return err
	}

	if ip.Spec.Approved {
		return nil // There are no upgrades.
	}

	ip.Spec.Approved = true

	_, err = k.client.UpdateInstallPlan(ctx, namespace, ip)

	return err
}

func (k *Kubernetes) getInstallPlan(ctx context.Context, namespace, name string) (*v1alpha1.InstallPlan, error) {
	var subs *v1alpha1.Subscription

	// If the subscription was recently created, the install plan might not be ready yet.
	err := wait.PollUntilContextTimeout(ctx, pollInterval, pollDuration, false, func(ctx context.Context) (bool, error) {
		var err error
		subs, err = k.client.GetSubscription(ctx, namespace, name)
		if err != nil {
			return false, err
		}
		if subs == nil || subs.Status.Install == nil || subs.Status.Install.Name == "" {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	}
	if subs == nil || subs.Status.Install == nil || subs.Status.Install.Name == "" {
		return nil, errors.Errorf("cannot get subscription for %q operator", name)
	}

	ip, err := k.client.GetInstallPlan(ctx, namespace, subs.Status.Install.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get install plan to upgrade %q", name)
	}

	return ip, err
}

// GetServerVersion returns server version.
func (k *Kubernetes) GetServerVersion() (*version.Info, error) {
	return k.client.GetServerVersion()
}

// GetClusterServiceVersion retrieves a ClusterServiceVersion by namespaced name.
func (k *Kubernetes) GetClusterServiceVersion(
	ctx context.Context,
	key types.NamespacedName,
) (*v1alpha1.ClusterServiceVersion, error) {
	return k.client.GetClusterServiceVersion(ctx, key)
}

// ListClusterServiceVersion list all CSVs for the given namespace.
func (k *Kubernetes) ListClusterServiceVersion(
	ctx context.Context,
	namespace string,
) (*v1alpha1.ClusterServiceVersionList, error) {
	return k.client.ListClusterServiceVersion(ctx, namespace)
}

// DeleteObject deletes an object.
func (k *Kubernetes) DeleteObject(obj runtime.Object) error {
	return k.client.DeleteObject(obj)
}

// ProvisionMonitoring provisions PMM monitoring and creates a VM Agent instance.
func (k *Kubernetes) ProvisionMonitoring(login, password, pmmPublicAddress string) error {
	randomCrypto, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return err
	}

	secretName := fmt.Sprintf("vm-operator-%d", randomCrypto)
	err = k.CreatePMMSecret(secretName, map[string][]byte{
		"username": []byte(login),
		"password": []byte(password),
	})
	if err != nil {
		return errors.Wrap(err, "cannot create PMM secret")
	}

	vmagent := vmAgentSpec(secretName, pmmPublicAddress)
	err = k.client.ApplyObject(vmagent)
	if err != nil {
		return errors.Wrap(err, "cannot apply vm agent spec")
	}

	files := []string{
		"crds/victoriametrics/crs/vmagent_rbac.yaml",
		"crds/victoriametrics/crs/vmnodescrape.yaml",
		"crds/victoriametrics/crs/vmpodscrape.yaml",
		"crds/victoriametrics/kube-state-metrics/service-account.yaml",
		"crds/victoriametrics/kube-state-metrics/cluster-role.yaml",
		"crds/victoriametrics/kube-state-metrics/cluster-role-binding.yaml",
		"crds/victoriametrics/kube-state-metrics/deployment.yaml",
		"crds/victoriametrics/kube-state-metrics/service.yaml",
		"crds/victoriametrics/kube-state-metrics.yaml",
	}
	for _, path := range files {
		file, err := data.OLMCRDs.ReadFile(path)
		if err != nil {
			return err
		}
		// retry 3 times because applying vmagent spec might take some time.
		for i := 0; i < 3; i++ {
			k.l.Debugf("Applying file %s", path)
			err = k.client.ApplyFile(file)
			if err != nil {
				k.l.Debugf("%s: retrying after error: %s", path, err)
				time.Sleep(10 * time.Second)
				continue
			}
			break
		}
		if err != nil {
			return errors.Wrapf(err, "cannot apply file: %q", path)
		}
	}
	return nil
}

// CleanupMonitoring remove all files installed by ProvisionMonitoring.
func (k *Kubernetes) CleanupMonitoring() error {
	files := []string{
		"crds/victoriametrics/kube-state-metrics.yaml",
		"crds/victoriametrics/kube-state-metrics/cluster-role-binding.yaml",
		"crds/victoriametrics/kube-state-metrics/cluster-role.yaml",
		"crds/victoriametrics/kube-state-metrics/deployment.yaml",
		"crds/victoriametrics/kube-state-metrics/service-account.yaml",
		"crds/victoriametrics/kube-state-metrics/service.yaml",
		"crds/victoriametrics/crs/vmagent_rbac.yaml",
		"crds/victoriametrics/crs/vmnodescrape.yaml",
		"crds/victoriametrics/crs/vmpodscrape.yaml",
	}
	for _, path := range files {
		file, err := data.OLMCRDs.ReadFile(path)
		if err != nil {
			return err
		}
		err = k.client.DeleteFile(file)
		if err != nil {
			return errors.Wrapf(err, "cannot apply file: %q", path)
		}
	}

	return nil
}

const specVMAgent = `
{
	"kind": "VMAgent",
	"apiVersion": "operator.victoriametrics.com/v1beta1",
	"metadata": {
		"name": "pmm-vmagent-%[1]s",
		"creationTimestamp": null
	},
	"spec": {
		"image": {},
		"replicaCount": 1,
		"resources": {
			"limits": {
				"cpu": "500m",
				"memory": "850Mi"
			},
			"requests": {
				"cpu": "250m",
				"memory": "350Mi"
			}
		},
		"remoteWrite": [
			{
				"url": "%[2]s/victoriametrics/api/v1/write",
				"basicAuth": {
					"username": {
						"name": "%[1]s",
						"key": "username"
					},
					"password": {
						"name": "%[1]s",
						"key": "password"
					}
				},
				"tlsConfig": {
					"ca": {},
					"cert": {},
					"insecureSkipVerify": true
				}
			}
		],
		"selectAllByDefault": true,
		"serviceScrapeSelector": {},
		"serviceScrapeNamespaceSelector": {},
		"podScrapeSelector": {},
		"podScrapeNamespaceSelector": {},
		"probeSelector": {},
		"probeNamespaceSelector": {},
		"staticScrapeSelector": {},
		"staticScrapeNamespaceSelector": {},
		"arbitraryFSAccessThroughSMs": {},
		"extraArgs": {
			"memory.allowedPercent": "40"
		}
	},
	"status": {
		"shards": 0,
		"selector": "",
		"replicas": 0,
		"updatedReplicas": 0,
		"availableReplicas": 0,
		"unavailableReplicas": 0
	}
}`

func vmAgentSpec(secretName, address string) runtime.Object { //nolint:ireturn
	manifest := fmt.Sprintf(specVMAgent, secretName, address)

	o, _, err := unstructured.UnstructuredJSONScheme.Decode([]byte(manifest), nil, nil)
	if err != nil {
		logrus.Panic(err)
	}

	return o
}
