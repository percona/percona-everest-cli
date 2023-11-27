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

// Package kubernetes provides functionality for kubernetes.
package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	everestv1alpha1 "github.com/percona/everest-operator/api/v1alpha1"
	"go.uber.org/zap"
	yamlv3 "gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
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
	"k8s.io/client-go/rest"

	"github.com/percona/percona-everest-cli/data"
	"github.com/percona/percona-everest-cli/pkg/kubernetes/client"
	everestVersion "github.com/percona/percona-everest-cli/pkg/version"
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

	// PerconaEverestDeploymentName stores the name of everest backend deployment.
	PerconaEverestDeploymentName = "percona-everest"

	pxcDeploymentName            = "percona-xtradb-cluster-operator"
	psmdbDeploymentName          = "percona-server-mongodb-operator"
	postgresDeploymentName       = "percona-postgresql-operator"
	everestDeploymentName        = "everest-operator-controller-manager"
	psmdbOperatorContainerName   = "percona-server-mongodb-operator"
	pxcOperatorContainerName     = "percona-xtradb-cluster-operator"
	everestOperatorContainerName = "manager"
	databaseClusterKind          = "DatabaseCluster"
	databaseClusterAPIVersion    = "everest.percona.com/v1alpha1"
	restartAnnotationKey         = "everest.percona.com/restart"
	managedByKey                 = "everest.percona.com/managed-by"
	// ContainerStateWaiting represents a state when container requires some
	// operations being done in order to complete start up.
	ContainerStateWaiting ContainerState = "waiting"
	// ContainerStateTerminated indicates that container began execution and
	// then either ran to completion or failed for some reason.
	ContainerStateTerminated ContainerState = "terminated"
	disableTelemetryEnvVar                  = "DISABLE_TELEMETRY"

	olmNamespace = "olm"

	// APIVersionCoreosV1 constant for some API requests.
	APIVersionCoreosV1 = "operators.coreos.com/v1"

	pollInterval = 1 * time.Second
	pollDuration = 300 * time.Second
)

var (
	// ErrEmptyVersionTag Got an empty version tag from GitHub API.
	ErrEmptyVersionTag       error = errors.New("got an empty version tag from Github")
	errNoEverestOperatorPods       = errors.New("no instances of everest-operator are running")
)

// Kubernetes is a client for Kubernetes.
type Kubernetes struct {
	client     client.KubeClientConnector
	l          *zap.SugaredLogger
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
func New(kubeconfigPath string, l *zap.SugaredLogger) (*Kubernetes, error) {
	client, err := client.NewFromKubeConfig(kubeconfigPath, l)
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		client: client,
		l:      l.With("component", "kubernetes"),
		httpClient: &http.Client{
			Timeout: time.Second * 5,
			Transport: &http.Transport{
				MaxIdleConns:    1,
				IdleConnTimeout: 10 * time.Second,
			},
		},
		kubeconfig: kubeconfigPath,
	}, nil
}

// Config returns *rest.Config.
func (k *Kubernetes) Config() *rest.Config {
	return k.client.Config()
}

