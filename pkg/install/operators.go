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
// Package install holds the main logic for installation commands.

// Package install ...
package install

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/percona/percona-everest-backend/client"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/percona/percona-everest-cli/commands/common"
	everestClient "github.com/percona/percona-everest-cli/pkg/everest/client"
	"github.com/percona/percona-everest-cli/pkg/kubernetes"
)

// Operators implements the main logic for commands.
type Operators struct {
	l *zap.SugaredLogger

	config        OperatorsConfig
	everestClient everestClientConnector
	kubeClient    *kubernetes.Kubernetes

	// monitoringInstanceName stores the resolved monitoring instance name.
	monitoringInstanceName string
}

const (
	catalogSourceNamespace           = "olm"
	operatorGroup                    = "percona-operators-group"
	catalogSource                    = "percona-dbaas-catalog"
	everestOperatorName              = "everest-operator"
	pxcOperatorName                  = "percona-xtradb-cluster-operator"
	psmdbOperatorName                = "percona-server-mongodb-operator"
	pgOperatorName                   = "percona-postgresql-operator"
	vmOperatorName                   = "victoriametrics-operator"
	everestServiceAccount            = "everest-admin"
	everestServiceAccountRole        = "everest-admin-role"
	everestServiceAccountRoleBinding = "everest-admin-role-binding"
	everestServiceAccountTokenSecret = "everest-admin-token"
	operatorInstallThreads           = 1
)

type (
	// MonitoringType identifies type of monitoring to be used.
	MonitoringType string

	// OperatorsConfig stores configuration for the operators.
	OperatorsConfig struct {
		// Name of the Kubernetes Cluster
		Name string
		// Namespace defines the namespace operators shall be installed to.
		Namespace string
		// SkipWizard skips wizard during installation.
		SkipWizard bool `mapstructure:"skip-wizard"`
		// KubeconfigPath is a path to a kubeconfig
		KubeconfigPath string `mapstructure:"kubeconfig"`

		Backup     BackupConfig
		Channel    ChannelConfig
		Everest    EverestConfig
		Monitoring MonitoringConfig
		Operator   OperatorConfig
	}

	// BackupConfig stores configuration for backup.
	BackupConfig struct {
		// Enable is true if backup shall be enabled.
		Enable bool
		// Name stores name of the backup.
		Name string
		// Endpoint stores URL to backup.
		Endpoint string
		// Bucket stores name of the bucket for backup.
		Bucket string
		// AccessKey stores username for backup.
		AccessKey string `mapstructure:"access-key"`
		// SecretKey stores password for backup.
		SecretKey string `mapstructure:"secret-key"`
		// Region stores region for backup.
		Region string
	}

	// EverestConfig stores config for Everest.
	EverestConfig struct {
		// Endpoint stores URL to Everest.
		Endpoint string
	}

	// MonitoringConfig stores configuration for monitoring.
	MonitoringConfig struct {
		// Enable is true if monitoring shall be enabled.
		Enable bool
		// InstanceName stores monitoring instance name from Everest.
		// If provided, the other monitoring configuration is ignored.
		InstanceName string `mapstructure:"instance-name"`
		// NewInstanceName defines name for a new monitoring instance
		// if it's created.
		NewInstanceName string `mapstructure:"new-instance-name"`
		// Type stores the type of monitoring to be used.
		Type MonitoringType
		// PMM stores configuration for PMM monitoring type.
		PMM *PMMConfig
	}

	// OperatorConfig identifies which operators shall be installed.
	OperatorConfig struct {
		// PG stores if PostgresSQL shall be installed.
		PG bool `mapstructure:"postgresql"`
		// PSMDB stores if MongoDB shall be installed.
		PSMDB bool `mapstructure:"mongodb"`
		// PXC stores if XtraDB Cluster shall be installed.
		PXC bool `mapstructure:"xtradb-cluster"`
	}

	// PMMConfig stores configuration for PMM monitoring type.
	PMMConfig struct {
		// Endpoint stores URL to PMM.
		Endpoint string
		// Username stores username for authentication against PMM.
		Username string
		// Password stores password for authentication against PMM.
		Password string
	}

	// ChannelConfig stores configuration for operator channels.
	ChannelConfig struct {
		// Everest stores channel for Everest.
		Everest string
		// PG stores channel for PostgreSQL.
		PG string `mapstructure:"postgresql"`
		// PSMDB stores channel for MongoDB.
		PSMDB string `mapstructure:"mongodb"`
		// PXC stores channel for xtradb cluster.
		PXC string `mapstructure:"xtradb-cluster"`
		// VictoriaMetrics stores channel for VictoriaMetrics.
		VictoriaMetrics string `mapstructure:"victoria-metrics"`
	}
)

