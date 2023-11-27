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
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/percona/percona-everest-backend/client"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/percona/percona-everest-cli/commands/common"
	everestClient "github.com/percona/percona-everest-cli/pkg/everest/client"
	"github.com/percona/percona-everest-cli/pkg/kubernetes"
)

// Install implements the main logic for commands.
type Install struct {
	l *zap.SugaredLogger

	config        Config
	everestClient everestClientConnector
	kubeClient    *kubernetes.Kubernetes

	// monitoringInstanceName stores the resolved monitoring instance name.
	monitoringInstanceName string
}

const (
	catalogSourceNamespace    = "olm"
	everestBackendServiceName = "percona-everest-backend"
	operatorGroup             = "percona-operators-group"
	catalogSource             = "percona-everest-catalog"
	everestOperatorName       = "everest-operator"
	pxcOperatorName           = "percona-xtradb-cluster-operator"
	psmdbOperatorName         = "percona-server-mongodb-operator"
	pgOperatorName            = "percona-postgresql-operator"
	vmOperatorName            = "victoriametrics-operator"
	operatorInstallThreads    = 1

	everestServiceAccount            = "everest-admin"
	everestServiceAccountRole        = "everest-admin-role"
	everestServiceAccountRoleBinding = "everest-admin-role-binding"
	everestWatchNamespacesEnvVar     = "WATCH_NAMESPACES"
	everestNamespace                 = "percona-everest"
)