// NewEmpty returns new Kubernetes object.
func NewEmpty(l *zap.SugaredLogger) *Kubernetes {
	return &Kubernetes{
		client: &client.Client{},
		l:      l.With("component", "kubernetes"),
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
func (k *Kubernetes) ListDatabaseClusters(ctx context.Context) (*everestv1alpha1.DatabaseClusterList, error) {
	return k.client.ListDatabaseClusters(ctx)
}

// GetDatabaseCluster returns PXC clusters by provided name.
func (k *Kubernetes) GetDatabaseCluster(ctx context.Context, name string) (*everestv1alpha1.DatabaseCluster, error) {
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
func (k *Kubernetes) PatchDatabaseCluster(cluster *everestv1alpha1.DatabaseCluster) error {
	return k.client.ApplyObject(cluster)
}

// CreateDatabaseCluster creates database cluster.
func (k *Kubernetes) CreateDatabaseCluster(cluster *everestv1alpha1.DatabaseCluster) error {
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
	return k.getOperatorVersion(ctx, everestDeploymentName, everestOperatorContainerName)
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
func (k *Kubernetes) CreatePMMSecret(namespace, secretName string, secrets map[string][]byte) error {
	secret := &corev1.Secret{ //nolint: exhaustruct
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: secrets,
	}
	return k.client.ApplyObject(secret)
}

// CreateRestore creates a restore.
func (k *Kubernetes) CreateRestore(restore *everestv1alpha1.DatabaseClusterRestore) error {
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
		return nil, errors.Join(err, errors.New("couldn't get logs"))
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
		return nil, errors.Join(err, errors.New("couldn't describe pod"))
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
		return nil, errors.Join(err, errors.New("could not get nodes of Kubernetes cluster"))
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
func (k *Kubernetes) InstallOLMOperator(ctx context.Context, upgrade bool) error {
	deployment, err := k.client.GetDeployment(ctx, "olm-operator", "olm")
	if err == nil && deployment != nil && deployment.ObjectMeta.Name != "" && !upgrade {
		k.l.Info("OLM operator is already installed")
		return nil // already installed
	}

	resources, err := k.applyResources(ctx)
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
		return errors.Join(err, errors.New("error while waiting for deployment rollout"))
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
			return fmt.Errorf("subscription/%s failed to install CSV: %w", subscriptionKey.Name, err)
		}
		log.Printf("Waiting for clusterserviceversion/%s to reach 'Succeeded' phase", csvKey.Name)
		if err := k.client.DoCSVWait(ctx, csvKey); err != nil {
			return fmt.Errorf("clusterserviceversion/%s failed to reach 'Succeeded' phase", csvKey.Name)
		}
	}

	return nil
}

// InstallPerconaCatalog installs percona catalog and ensures that packages are available.
func (k *Kubernetes) InstallPerconaCatalog(ctx context.Context) error {
	data, err := fs.ReadFile(data.OLMCRDs, "crds/olm/percona-dbaas-catalog.yaml")
	if err != nil {
		return errors.Join(err, errors.New("failed to read percona catalog file"))
	}
	o := make(map[string]interface{})
	if err := yamlv3.Unmarshal(data, &o); err != nil {
		return err
	}

	if err := unstructured.SetNestedField(o, everestVersion.CatalogImage(), "spec", "image"); err != nil {
		return err
	}
	data, err = yamlv3.Marshal(o)
	if err != nil {
		return err
	}

	if err := k.client.ApplyFile(data); err != nil {
		return errors.Join(err, errors.New("cannot apply percona catalog file"))
	}
	if err := k.client.DoPackageWait(ctx, "everest-operator"); err != nil {
		return errors.Join(err, errors.New("timeout waiting for package"))
	}
	return nil
}

func (k *Kubernetes) applyResources(ctx context.Context) ([]unstructured.Unstructured, error) {
	files := []string{
		"crds/olm/crds.yaml",
		"crds/olm/olm.yaml",
	}

	resources := []unstructured.Unstructured{}
	for _, f := range files {
		// The scopelint linter warns about using the f variable in a function.
		// While it's safe, we assign f := f to silent the warning.
		f := f

		data, err := fs.ReadFile(data.OLMCRDs, f)
		if err != nil {
			return nil, errors.Join(err, fmt.Errorf("failed to read %q file", f))
		}

		applyFile := func(ctx context.Context) (bool, error) {
			k.l.Debugf("Applying %q file", f)
			if err := k.client.ApplyFile(data); err != nil {
				k.l.Debug(errors.Join(err, fmt.Errorf("cannot apply %q file", f)))
				k.l.Warn(fmt.Errorf("cannot apply %q file. Reapplying it", f))
				return false, nil
			}
			return true, nil
		}

		if err := wait.PollUntilContextTimeout(ctx, time.Second, 30*time.Second, true, applyFile); err != nil {
			return nil, errors.Join(err, fmt.Errorf("cannot apply %q file", f))
		}

		r, err := decodeResources(data)
		if err != nil {
			return nil, errors.Join(err, fmt.Errorf("cannot decode resources in %q", f))
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
		return errors.Join(err, errors.New("error while waiting for deployment rollout"))
	}
	if err := k.client.DoRolloutWait(ctx, types.NamespacedName{Namespace: "olm", Name: "catalog-operator"}); err != nil {
		return errors.Join(err, errors.New("error while waiting for deployment rollout"))
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
	TargetNamespaces       []string
	SubscriptionConfig     *v1alpha1.SubscriptionConfig
}

// InstallOperator installs an operator via OLM.
func (k *Kubernetes) InstallOperator(ctx context.Context, req InstallOperatorRequest) error {
	disableTelemetry, ok := os.LookupEnv(disableTelemetryEnvVar)
	if !ok || disableTelemetry != "true" {
		disableTelemetry = "false"
	}
	config := &olmv1alpha1.SubscriptionConfig{Env: []corev1.EnvVar{}}
	if req.SubscriptionConfig != nil {
		config = req.SubscriptionConfig
	}
	config.Env = append(config.Env, corev1.EnvVar{
		Name:  disableTelemetryEnvVar,
		Value: disableTelemetry,
	})
	subscription := &olmv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			Kind:       olmv1alpha1.SubscriptionKind,
			APIVersion: olmv1alpha1.SubscriptionCRDAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.Namespace,
			Name:      req.Name,
		},
		Spec: &v1alpha1.SubscriptionSpec{
			CatalogSource:          req.CatalogSource,
			CatalogSourceNamespace: "olm",
			Package:                req.Name,
			Channel:                req.Channel,
			StartingCSV:            req.StartingCSV,
			InstallPlanApproval:    olmv1alpha1.ApprovalManual,
			Config:                 config,
		},
	}
	subs, err := k.client.CreateSubscription(ctx, req.Namespace, subscription)

	if err != nil {
		return errors.Join(err, errors.New("cannot create a subscription to install the operator"))
	}

	err = wait.PollUntilContextTimeout(ctx, pollInterval, pollDuration, false, func(ctx context.Context) (bool, error) {
		k.l.Debugf("Polling subscription %s/%s", req.Namespace, req.Name)
		subs, err = k.client.GetSubscription(ctx, req.Namespace, req.Name)
		if err != nil {
			return false, errors.Join(err, fmt.Errorf("cannot get an install plan for the operator subscription: %q", req.Name))
		}
		if subs == nil || (subs != nil && subs.Status.InstallPlanRef == nil) {
			return false, nil
		}

		return k.approveInstallPlan(ctx, req.Namespace, subs.Status.InstallPlanRef.Name)
	})
	if err != nil {
		return err
	}
	deploymentName := req.Name
	if req.Name == "everest-operator" {
		deploymentName = everestDeploymentName
	}
	if req.Name == "victoriametrics-operator" {
		deploymentName = "vm-operator-vm-operator"
	}

	return k.client.DoRolloutWait(ctx, types.NamespacedName{Namespace: req.Namespace, Name: deploymentName})
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
		if ok := errors.As(err, &sErr); ok && sErr.Status().Reason == metav1.StatusReasonConflict {
			// The install plan has changed. We retry to get an updated install plan.
			k.l.Debugf("Retrying install plan update due to a version conflict. Error: %s", err)
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (k *Kubernetes) CreateOperatorGroup(ctx context.Context, name, namespace string, targetNamespaces []string) error {
	targetNamespaces = append(targetNamespaces, namespace)
	og, err := k.client.GetOperatorGroup(ctx, namespace, name)
	if err != nil && apierrors.IsNotFound(err) {
		return err
	}
	if err != nil && apierrors.IsNotFound(err) {
		_, err = k.client.CreateOperatorGroup(ctx, namespace, name, targetNamespaces)
		if err != nil {
			return err
		}
		return nil
	}
	var update bool
	for _, namespace := range targetNamespaces {
		namespace := namespace
		if !contains(og.Spec.TargetNamespaces, namespace) {
			update = true
		}
	}
	if update {
		og.Spec.TargetNamespaces = targetNamespaces
		return k.client.ApplyObject(og)
	}
	return nil

}
func contains(s []string, e string) bool {
	for _, a := range s {
		a := a
		if a == e {
			return true
		}
	}
	return false
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
		return nil, fmt.Errorf("cannot get subscription for %q operator", name)
	}

	ip, err := k.client.GetInstallPlan(ctx, namespace, subs.Status.Install.Name)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("cannot get install plan to upgrade %q", name))
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

// ProvisionMonitoring provisions PMM monitoring.
func (k *Kubernetes) ProvisionMonitoring(namespace string) error {
	for _, path := range k.victoriaMetricsCRDFiles() {
		file, err := data.OLMCRDs.ReadFile(path)
		if err != nil {
			return err
		}
		// retry 3 times because applying vmagent spec might take some time.
		for i := 0; i < 3; i++ {
			k.l.Debugf("Applying file %s", path)

			err = k.client.ApplyManifestFile(file, namespace)
			if err != nil {
				k.l.Debugf("%s: retrying after error: %s", path, err)
				time.Sleep(10 * time.Second)
				continue
			}
			break
		}
		if err != nil {
			return errors.Join(err, fmt.Errorf("cannot apply file: %q", path))
		}
	}

	return nil
}

func (k *Kubernetes) victoriaMetricsCRDFiles() []string {
	return []string{
		"crds/victoriametrics/crs/vmagent_rbac_account.yaml",
		"crds/victoriametrics/crs/vmagent_rbac_role.yaml",
		"crds/victoriametrics/crs/vmagent_rbac_role_binding.yaml",
		"crds/victoriametrics/crs/vmnodescrape.yaml",
		"crds/victoriametrics/crs/vmpodscrape.yaml",
		"crds/victoriametrics/kube-state-metrics/service-account.yaml",
		"crds/victoriametrics/kube-state-metrics/cluster-role.yaml",
		"crds/victoriametrics/kube-state-metrics/cluster-role-binding.yaml",
		"crds/victoriametrics/kube-state-metrics/deployment.yaml",
		"crds/victoriametrics/kube-state-metrics/service.yaml",
		"crds/victoriametrics/kube-state-metrics.yaml",
	}
}

// RestartEverest restarts everest pod.
func (k *Kubernetes) RestartEverest(ctx context.Context, name, namespace string) error {
	var podsToRestart []corev1.Pod
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		p, err := k.getEverestPods(ctx, name, namespace)
		if err != nil {
			return false, err
		}
		podsToRestart = p
		return true, nil
	})
	if err != nil {
		return err
	}
	for _, pod := range podsToRestart {
		err = k.client.DeletePod(ctx, namespace, pod.Name)
		if err != nil {
			return err
		}
	}

	return wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		pods, err := k.getEverestPods(ctx, name, namespace)
		if err != nil {
			return false, err
		}
		podsStatuses := make(map[string]struct{})
		for _, pod := range pods {
			pod := pod
			for _, restartedPod := range podsToRestart {
				if restartedPod.UID == pod.UID {
					return false, nil
				}
			}
			if pod.Status.Phase == corev1.PodRunning && pod.Status.ContainerStatuses[0].Ready {
				podsStatuses[string(pod.UID)] = struct{}{}
			}
		}
		return len(podsStatuses) == len(pods), nil
	})
}