// NewOperators returns a new Operators struct.
func NewOperators(c OperatorsConfig, l *zap.SugaredLogger) (*Operators, error) {
	cli := &Operators{
		config: c,
		l:      l.With("component", "install/operators"),
	}

	k, err := kubernetes.New(c.KubeconfigPath, cli.l)
	if err != nil {
		var u *url.Error
		if errors.As(err, &u) {
			cli.l.Error("Could not connect to Kubernetes. " +
				"Make sure Kubernetes is running and is accessible from this computer/server.")
		}
		return nil, err
	}
	cli.kubeClient = k
	return cli, nil
}

// Run runs the operators installation process.
func (o *Operators) Run(ctx context.Context) error {
	if !o.config.SkipWizard {
		if err := o.runWizard(ctx); err != nil {
			return err
		}
	}

	if o.everestClient == nil {
		if err := o.configureEverestConnector(); err != nil {
			return err
		}
	}

	if err := o.checkEverestConnection(ctx); err != nil {
		var u *url.Error
		if errors.As(err, &u) {
			o.l.Debug(err)

			l := o.l.WithOptions(zap.AddStacktrace(zap.DPanicLevel))
			l.Error("Could not connect to Everest. " +
				"Make sure Everest is running and is accessible from this machine.",
			)
			return common.ErrExitWithError
		}

		return errors.Join(err, errors.New("could not check connection to Everest"))
	}

	if err := o.resolveMonitoringInstanceName(ctx); err != nil {
		return err
	}

	return o.performProvisioning(ctx)
}

func (o *Operators) checkEverestConnection(ctx context.Context) error {
	o.l.Info("Checking connection to Everest")
	_, err := o.everestClient.ListMonitoringInstances(ctx)
	return err
}

func (o *Operators) performProvisioning(ctx context.Context) error {
	if err := o.provisionNamespace(); err != nil {
		return err
	}
	if err := o.provisionAllOperators(ctx); err != nil {
		return err
	}

	var k *client.KubernetesCluster
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		k, err = o.connectToEverest(gCtx)
		return err
	})
	g.Go(func() error {
		return o.createEverestBackupStorage(gCtx)
	})

	if err := g.Wait(); err != nil {
		return err
	}

	if o.config.Monitoring.Enable {
		o.l.Info("Deploying VMAgent to k8s cluster")

		// We retry for a bit since the MonitoringConfig may not be properly
		// deployed yet and we get a HTTP 500 in this case.
		err := wait.PollUntilContextTimeout(ctx, 3*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
			o.l.Debug("Trying to enable Kubernetes cluster monitoring")
			err := o.everestClient.SetKubernetesClusterMonitoring(ctx, k.Id, client.KubernetesClusterMonitoring{
				Enable:                 true,
				MonitoringInstanceName: o.monitoringInstanceName,
			})
			if err != nil {
				o.l.Debug(errors.Join(err, errors.New("could not enable Kubernetes cluster monitoring")))
				return false, nil
			}

			return true, nil
		})
		if err != nil {
			return errors.Join(err, errors.New("could not enable Kubernetes cluster monitoring"))
		}

		o.l.Info("VMAgent deployed successfully")
	}

	return nil
}

func (o *Operators) resolveMonitoringInstanceName(ctx context.Context) error {
	if !o.config.Monitoring.Enable || o.monitoringInstanceName != "" {
		return nil
	}

	if o.config.Monitoring.InstanceName != "" {
		i, err := o.everestClient.GetMonitoringInstance(ctx, o.config.Monitoring.InstanceName)
		if err != nil {
			return errors.Join(err, fmt.Errorf("could not get monitoring instance with name %s from Everest", o.config.Monitoring.InstanceName))
		}
		o.monitoringInstanceName = i.Name
		return nil
	}

	if o.config.Monitoring.NewInstanceName == "" {
		return errors.New("monitoring.new-instance-name is required when creating a new monitoring instance")
	}

	err := o.createPMMMonitoringInstance(
		ctx, o.config.Monitoring.NewInstanceName, o.config.Monitoring.PMM.Endpoint,
		o.config.Monitoring.PMM.Username, o.config.Monitoring.PMM.Password,
	)
	if err != nil {
		return errors.Join(err, errors.New("could not create a new PMM monitoring instance in Everest"))
	}

	o.monitoringInstanceName = o.config.Monitoring.NewInstanceName

	return nil
}

