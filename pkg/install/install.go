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

	"github.com/AlecAivazis/survey/v2"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/percona/percona-everest-cli/pkg/kubernetes"
	"github.com/percona/percona-everest-cli/pkg/password"
)

// Install implements the main logic for commands.
type Install struct {
	l *zap.SugaredLogger

	config     Config
	kubeClient *kubernetes.Kubernetes
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
)

var errAlreadyGenerated = errors.New("token is already generated")

type (
	// Config stores configuration for the operators.
	Config struct {
		// Name of the Kubernetes Cluster
		Name string
		// Namespace defines the namespace operators shall be installed to.
		Namespace string
		// SkipWizard skips wizard during installation.
		SkipWizard bool `mapstructure:"skip-wizard"`
		// KubeconfigPath is a path to a kubeconfig
		KubeconfigPath string `mapstructure:"kubeconfig"`

		Channel  ChannelConfig
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
	if err := o.provisionNamespace(); err != nil {
		return err
	}
	if err := o.performProvisioning(ctx); err != nil {
		return err
	}
	_, err := o.kubeClient.GetSecret(ctx, password.SecretName, o.config.Namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Join(err, errors.New("could not get the everest token secret"))
	}
	if err != nil && k8serrors.IsNotFound(err) {
		pwd, err := o.generatePassword(ctx)
		if err != nil {
			return err
		}
		o.l.Info(pwd)
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

func (o *Install) performProvisioning(ctx context.Context) error {
	if err := o.provisionAllOperators(ctx); err != nil {
		return err
	}
	d, err := o.kubeClient.GetDeployment(ctx, kubernetes.PerconaEverestDeploymentName, o.config.Namespace)
	var everestExists bool
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if d != nil && d.Name == kubernetes.PerconaEverestDeploymentName {
		everestExists = true
	}

	if !everestExists {
		o.l.Info(fmt.Sprintf("Deploying Everest to %s", o.config.Namespace))
		err = o.kubeClient.InstallEverest(ctx, o.config.Namespace)
		if err != nil {
			return err
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
	pNamespace := &survey.Input{
		Message: "Namespace to deploy Everest to",
		Default: o.config.Namespace,
	}
	return survey.AskOne(pNamespace, &o.config.Namespace)
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

// provisionNamespace provisions a namespace for Everest.
func (o *Install) provisionNamespace() error {
	o.l.Infof("Creating namespace %s", o.config.Namespace)
	err := o.kubeClient.CreateNamespace(o.config.Namespace)
	if err != nil {
		return errors.Join(err, errors.New("could not provision namespace"))
	}

	o.l.Infof("Namespace %s has been created", o.config.Namespace)
	return nil
}

// provisionAllOperators provisions all configured operators to a k8s cluster.
func (o *Install) provisionAllOperators(ctx context.Context) error {
	o.l.Info("Started provisioning the cluster")

	if err := o.provisionOLM(ctx); err != nil {
		return err
	}

	if err := o.provisionInstall(ctx); err != nil {
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

func (o *Install) provisionInstall(ctx context.Context) error {
	deploymentsBefore, err := o.kubeClient.ListEngineDeploymentNames(ctx, o.config.Namespace)
	if err != nil {
		return err
	}
	g, gCtx := errgroup.WithContext(ctx)
	// We set the limit to 1 since operator installation
	// requires an update to the same installation plan which
	// results in race-conditions with a higher limit.
	// The limit can be removed after it's refactored.
	g.SetLimit(operatorInstallThreads)

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

	if err := o.installOperator(ctx, o.config.Channel.Everest, everestOperatorName)(); err != nil {
		return err
	}
	deploymentsAfter, err := o.kubeClient.ListEngineDeploymentNames(ctx, o.config.Namespace)
	if err != nil {
		return err
	}
	if len(deploymentsBefore) != 0 && len(deploymentsBefore) != len(deploymentsAfter) {
		return o.restartEverestOperatorPod(ctx)
	}
	return nil
}

func (o *Install) installOperator(ctx context.Context, channel, operatorName string) func() error {
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

func (o *Install) generatePassword(ctx context.Context) (*password.ResetResponse, error) {

	o.l.Info("Creating password for Everest")

	r, err := password.NewReset(
		password.ResetConfig{
			KubeconfigPath: o.config.KubeconfigPath,
			Namespace:      o.config.Namespace,
		},
		o.l,
	)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not initialize reset password"))
	}

	res, err := r.Run(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not create password"))
	}

	o.l.Debug(res)

	return res, nil
}

func (o *Install) restartEverestOperatorPod(ctx context.Context) error {
	return o.kubeClient.RestartEverest(ctx, "everest-operator", o.config.Namespace)
}