func (k *Kubernetes) getEverestPods(ctx context.Context, name, namespace string) ([]corev1.Pod, error) {
	podList, err := k.client.ListPods(ctx, namespace, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app.kubernetes.io/name": name,
			},
		}),
	})
	if err != nil {
		return []corev1.Pod{}, err
	}
	if len(podList.Items) == 0 {
		return []corev1.Pod{}, errNoEverestOperatorPods
	}
	return podList.Items, nil
}

// ListEngineDeploymentNames returns a string array containing found engine deployments for the Everest.
func (k *Kubernetes) ListEngineDeploymentNames(ctx context.Context, namespace string) ([]string, error) {
	names := []string{}
	deploymentList, err := k.client.ListDeployments(ctx, namespace)
	if err != nil {
		return names, err
	}
	for _, deployment := range deploymentList.Items {
		switch deployment.Name {
		case pxcDeploymentName, psmdbDeploymentName, postgresDeploymentName:
			names = append(names, deployment.Name)
		}
	}
	return names, nil
}

// ApplyObject applies object.
func (k *Kubernetes) ApplyObject(obj runtime.Object) error {
	return k.client.ApplyObject(obj)
}

// InstallEverest downloads the manifest file and applies it against provisioned k8s cluster.
func (k *Kubernetes) InstallEverest(ctx context.Context, namespace string) error {
	data, err := k.getManifestData(ctx)
	if err != nil {
		return errors.Join(err, errors.New("failed downloading everest monitoring file"))
	}

	err = k.client.ApplyManifestFile(data, namespace)

	if err != nil {
		return errors.Join(err, errors.New("failed applying manifest file"))
	}
	if err := k.client.DoRolloutWait(ctx, types.NamespacedName{Name: PerconaEverestDeploymentName, Namespace: namespace}); err != nil {
		return errors.Join(err, errors.New("failed waiting for the Everest deployment to be ready"))
	}
	return nil
}

