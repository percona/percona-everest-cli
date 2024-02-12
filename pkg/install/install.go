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
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	goversion "github.com/hashicorp/go-version"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	versionpb "github.com/Percona-Lab/percona-version-service/versionpb"
	"github.com/percona/percona-everest-cli/pkg/kubernetes"
	"github.com/percona/percona-everest-cli/pkg/token"
	"github.com/percona/percona-everest-cli/pkg/version"
)

// Install implements the main logic for commands.
type Install struct {
	l *zap.SugaredLogger

	config     Config
	kubeClient *kubernetes.Kubernetes
}

const (
	everestBackendServiceName = "percona-everest-backend"
	everestOperatorName       = "everest-operator"
	pxcOperatorName           = "percona-xtradb-cluster-operator"
	psmdbOperatorName         = "percona-server-mongodb-operator"
	pgOperatorName            = "percona-postgresql-operator"
	vmOperatorName            = "victoriametrics-operator"
	operatorInstallThreads    = 1

	everestServiceAccount                   = "everest-admin"
	everestServiceAccountRole               = "everest-admin-role"
	everestServiceAccountRoleBinding        = "everest-admin-role-binding"
	everestServiceAccountClusterRoleBinding = "everest-admin-cluster-role-binding"

	everestOperatorChannel = "stable-v0"
	pxcOperatorChannel     = "stable-v1"
	psmdbOperatorChannel   = "stable-v1"
	pgOperatorChannel      = "stable-v2"
	vmOperatorChannel      = "stable-v0"

	// catalogSourceNamespace is the namespace where the catalog source is installed.
	catalogSourceNamespace = "olm"
	// catalogSource is the name of the catalog source.
	catalogSource = "everest-catalog"

	// systemOperatorGroup is the name of the system operator group.
	systemOperatorGroup = "everest-system"
	// monitoringOperatorGroup is the name of the monitoring operator group.
	monitoringOperatorGroup = "everest-monitoring"
	// dbsOperatorGroup is the name of the database operator group.
	dbsOperatorGroup = "everest-databases"

	// SystemNamespace is the namespace where everest is installed.
	SystemNamespace = "everest-system"
	// monitoringNamespace is the namespace where the monitoring stack is installed.
	monitoringNamespace = "everest-monitoring"
	// EverestMonitoringNamespaceEnvVar is the name of the environment variable that holds the monitoring namespace.
	EverestMonitoringNamespaceEnvVar = "MONITORING_NAMESPACE"
	// disableTelemetryEnvVar is the name of the environment variable that disables telemetry.
	disableTelemetryEnvVar = "DISABLE_TELEMETRY"
)

