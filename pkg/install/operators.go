// Package install holds the main logic for installation commands.
package install

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/uuid"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/percona/percona-everest-cli/pkg/kubernetes"
)

// Operators implements the main logic for commands.
type Operators struct {
	config     *OperatorsConfig
	kubeClient *kubernetes.Kubernetes
	l          *logrus.Entry
}

const (
	namespace                        = "default"
	catalogSourceNamespace           = "olm"
	operatorGroup                    = "percona-operators-group"
	catalogSource                    = "percona-dbaas-catalog"
	dbaasOperatorName                = "dbaas-operator"
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
		Channel        ChannelConfig    `mapstructure:"channel"`
		EnableBackup   bool             `mapstructure:"enable_backup"`
		InstallOLM     bool             `mapstructure:"install_olm"`
		KubeconfigPath string           `mapstructure:"kubeconfig"`
		Monitoring     MonitoringConfig `mapstructure:"monitoring"`
		Operator       OperatorConfig   `mapstructure:"operator"`
	}

	// MonitoringConfig stores configuration for monitoring.
	MonitoringConfig struct {
		// Enabled is true if monitoring shall be enabled.
		Enabled bool `mapstructure:"enabled"`
		// Type stores the type of monitoring to be used.
		Type MonitoringType `mapstructure:"type"`
		// PMM stores configuration for PMM monitoring type.
		PMM *PMMConfig `mapstructure:"pmm"`
	}

	// OperatorConfig identifies which operators shall be installed.
	OperatorConfig struct {
		// PG stores if PostgresSQL shall be installed.
		PG bool `mapstructure:"postgresql"`
		// PSMDB stores if MongoDB shall be installed.
		PSMDB bool `mapstructure:"mongodb"`
		// PXC stores if XtraDB Cluster shall be installed.
		PXC bool `mapstructure:"xtradb_cluster"`
	}

	// PMMConfig stores configuration for PMM monitoring type.
	PMMConfig struct {
		// Endpoint stores URL to PMM.
		Endpoint string `mapstructure:"endpoint"`
		// Username stores username for authentication against PMM.
		Username string `mapstructure:"username"`
		// Password stores password for authentication against PMM.
		Password string `mapstructure:"password"`
	}

	// ChannelConfig stores configuration for operator channels.
	ChannelConfig struct {
		// Everest stores channel for Everest.
		Everest string `mapstructure:"everest"`
		// PG stores channel for PostgreSQL.
		PG string `mapstructure:"postgresql"`
		// PSMDB stores channel for MongoDB.
		PSMDB string `mapstructure:"mongodb"`
		// PXC stores channel for xtradb cluster.
		PXC string `mapstructure:"xtradb_cluster"`
		// VictoriaMetrics stores channel for VictoriaMetrics.
		VictoriaMetrics string `mapstructure:"victoria_metrics"`
	}
)

// NewOperators returns a new Operators struct.
func NewOperators(c *OperatorsConfig) (*Operators, error) {
	if c == nil {
		panic("OperatorsConfig is required")
	}

	cli := &Operators{
		config: c,
		l:      logrus.WithField("component", "install/operators"),
	}

	k, err := kubernetes.New(c.KubeconfigPath, cli.l)
	if err != nil {
		return nil, err
	}
	cli.kubeClient = k
	return cli, nil
}

// RunWizard runs installation wizard.
func (o *Operators) RunWizard() error {
	if err := o.runMonitoringWizard(); err != nil {
		return err
	}

	if err := o.runBackupWizard(); err != nil {
		return err
	}

	return o.runOperatorsWizard()
}