func (o *Operators) createPMMMonitoringInstance(ctx context.Context, name, url, username, password string) error {
	_, err := o.everestClient.CreateMonitoringInstance(ctx, client.MonitoringInstanceCreateParams{
		Type: client.MonitoringInstanceCreateParamsTypePmm,
		Name: name,
		Url:  url,
		Pmm: &client.PMMMonitoringInstanceSpec{
			User:     username,
			Password: password,
		},
	})
	if err != nil {
		return errors.Join(err, errors.New("could not create a new monitoring instance"))
	}

	return nil
}

func (o *Operators) configureEverestConnector() error {
	cl, err := everestClient.NewEverestFromURL(o.config.Everest.Endpoint)
	if err != nil {
		return err
	}
	o.everestClient = cl

	return nil
}

// runWizard runs installation wizard.
func (o *Operators) runWizard(ctx context.Context) error {
	if err := o.runEverestWizard(); err != nil {
		return err
	}

	if err := o.runMonitoringWizard(ctx); err != nil {
		return err
	}

	if err := o.runBackupWizard(); err != nil {
		return err
	}

	return o.runOperatorsWizard()
}

func (o *Operators) runEverestWizard() error {
	pEndpoint := &survey.Input{
		Message: "Everest URL",
		Default: o.config.Everest.Endpoint,
	}
	if err := survey.AskOne(pEndpoint, &o.config.Everest.Endpoint); err != nil {
		return err
	}

	if err := o.configureEverestConnector(); err != nil {
		return err
	}

	clusterName := o.kubeClient.ClusterName()
	if o.config.Name != "" {
		clusterName = o.config.Name
	}

	pName := &survey.Input{
		Message: "Choose your Kubernetes Cluster name",
		Default: clusterName,
	}

	return survey.AskOne(pName, &o.config.Name)
}