func (k *Kubernetes) getManifestData(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, everestVersion.ManifestURL(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	return io.ReadAll(resp.Body)
}

// DeleteEverest downloads the manifest file and deletes it from provisioned k8s cluster.
func (k *Kubernetes) DeleteEverest(ctx context.Context, namespace string) error {
	data, err := k.getManifestData(ctx)
	if err != nil {
		return errors.Join(err, errors.New("failed downloading everest monitoring file"))
	}

	err = k.client.DeleteManifestFile(data, namespace)
	if err != nil {
		return errors.Join(err, errors.New("failed deleting manifest file"))
	}
	return nil
}

// PersistNamespaces stores provided namespaces in the configMap.
func (k *Kubernetes) PersistNamespaces(ctx context.Context, namespace string, namespaces []string) error {
	namespaces = append(namespaces, namespace)
	cMap, err := k.client.GetConfigMap(ctx, namespace, "everest-namespaces")
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err != nil && apierrors.IsNotFound(err) {
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "everest-namespaces",
				Namespace: namespace,
			},
			Data: map[string]string{
				"namespaces": strings.Join(namespaces, ","),
			},
		}
		_, err := k.client.CreateConfigMap(ctx, namespace, configMap)
		return err
	}
	if cMap != nil && cMap.Name != "everest-namespaces" {
		return nil
	}
	v, ok := cMap.Data["namespaces"]
	if !ok {
		return nil
	}
	var update bool
	existingNamespaces := strings.Split(v, ",")
	for _, namespace := range namespaces {
		namespace := namespace
		if !contains(existingNamespaces, namespace) {
			update = true
		}
	}
	if update {
		cMap.Data["namespaces"] = strings.Join(namespaces, ",")
		return k.client.ApplyObject(cMap)
	}
	return nil

}

// GetDeployment returns k8s deployment by provided name and namespace.
func (k *Kubernetes) GetDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error) {
	return k.client.GetDeployment(ctx, name, namespace)
}