type (
	// MonitoringType identifies type of monitoring to be used.
	MonitoringType string

	// Config stores configuration for the operators.
	Config struct {
		// Name of the Kubernetes Cluster
		Name string
		// Namespaces defines namespaces that everest can operate in.
		Namespaces []string `mapstructure:"namespace"`
		// SkipWizard skips wizard during installation.
		SkipWizard bool `mapstructure:"skip-wizard"`
		// KubeconfigPath is a path to a kubeconfig
		KubeconfigPath string `mapstructure:"kubeconfig"`

		Channel    ChannelConfig
		Monitoring MonitoringConfig
		Operator   OperatorConfig
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

// NewInstall returns a new Install struct.
func NewInstall(c Config, l *zap.SugaredLogger) (*Install, error) {
	cli := &Install{
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
func (o *Install) Run(ctx context.Context) error {
	if err := o.populateConfig(); err != nil {
		return err
	}
	if err := o.createNamespace(everestNamespace); err != nil {
		return err
	}
	if err := o.installDefaultComponents(ctx); err != nil {
		return err
	}

	if err := o.provisionAllNamespaces(ctx); err != nil {
		return err
	}
	if err := o.installEverest(ctx); err != nil {
		return err
	}
	if o.config.Monitoring.Enable {
		if err := o.provisionMonitoringInAllNamespaces(ctx); err != nil {
			return err
		}
	}
	if err := o.kubeClient.PersistNamespaces(ctx, everestNamespace, o.config.Namespaces); err != nil {
		return err
	}

	return nil
}

func (o *Install) populateConfig() error {
	if !o.config.SkipWizard {
		if err := o.runWizard(); err != nil {
			return err
		}
	}

	if o.config.Name == "" {
		o.config.Name = o.kubeClient.ClusterName()
	}

	return nil
}

func (o *Install) checkEverestConnection(ctx context.Context) error {
	o.l.Info("Checking connection to Everest")
	_, err := o.everestClient.ListMonitoringInstances(ctx)
	return err
}

func (o *Install) installDefaultComponents(ctx context.Context) error {
	if err := o.provisionOLM(ctx); err != nil {
		return err
	}
	if err := o.kubeClient.CreateOperatorGroup(ctx, operatorGroup, everestNamespace, o.config.Namespaces); err != nil {
		return err
	}
	return nil
}

func (o *Install) installEverest(ctx context.Context) error {
	if err := o.installOperator(ctx, o.config.Channel.Everest, everestOperatorName, everestNamespace)(); err != nil {
		return err
	}
	d, err := o.kubeClient.GetDeployment(ctx, kubernetes.PerconaEverestDeploymentName, everestNamespace)
	var everestExists bool
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if d != nil && d.Name == kubernetes.PerconaEverestDeploymentName {
		everestExists = true
	}

	if !everestExists {
		o.l.Info(fmt.Sprintf("Deploying Everest to %s", everestNamespace))
		err = o.kubeClient.InstallEverest(ctx, everestNamespace)
		if err != nil {
			return err
		}
		o.l.Info("Everest has been installed. Configuring connection")
	}
	if err := o.configureEverestConnector(); err != nil {
		return err
	}
	return nil
}

func (o *Install) provisionAllNamespaces(ctx context.Context) error {
	for _, namespace := range o.config.Namespaces {
		namespace := namespace
		if err := o.createNamespace(namespace); err != nil {
			return err
		}
		if err := o.kubeClient.CreateOperatorGroup(ctx, operatorGroup, namespace, []string{}); err != nil {
			return err
		}

		if err := o.provisionNamespace(ctx, namespace); err != nil {
			return err
		}
		o.l.Info("Creating role for Everest service account")
		err := o.kubeClient.CreateRole(namespace, everestServiceAccountRole, o.serviceAccountRolePolicyRules())
		if err != nil {
			return errors.Join(err, errors.New("could not create role"))
		}

		o.l.Info("Binding role to Everest Service account")
		err = o.kubeClient.CreateRoleBinding(
			namespace,
			everestServiceAccountRoleBinding,
			everestServiceAccountRole,
			everestServiceAccount,
		)
		if err != nil {
			return errors.Join(err, errors.New("could not create role binding"))
		}
	}
	return nil
}

func (o *Install) provisionNamespace(ctx context.Context, namespace string) error {
	if err := o.provisionAllOperators(ctx, namespace); err != nil {
		return err
	}
	return nil
}

func (o *Install) provisionMonitoringInAllNamespaces(ctx context.Context) error {
	l := o.l.With("action", "monitoring")
	l.Info("Preparing k8s cluster for monitoring")
	namespaces := []string{everestNamespace}
	namespaces = append(namespaces, o.config.Namespaces...)
	for _, namespace := range namespaces {
		if err := o.kubeClient.ProvisionMonitoring(namespace); err != nil {
			return errors.Join(err, errors.New("could not provision monitoring configuration"))
		}

		l.Info("K8s cluster monitoring has been provisioned successfully")
		if err := o.resolveMonitoringInstanceName(ctx); err != nil {
			return err
		}
		o.l.Info("Deploying VMAgent to k8s cluster")
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

		// We retry for a bit since the MonitoringConfig may not be properly
		// deployed yet and we get a HTTP 500 in this case.
		err := wait.PollUntilContextTimeout(ctx, 3*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
			o.l.Debug("Trying to enable Kubernetes cluster monitoring")
			err := o.everestClient.SetKubernetesClusterMonitoring(ctx, client.KubernetesClusterMonitoring{
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

func (o *Install) resolveMonitoringInstanceName(ctx context.Context) error {
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

func (o *Install) createPMMMonitoringInstance(ctx context.Context, name, url, username, password string) error {
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

func (o *Install) configureEverestConnector() error {
	e, err := everestClient.NewProxiedEverest(o.kubeClient.Config(), everestNamespace)
	if err != nil {
		return err
	}
	o.everestClient = e
	return nil
}

// runWizard runs installation wizard.
func (o *Install) runWizard() error {
	if err := o.runEverestWizard(); err != nil {
		return err
	}

	if err := o.runMonitoringWizard(); err != nil {
		return err
	}

	return o.runInstallWizard()
}

func (o *Install) runEverestWizard() error {
	var namespaces string
	pNamespace := &survey.Input{
		Message: "Namespaces to deploy Everest to. Comma separated",
		Default: namespaces,
	}
	if err := survey.AskOne(pNamespace, &namespaces); err != nil {
		return err
	}
	o.config.Namespaces = strings.Split(namespaces, ",")
	return nil
}

func (o *Install) runMonitoringWizard() error {
	pMonitor := &survey.Confirm{
		Message: "Do you want to enable monitoring?",
		Default: o.config.Monitoring.Enable,
	}
	err := survey.AskOne(pMonitor, &o.config.Monitoring.Enable)
	if err != nil {
		return err
	}

	if o.config.Monitoring.Enable {
		if err := o.runMonitoringConfigWizard(); err != nil {
			return err
		}
	}

	return nil
}

func (o *Install) runMonitoringConfigWizard() error {
	if o.config.Monitoring.PMM == nil {
		o.config.Monitoring.PMM = &PMMConfig{}
	}

	if o.config.Monitoring.InstanceName == "" {
		if err := o.runMonitoringNewURLWizard(); err != nil {
			return err
		}
	}

	return nil
}

func (o *Install) runMonitoringNewURLWizard() error {
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

func (o *Install) runInstallWizard() error {
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

// createNamespace provisions a namespace for Everest.
func (o *Install) createNamespace(namespace string) error {
	o.l.Infof("Creating namespace %s", namespace)
	err := o.kubeClient.CreateNamespace(namespace)
	if err != nil {
		return errors.Join(err, errors.New("could not provision namespace"))
	}

	o.l.Infof("Namespace %s has been created", namespace)
	return nil
}

// provisionAllOperators provisions all configured operators to a k8s cluster.
func (o *Install) provisionAllOperators(ctx context.Context, namespace string) error {
	o.l.Info("Started provisioning the cluster")
	if err := o.provisionOperators(ctx, namespace); err != nil {
		return err
	}

	return nil
}

func (o *Install) provisionOLM(ctx context.Context) error {
	o.l.Info("Installing Operator Lifecycle Manager")
	if err := o.kubeClient.InstallOLMOperator(ctx, false); err != nil {
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

func (o *Install) provisionOperators(ctx context.Context, namespace string) error {
	deploymentsBefore, err := o.kubeClient.ListEngineDeploymentNames(ctx, everestNamespace)
	if err != nil {
		return err
	}
	g, gCtx := errgroup.WithContext(ctx)
	// We set the limit to 1 since operator installation
	// requires an update to the same installation plan which
	// results in race-conditions with a higher limit.
	// The limit can be removed after it's refactored.
	g.SetLimit(operatorInstallThreads)

	if o.config.Monitoring.Enable {
		g.Go(o.installOperator(gCtx, o.config.Channel.VictoriaMetrics, vmOperatorName, namespace))
	}

	if o.config.Operator.PXC {
		g.Go(o.installOperator(gCtx, o.config.Channel.PXC, pxcOperatorName, namespace))
	}
	if o.config.Operator.PSMDB {
		g.Go(o.installOperator(gCtx, o.config.Channel.PSMDB, psmdbOperatorName, namespace))
	}
	if o.config.Operator.PG {
		g.Go(o.installOperator(gCtx, o.config.Channel.PG, pgOperatorName, namespace))
	}
	if err := g.Wait(); err != nil {
		return err
	}

	deploymentsAfter, err := o.kubeClient.ListEngineDeploymentNames(ctx, namespace)
	if err != nil {
		return err
	}
	if len(deploymentsBefore) != 0 && len(deploymentsBefore) != len(deploymentsAfter) {
		return o.restartEverestOperatorPod(ctx)
	}
	return nil
}

func (o *Install) installOperator(ctx context.Context, channel, operatorName, namespace string) func() error {
	return func() error {
		// We check if the context has not been cancelled yet to return early
		if err := ctx.Err(); err != nil {
			o.l.Debugf("Cancelled %s operator installation due to context error: %s", operatorName, err)
			return err
		}

		o.l.Infof("Installing %s operator", operatorName)

		params := kubernetes.InstallOperatorRequest{
			Namespace:              namespace,
			Name:                   operatorName,
			OperatorGroup:          operatorGroup,
			CatalogSource:          catalogSource,
			CatalogSourceNamespace: catalogSourceNamespace,
			Channel:                channel,
			InstallPlanApproval:    v1alpha1.ApprovalManual,
		}
		if len(o.config.Namespaces) != 0 && operatorName == everestOperatorName {
			params.TargetNamespaces = o.config.Namespaces
			params.SubscriptionConfig = &v1alpha1.SubscriptionConfig{
				Env: []corev1.EnvVar{
					{
						Name:  everestWatchNamespacesEnvVar,
						Value: strings.Join(o.config.Namespaces, ","),
					},
				},
			}
		}

		if err := o.kubeClient.InstallOperator(ctx, params); err != nil {
			o.l.Errorf("failed installing %s operator", operatorName)
			return err
		}
		o.l.Infof("%s operator has been installed", operatorName)

		return nil
	}
}

func (o *Install) restartEverestOperatorPod(ctx context.Context) error {
	return o.kubeClient.RestartEverest(ctx, "everest-operator", everestNamespace)
}

func (o *Install) serviceAccountRolePolicyRules() []rbacv1.PolicyRule {
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