func (o *Operators) runMonitoringWizard(ctx context.Context) error {
	pMonitor := &survey.Confirm{
		Message: "Do you want to enable monitoring?",
		Default: o.config.Monitoring.Enable,
	}
	err := survey.AskOne(pMonitor, &o.config.Monitoring.Enable)
	if err != nil {
		return err
	}

	if o.config.Monitoring.Enable {
		if err := o.runMonitoringConfigWizard(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (o *Operators) runMonitoringConfigWizard(ctx context.Context) error {
	if o.config.Monitoring.PMM == nil {
		o.config.Monitoring.PMM = &PMMConfig{}
	}

	if o.config.Monitoring.InstanceName == "" {
		if err := o.runMonitoringURLWizard(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (o *Operators) runMonitoringURLWizard(ctx context.Context) error {
	instances, err := o.everestClient.ListMonitoringInstances(ctx)
	if err != nil {
		var u *url.Error
		if errors.As(err, &u) {
			o.l.Debug(err)
		} else {
			o.l.Error(err)
		}

		l := o.l.WithOptions(zap.AddStacktrace(zap.DPanicLevel))
		l.Error("Could not get a list of monitoring instances from Everest. " +
			"Make sure Everest is running and is accessible from this machine.")
		return common.ErrExitWithError
	}

	if len(instances) == 0 {
		return o.runMonitoringNewURLWizard()
	}

	opts := make([]string, 0, len(instances)+1)
	for _, i := range instances {
		opts = append(opts, i.Name)
	}
	opts = append(opts, "Add new monitoring instance")

	pInstance := &survey.Select{
		Message: "Select monitoring instance:",
		Options: opts,
	}
	ix := 0
	if err := survey.AskOne(pInstance, &ix); err != nil {
		return err
	}

	if ix > len(instances)-1 {
		return o.runMonitoringNewURLWizard()
	}

	o.monitoringInstanceName = instances[ix].Name

	return nil
}

func (o *Operators) runMonitoringNewURLWizard() error {
	pURL := &survey.Input{
		Message: "PMM URL Endpoint",
		Default: o.config.Monitoring.PMM.Endpoint,
	}
	if err := survey.AskOne(
		pURL,
		&o.config.Monitoring.PMM.Endpoint,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}

	pUser := &survey.Input{
		Message: "Username",
		Default: o.config.Monitoring.PMM.Username,
	}
	if err := survey.AskOne(
		pUser,
		&o.config.Monitoring.PMM.Username,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}

	pPass := &survey.Password{Message: "Password"}
	if err := survey.AskOne(
		pPass,
		&o.config.Monitoring.PMM.Password,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}

	pName := &survey.Input{
		Message: "Name for the new monitoring instance",
		Default: o.config.Monitoring.NewInstanceName,
	}
	if err := survey.AskOne(
		pName,
		&o.config.Monitoring.NewInstanceName,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}

	return nil
}

func (o *Operators) runBackupWizard() error {
	pBackup := &survey.Confirm{
		Message: "Do you want to enable backups?",
		Default: o.config.Backup.Enable,
	}

	if err := survey.AskOne(pBackup, &o.config.Backup.Enable); err != nil {
		return err
	}

	if o.config.Backup.Enable {
		return o.runBackupConfigWizard()
	}

	return nil
}

func (o *Operators) runBackupConfigWizard() error {
	pName := &survey.Input{
		Message: "Name",
		Default: o.config.Backup.Name,
	}
	if err := survey.AskOne(
		pName,
		&o.config.Backup.Name,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}

	pURL := &survey.Input{
		Message: "URL Endpoint",
		Default: o.config.Backup.Endpoint,
	}
	if err := survey.AskOne(
		pURL,
		&o.config.Backup.Endpoint,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}

	pRegion := &survey.Input{
		Message: "Region",
		Default: o.config.Backup.Region,
	}
	if err := survey.AskOne(
		pRegion,
		&o.config.Backup.Region,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}

	pBucket := &survey.Input{
		Message: "Bucket",
		Default: o.config.Backup.Bucket,
	}
	if err := survey.AskOne(
		pBucket,
		&o.config.Backup.Bucket,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}

	return o.runBackupCredentialsConfigWizard()
}

func (o *Operators) runBackupCredentialsConfigWizard() error {
	pUser := &survey.Input{
		Message: "Access key",
		Default: o.config.Backup.AccessKey,
	}
	if err := survey.AskOne(
		pUser,
		&o.config.Backup.AccessKey,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}

	pPass := &survey.Password{Message: "Secret key"}
	return survey.AskOne(
		pPass,
		&o.config.Backup.SecretKey,
		survey.WithValidator(survey.Required),
	)
}

func (o *Operators) runOperatorsWizard() error {
	operatorOpts := []struct {
		label    string
		boolFlag *bool
	}{
		{"MySQL", &o.config.Operator.PXC},
		{"MongoDB", &o.config.Operator.PSMDB},
		{"PostgreSQL", &o.config.Operator.PG},
	}
	operatorLabels := make([]string, 0, len(operatorOpts))
	for _, v := range operatorOpts {
		operatorLabels = append(operatorLabels, v.label)
	}
	operatorDefaults := make([]string, 0, len(operatorOpts))
	for _, v := range operatorOpts {
		if *v.boolFlag {
			operatorDefaults = append(operatorDefaults, v.label)
		}
	}

	pOps := &survey.MultiSelect{
		Message: "What operators do you want to install?",
		Default: operatorDefaults,
		Options: operatorLabels,
	}
	opIndexes := []int{}
	if err := survey.AskOne(
		pOps,
		&opIndexes,
		survey.WithValidator(survey.MinItems(1)),
	); err != nil {
		return err
	}

	if len(opIndexes) == 0 {
		return errors.New("at least one operator needs to be selected")
	}

	// We reset all flags to false so we select only
	// the ones which the user selected in the multiselect.
	for _, op := range operatorOpts {
		*op.boolFlag = false
	}

	for _, i := range opIndexes {
		o.l.Debugf("Enabling %s operator", operatorOpts[i].label)
		*operatorOpts[i].boolFlag = true
	}

	return nil
}

// provisionNamespace provisions a namespace for Everest.
func (o *Operators) provisionNamespace() error {
	o.l.Infof("Creating namespace %s", o.config.Namespace)
	err := o.kubeClient.CreateNamespace(o.config.Namespace)
	if err != nil {
		return errors.Join(err, errors.New("could not provision namespace"))
	}

	o.l.Infof("Namespace %s has been created", o.config.Namespace)
	return nil
}

// provisionAllOperators provisions all configured operators to a k8s cluster.
func (o *Operators) provisionAllOperators(ctx context.Context) error {
	o.l.Info("Started provisioning the cluster")

	if err := o.provisionOLM(ctx); err != nil {
		return err
	}

	if err := o.provisionOperators(ctx); err != nil {
		return err
	}

	if o.config.Monitoring.Enable {
		if err := o.provisionMonitoring(); err != nil {
			return err
		}
	}

	return nil
}

func (o *Operators) provisionOLM(ctx context.Context) error {
	o.l.Info("Installing Operator Lifecycle Manager")
	if err := o.kubeClient.InstallOLMOperator(ctx); err != nil {
		o.l.Error("failed installing OLM")
		return err
	}
	o.l.Info("OLM has been installed")
	o.l.Info("Installing Percona OLM Catalog")
	if err := o.kubeClient.InstallPerconaCatalog(ctx); err != nil {
		o.l.Errorf("failed installing OLM catalog: %v", err)
		return err
	}
	o.l.Info("Percona OLM Catalog has been installed")

	return nil
}

func (o *Operators) provisionOperators(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)
	// We set the limit to 1 since operator installation
	// requires an update to the same installation plan which
	// results in race-conditions with a higher limit.
	// The limit can be removed after it's refactored.
	g.SetLimit(operatorInstallThreads)

	if o.config.Monitoring.Enable {
		g.Go(o.installOperator(gCtx, o.config.Channel.VictoriaMetrics, vmOperatorName))
	}

	if o.config.Operator.PXC {
		g.Go(o.installOperator(gCtx, o.config.Channel.PXC, pxcOperatorName))
	}
	if o.config.Operator.PSMDB {
		g.Go(o.installOperator(gCtx, o.config.Channel.PSMDB, psmdbOperatorName))
	}
	if o.config.Operator.PG {
		g.Go(o.installOperator(gCtx, o.config.Channel.PG, pgOperatorName))
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return o.installOperator(ctx, o.config.Channel.Everest, everestOperatorName)()
}

func (o *Operators) installOperator(ctx context.Context, channel, operatorName string) func() error {
	return func() error {
		// We check if the context has not been cancelled yet to return early
		if err := ctx.Err(); err != nil {
			o.l.Debugf("Cancelled %s operator installation due to context error: %s", operatorName, err)
			return err
		}

		o.l.Infof("Installing %s operator", operatorName)

		params := kubernetes.InstallOperatorRequest{
			Namespace:              o.config.Namespace,
			Name:                   operatorName,
			OperatorGroup:          operatorGroup,
			CatalogSource:          catalogSource,
			CatalogSourceNamespace: catalogSourceNamespace,
			Channel:                channel,
			InstallPlanApproval:    v1alpha1.ApprovalManual,
		}

		if err := o.kubeClient.InstallOperator(ctx, params); err != nil {
			o.l.Errorf("failed installing %s operator", operatorName)
			return err
		}
		o.l.Infof("%s operator has been installed", operatorName)

		return nil
	}
}

func (o *Operators) provisionMonitoring() error {
	l := o.l.With("action", "monitoring")
	l.Info("Preparing k8s cluster for monitoring")
	if err := o.kubeClient.ProvisionMonitoring(o.config.Namespace); err != nil {
		return errors.Join(err, errors.New("could not provision monitoring configuration"))
	}

	l.Info("K8s cluster monitoring has been provisioned successfully")

	return nil
}

// connectToEverest connects the k8s cluster to Everest.
func (o *Operators) connectToEverest(ctx context.Context) (*client.KubernetesCluster, error) {
	if err := o.prepareServiceAccount(); err != nil {
		return nil, errors.Join(err, errors.New("could not prepare a service account"))
	}

	o.l.Info("Generating kubeconfig")
	kubeconfig, err := o.getServiceAccountKubeConfig(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not get a new kubeconfig file for a service account"))
	}

	o.l.Info("Connecting your Kubernetes cluster to Everest")

	k, err := o.everestClient.RegisterKubernetesCluster(ctx, client.CreateKubernetesClusterParams{
		Kubeconfig: base64.StdEncoding.EncodeToString([]byte(kubeconfig)),
		Name:       o.config.Name,
		Namespace:  &o.config.Namespace,
	})
	if err != nil {
		return nil, errors.Join(err, errors.New("could not register a new Kubernetes cluster with Everest"))
	}

	o.l.Info("Connected Kubernetes cluster to Everest")

	return k, nil
}

func (o *Operators) createEverestBackupStorage(ctx context.Context) error {
	if !o.config.Backup.Enable {
		return nil
	}

	o.l.Info("Creating a new backup storage in Everest")

	_, err := o.everestClient.CreateBackupStorage(ctx, client.CreateBackupStorageParams{
		Type:       client.CreateBackupStorageParamsTypeS3,
		Name:       o.config.Backup.Name,
		BucketName: o.config.Backup.Bucket,
		AccessKey:  o.config.Backup.AccessKey,
		SecretKey:  o.config.Backup.SecretKey,
		Url:        &o.config.Backup.Endpoint,
		Region:     o.config.Backup.Region,
	})
	if err != nil {
		return errors.Join(err, errors.New("could not create a new backup storage in Everest"))
	}

	o.l.Info("Created a new backup storage in Everest")

	return nil
}

func (o *Operators) prepareServiceAccount() error {
	o.l.Info("Creating service account for Everest")
	if err := o.kubeClient.CreateServiceAccount(everestServiceAccount, o.config.Namespace); err != nil {
		return errors.Join(err, errors.New("could not create service account"))
	}

	o.l.Info("Creating role for Everest service account")
	err := o.kubeClient.CreateRole(o.config.Namespace, everestServiceAccountRole, o.serviceAccountRolePolicyRules())
	if err != nil {
		return errors.Join(err, errors.New("could not create role"))
	}

	o.l.Info("Binding role to Everest Service account")
	err = o.kubeClient.CreateRoleBinding(
		o.config.Namespace,
		everestServiceAccountRoleBinding,
		everestServiceAccountRole,
		everestServiceAccount,
	)
	if err != nil {
		return errors.Join(err, errors.New("could not create role binding"))
	}

	o.l.Info("Creating cluster role for Everest service account")
	err = o.kubeClient.CreateClusterRole(
		everestServiceAccountRole, o.serviceAccountClusterRolePolicyRules(),
	)
	if err != nil {
		return errors.Join(err, errors.New("could not create cluster role"))
	}

	o.l.Info("Binding cluster role to Everest Service account")
	err = o.kubeClient.CreateClusterRoleBinding(
		o.config.Namespace,
		everestServiceAccountRoleBinding,
		everestServiceAccountRole,
		everestServiceAccount,
	)
	if err != nil {
		return errors.Join(err, errors.New("could not create cluster role binding"))
	}

	return nil
}

func (o *Operators) serviceAccountRolePolicyRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{"everest.percona.com"},
			Resources: []string{"databaseclusters", "databaseclusterrestores"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{"everest.percona.com"},
			Resources: []string{"databaseengines"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{"everest.percona.com"},
			Resources: []string{"databaseclusterrestores"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{"everest.percona.com"},
			Resources: []string{"databaseclusterbackups"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{"everest.percona.com"},
			Resources: []string{"backupstorages"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{"everest.percona.com"},
			Resources: []string{"monitoringconfigs"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{"operator.victoriametrics.com"},
			Resources: []string{"vmagents"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"namespaces"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"*"},
		},
	}
}

func (o *Operators) serviceAccountClusterRolePolicyRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"storageclasses"},
			Verbs:     []string{"list"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"get", "list"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"list"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumes"},
			Verbs:     []string{"list"},
		},
	}
}

func (o *Operators) getServiceAccountKubeConfig(ctx context.Context) (string, error) {
	// Create token secret
	err := o.kubeClient.CreateServiceAccountToken(everestServiceAccount, everestServiceAccountTokenSecret, o.config.Namespace)
	if err != nil {
		return "", err
	}

	var secret *corev1.Secret
	checkSecretData := func(ctx context.Context) (bool, error) {
		o.l.Debugf("Getting secret for %s", everestServiceAccountTokenSecret)
		s, err := o.kubeClient.GetSecret(ctx, everestServiceAccountTokenSecret, o.config.Namespace)
		if err != nil {
			return false, err
		}

		if _, ok := s.Data["token"]; !ok {
			return false, nil
		}

		secret = s

		return true, nil
	}
	// We poll for the secret as it's created asynchronously
	err = wait.PollUntilContextTimeout(ctx, time.Second, 10*time.Second, true, checkSecretData)
	if err != nil {
		return "", errors.Join(err, errors.New("could not get token from secret for a service account"))
	}

	return o.kubeClient.GenerateKubeConfigWithToken(everestServiceAccount, secret)
}