type (
	// Config stores configuration for the operators.
	Config struct {
		// Namespaces defines namespaces that everest can operate in.
		Namespaces []string `mapstructure:"namespace"`
		// SkipWizard skips wizard during installation.
		SkipWizard bool `mapstructure:"skip-wizard"`
		// KubeconfigPath is a path to a kubeconfig
		KubeconfigPath string `mapstructure:"kubeconfig"`
		// VersionMetadataURL stores hostname to retrieve version metadata information from.
		VersionMetadataURL string `mapstructure:"version-metadata-url"`

		Operator OperatorConfig
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
)

// NewInstall returns a new Install struct.
func NewInstall(c Config, l *zap.SugaredLogger) (*Install, error) {
	cli := &Install{
		config: c,
		l:      l.With("component", "install"),
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

	meta, err := version.Metadata(ctx, o.config.VersionMetadataURL)
	if err != nil {
		return err
	}

	latest, err := o.latestVersion(meta)
	if err != nil {
		return err
	}

	if err := o.provisionOLM(ctx, latest); err != nil {
		return err
	}

	if err := o.provisionMonitoringStack(ctx); err != nil {
		return err
	}

	// TODO: revisit - we need to install correct version based on metadata.
	if err := o.provisionDBNamespaces(ctx); err != nil {
		return err
	}

	// TODO: install correct version based on metadata.
	if err := o.provisionEverestOperator(ctx); err != nil {
		return err
	}

	if err := o.provisionEverest(ctx, latest); err != nil {
		return err
	}
	_, err = o.kubeClient.GetSecret(ctx, token.SecretName, SystemNamespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Join(err, errors.New("could not get the everest token secret"))
	}
	if err != nil && k8serrors.IsNotFound(err) {
		pwd, err := o.generateToken(ctx)
		if err != nil {
			return err
		}
		o.l.Info("\n" + pwd.String() + "\n\n")
	}

	return nil
}

func (o *Install) populateConfig() error {
	if !o.config.SkipWizard {
		if err := o.runWizard(); err != nil {
			return err
		}
	}

	if len(o.config.Namespaces) == 0 {
		return errors.New("namespace list is empty. Specify at least one namespace using the --namespace flag")
	}
	for _, ns := range o.config.Namespaces {
		if ns == SystemNamespace || ns == monitoringNamespace {
			return fmt.Errorf("'%s' namespace is reserved for Everest internals. Please specify another namespace", ns)
		}
	}

	return nil
}

func (o *Install) latestVersion(meta *versionpb.MetadataResponse) (*goversion.Version, error) {
	var latest *goversion.Version
	for _, v := range meta.Versions {
		ver, err := goversion.NewSemver(v.Version)
		if err != nil {
			o.l.Debugf("Could not parse version %s. Error: %s", v.Version, err)
			continue
		}

		if latest == nil || latest.GreaterThan(ver) {
			latest = ver
			continue
		}
	}

	if latest == nil {
		return nil, errors.New("could not determine the latest Everest version")
	}

	return latest, nil
}

func (o *Install) installVMOperator(ctx context.Context) error {
	o.l.Info("Creating operator group for everest")
	if err := o.kubeClient.CreateOperatorGroup(ctx, monitoringOperatorGroup, monitoringNamespace, []string{}); err != nil {
		return err
	}
	o.l.Infof("Installing %s operator", vmOperatorName)

	params := kubernetes.InstallOperatorRequest{
		Namespace:              monitoringNamespace,
		Name:                   vmOperatorName,
		OperatorGroup:          monitoringOperatorGroup,
		CatalogSource:          catalogSource,
		CatalogSourceNamespace: catalogSourceNamespace,
		Channel:                vmOperatorChannel,
		InstallPlanApproval:    v1alpha1.ApprovalManual,
	}

	if err := o.kubeClient.InstallOperator(ctx, params); err != nil {
		o.l.Errorf("failed installing %s operator", vmOperatorName)
		return err
	}
	o.l.Infof("%s operator has been installed", vmOperatorName)
	return nil
}

func (o *Install) provisionMonitoringStack(ctx context.Context) error {
	l := o.l.With("action", "monitoring")
	if err := o.createNamespace(monitoringNamespace); err != nil {
		return err
	}

	l.Info("Preparing k8s cluster for monitoring")
	// TODO: shall we grab VM operator version from metadata?
	if err := o.installVMOperator(ctx); err != nil {
		return err
	}
	if err := o.kubeClient.ProvisionMonitoring(monitoringNamespace); err != nil {
		return errors.Join(err, errors.New("could not provision monitoring configuration"))
	}

	l.Info("K8s cluster monitoring has been provisioned successfully")
	return nil
}

func (o *Install) provisionEverestOperator(ctx context.Context) error {
	if err := o.createNamespace(SystemNamespace); err != nil {
		return err
	}

	o.l.Info("Creating operator group for everest")
	if err := o.kubeClient.CreateOperatorGroup(ctx, systemOperatorGroup, SystemNamespace, o.config.Namespaces); err != nil {
		return err
	}

	if err := o.installOperator(ctx, everestOperatorChannel, everestOperatorName, SystemNamespace)(); err != nil {
		return err
	}

	return nil
}

func (o *Install) provisionEverest(ctx context.Context, v *goversion.Version) error {
	d, err := o.kubeClient.GetDeployment(ctx, kubernetes.PerconaEverestDeploymentName, SystemNamespace)
	var everestExists bool
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if d != nil && d.Name == kubernetes.PerconaEverestDeploymentName {
		everestExists = true
	}

	if !everestExists {
		o.l.Info(fmt.Sprintf("Deploying Everest to %s", SystemNamespace))
		if err = o.kubeClient.InstallEverest(ctx, SystemNamespace, v); err != nil {
			return err
		}
	} else {
		// TODO: revisit - we shall probably not restart but offer upgrade.
		o.l.Info("Restarting Everest")
		if err := o.kubeClient.RestartEverest(ctx, everestOperatorName, SystemNamespace); err != nil {
			return err
		}
		if err := o.kubeClient.RestartEverest(ctx, everestBackendServiceName, SystemNamespace); err != nil {
			return err
		}
	}

	// TODO: get from Everest, not cli.
	o.l.Info("Updating cluster role bindings for everest-admin")
	if err := o.kubeClient.UpdateClusterRoleBinding(ctx, everestServiceAccountClusterRoleBinding, o.config.Namespaces); err != nil {
		return err
	}

	return nil
}

func (o *Install) provisionDBNamespaces(ctx context.Context) error {
	for _, namespace := range o.config.Namespaces {
		namespace := namespace
		if err := o.createNamespace(namespace); err != nil {
			return err
		}
		if err := o.kubeClient.CreateOperatorGroup(ctx, dbsOperatorGroup, namespace, []string{}); err != nil {
			return err
		}

		o.l.Infof("Installing operators into %s namespace", namespace)
		if err := o.provisionOperators(ctx, namespace); err != nil {
			return err
		}
		o.l.Info("Creating role for the Everest service account")
		// TODO: this shall come from Everest, not cli.
		err := o.kubeClient.CreateRole(namespace, everestServiceAccountRole, o.serviceAccountRolePolicyRules())
		if err != nil {
			return errors.Join(err, errors.New("could not create role"))
		}

		o.l.Info("Binding role to the Everest Service account")
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

// runWizard runs installation wizard.
func (o *Install) runWizard() error {
	if err := o.runEverestWizard(); err != nil {
		return err
	}

	return o.runInstallWizard()
}

func (o *Install) runEverestWizard() error {
	var namespaces string
	pNamespace := &survey.Input{
		Message: "Namespaces managed by Everest (comma separated)",
		Default: namespaces,
	}
	if err := survey.AskOne(pNamespace, &namespaces); err != nil {
		return err
	}

	nsList := strings.Split(namespaces, ",")
	for _, ns := range nsList {
		ns = strings.TrimSpace(ns)
		if ns == "" {
			continue
		}

		if ns == SystemNamespace {
			return fmt.Errorf("'%s' namespace is reserved for Everest internals. Please specify another namespace", ns)
		}

		o.config.Namespaces = append(o.config.Namespaces, ns)
	}

	if len(o.config.Namespaces) == 0 {
		return errors.New("namespace list is empty. Specify at least one namespace")
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

func (o *Install) provisionOLM(ctx context.Context, v *goversion.Version) error {
	o.l.Info("Installing Operator Lifecycle Manager")
	// TODO: do we upgrade OLM if the version is too old?
	if err := o.kubeClient.InstallOLMOperator(ctx, false); err != nil {
		o.l.Error("failed installing OLM")
		return err
	}
	o.l.Info("OLM has been installed")
	o.l.Info("Installing Percona OLM Catalog")

	if err := o.kubeClient.InstallPerconaCatalog(ctx, v); err != nil {
		o.l.Errorf("failed installing OLM catalog: %v", err)
		return err
	}
	o.l.Info("Percona OLM Catalog has been installed")

	return nil
}

func (o *Install) provisionOperators(ctx context.Context, namespace string) error {
	g, gCtx := errgroup.WithContext(ctx)
	// We set the limit to 1 since operator installation
	// requires an update to the same installation plan which
	// results in race-conditions with a higher limit.
	// The limit can be removed after it's refactored.
	g.SetLimit(operatorInstallThreads)

	if o.config.Operator.PXC {
		g.Go(o.installOperator(gCtx, pxcOperatorChannel, pxcOperatorName, namespace))
	}
	if o.config.Operator.PSMDB {
		g.Go(o.installOperator(gCtx, psmdbOperatorChannel, psmdbOperatorName, namespace))
	}
	if o.config.Operator.PG {
		g.Go(o.installOperator(gCtx, pgOperatorChannel, pgOperatorName, namespace))
	}
	if err := g.Wait(); err != nil {
		return err
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

		disableTelemetry, ok := os.LookupEnv(disableTelemetryEnvVar)
		if !ok || disableTelemetry != "true" {
			disableTelemetry = "false"
		}

		params := kubernetes.InstallOperatorRequest{
			Namespace:              namespace,
			Name:                   operatorName,
			OperatorGroup:          systemOperatorGroup,
			CatalogSource:          catalogSource,
			CatalogSourceNamespace: catalogSourceNamespace,
			Channel:                channel,
			InstallPlanApproval:    v1alpha1.ApprovalManual,
			SubscriptionConfig: &v1alpha1.SubscriptionConfig{
				Env: []corev1.EnvVar{
					{
						Name:  disableTelemetryEnvVar,
						Value: disableTelemetry,
					},
				},
			},
		}
		if operatorName == everestOperatorName {
			params.TargetNamespaces = o.config.Namespaces
			params.SubscriptionConfig.Env = append(params.SubscriptionConfig.Env, []corev1.EnvVar{
				{
					Name:  EverestMonitoringNamespaceEnvVar,
					Value: monitoringNamespace,
				},
				{
					Name:  kubernetes.EverestDBNamespacesEnvVar,
					Value: strings.Join(o.config.Namespaces, ","),
				},
			}...)
		}

		if err := o.kubeClient.InstallOperator(ctx, params); err != nil {
			o.l.Errorf("failed installing %s operator", operatorName)
			return err
		}
		o.l.Infof("%s operator has been installed", operatorName)

		return nil
	}
}

func (o *Install) serviceAccountRolePolicyRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{"everest.percona.com"},
			Resources: []string{"databaseclusters"},
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

func (o *Install) generateToken(ctx context.Context) (*token.ResetResponse, error) {
	o.l.Info("Creating token for Everest")

	r, err := token.NewReset(
		token.ResetConfig{
			KubeconfigPath: o.config.KubeconfigPath,
			Namespace:      SystemNamespace,
		},
		o.l,
	)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not initialize reset token"))
	}

	res, err := r.Run(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not create token"))
	}

	return res, nil
}