func (o *Operators) runMonitoringWizard() error {
	pMonitor := &survey.Confirm{
		Message: "Do you want to enable monitoring?",
		Default: o.config.Monitoring.Enabled,
	}
	err := survey.AskOne(pMonitor, &o.config.Monitoring.Enabled)
	if err != nil {
		return err
	}

	if o.config.Monitoring.Enabled {
		pURL := &survey.Input{
			Message: "URL Endpoint",
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
	}

	return nil
}

func (o *Operators) runBackupWizard() error {
	pBackup := &survey.Confirm{
		Message: "Do you want to enable backups?",
		Default: o.config.EnableBackup,
	}
	return survey.AskOne(pBackup, &o.config.EnableBackup)
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

// ProvisionOperators provisions all configured operators to a k8s cluster.
func (o *Operators) ProvisionOperators() error {
	o.l.Info("Started provisioning the cluster")
	ctx := context.TODO()

	if err := o.provisionOLM(ctx); err != nil {
		return err
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return o.provisionOperators(gCtx)
	})

	if o.config.Monitoring.Enabled {
		g.Go(func() error {
			return o.provisionPMMMonitoring(gCtx)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func (o *Operators) provisionOLM(ctx context.Context) error {
	if o.config.InstallOLM {
		o.l.Info("Installing Operator Lifecycle Manager")
		if err := o.kubeClient.InstallOLMOperator(ctx); err != nil {
			o.l.Error("failed installing OLM")
			return err
		}
	}
	o.l.Info("OLM has been installed")

	return nil
}

func (o *Operators) provisionOperators(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)
	// We set the limit to 1 since operator installation
	// requires an update to the same installation plan which
	// results in race-conditions with a higher limit.
	// The limit can be removed after it's refactored.
	g.SetLimit(operatorInstallThreads)

	if o.config.Monitoring.Enabled {
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

	return o.installOperator(ctx, o.config.Channel.Everest, dbaasOperatorName)()
}

func (o *Operators) installOperator(ctx context.Context, channel, operatorName string) func() error {
	return func() error {
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

		if err := o.kubeClient.InstallOperator(ctx, params); err != nil {
			o.l.Errorf("failed installing %s operator", operatorName)
			return err
		}
		o.l.Infof("%s operator has been installed", operatorName)

		return nil
	}
}

//nolint:unused
func (o *Operators) provisionPMMMonitoring(ctx context.Context) error {
	l := o.l.WithField("action", "PMM")
	l.Info("Setting up PMM monitoring")

	account := fmt.Sprintf("everest-service-account-%s", uuid.NewString())
	l.Info("Creating a new service account in PMM")
	token, err := o.provisionPMM(ctx, account)
	if err != nil {
		return err
	}
	l.Info("New token has been generated")
	l.Info("Provisioning monitoring in k8s cluster")
	err = o.kubeClient.ProvisionMonitoring(account, token, o.config.Monitoring.PMM.Endpoint)
	if err != nil {
		l.Error("failed provisioning monitoring")
		return errors.Wrap(err, "could not provision PMM Monitoring")
	}

	l.Info("PMM Monitoring provisioned successfully")

	return nil
}

//nolint:unused
func (o *Operators) provisionPMM(ctx context.Context, account string) (string, error) {
	token, err := o.createPMMAdminToken(ctx, account, "")
	return token, err
}

// ConnectToEverest connects the k8s cluster to Everest.
func (o *Operators) ConnectToEverest(ctx context.Context) error {
	if err := o.prepareServiceAccount(namespace); err != nil {
		return errors.Wrap(err, "could not prepare a service account")
	}

	o.l.Info("Generating kubeconfig")
	_, err := o.getServiceAccountKubeConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get a new kubeconfig file for a service account")
	}

	o.l.Info("Connecting your Kubernetes cluster to Everest")

	return nil
}

func (o *Operators) prepareServiceAccount(namespace string) error {
	o.l.Info("Creating service account for Everest")
	if err := o.kubeClient.CreateServiceAccount(everestServiceAccount); err != nil {
		return errors.Wrap(err, "could not create service account")
	}

	o.l.Info("Creating role for Everest service account")
	err := o.kubeClient.CreateRole(namespace, everestServiceAccountRole, []rbacv1.PolicyRule{
		{
			APIGroups: []string{"dbaas.percona.com"},
			Resources: []string{"databaseclusters", "databaseclusterrestores"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{"dbaas.percona.com"},
			Resources: []string{"databaseengines"},
			Verbs:     []string{"get", "list"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"create", "get"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"storageclasses"},
			Verbs:     []string{"get", "list"},
		},
	})
	if err != nil {
		return errors.Wrap(err, "could not create role")
	}

	o.l.Info("Binding role to Everest Service account")
	err = o.kubeClient.CreateRoleBinding(
		namespace,
		everestServiceAccountRoleBinding,
		everestServiceAccountRole,
		everestServiceAccount,
	)

	return errors.Wrap(err, "could not create cluster role binding")
}

func (o *Operators) getServiceAccountKubeConfig(ctx context.Context) (string, error) {
	// Create token secret
	err := o.kubeClient.CreateServiceAccountToken(everestServiceAccount, everestServiceAccountTokenSecret)
	if err != nil {
		return "", err
	}

	var secret *corev1.Secret
	checkSecretData := func(ctx context.Context) (bool, error) {
		o.l.Debugf("Getting secret for %s", everestServiceAccountTokenSecret)
		s, err := o.kubeClient.GetSecret(ctx, everestServiceAccountTokenSecret)
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
		return "", errors.Wrap(err, "could not get token from secret for a service account")
	}

	return o.kubeClient.GenerateKubeConfigWithToken(everestServiceAccount, secret)
}

//nolint:unused
func (o *Operators) createPMMAdminToken(ctx context.Context, name string, token string) (string, error) {
	apiKey := map[string]string{
		"name": name,
		"role": "Admin",
	}
	b, err := json.Marshal(apiKey)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/graph/api/auth/keys", o.config.Monitoring.PMM.Endpoint),
		bytes.NewReader(b),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if token == "" {
		req.SetBasicAuth(o.config.Monitoring.PMM.Username, o.config.Monitoring.PMM.Password)
	} else {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close() //nolint:errcheck
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return "", err
	}
	key, ok := m["key"].(string)
	if !ok {
		return "", errors.New("cannot unmarshal key in createAdminToken")
	}

	return key, nil
}
