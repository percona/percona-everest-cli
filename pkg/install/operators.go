// Package install holds the main logic for installation commands.
package install

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/uuid"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/percona/percona-everest-backend/client"
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
	config        *OperatorsConfig
	everestClient everestClientConnector
	kubeClient    *kubernetes.Kubernetes
	l             *logrus.Entry
}

const (
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
		Channel ChannelConfig `mapstructure:"channel"`
		// EnableBackup is true if backup shall be enabled.
		EnableBackup bool          `mapstructure:"enable_backup"`
		Everest      EverestConfig `mapstructure:"everest"`
		// InstallOLM is true if OLM shall be installed.
		InstallOLM bool `mapstructure:"install_olm"`
		// KubeconfigPath is a path to a kubeconfig
		KubeconfigPath string           `mapstructure:"kubeconfig"`
		Monitoring     MonitoringConfig `mapstructure:"monitoring"`
		// Name of the Kubernetes Cluster
		Name     string         `mapstructure:"name"`
		Operator OperatorConfig `mapstructure:"operator"`
	}

	// EverestConfig stores config for Everest.
	EverestConfig struct {
		// Endpoint stores URL to Everest.
		Endpoint string `mapstructure:"endpoint"`
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
		// Namespace defines the namespace operators shall be installed to.
		Namespace string `mapstructure:"namespace"`
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
func NewOperators(c *OperatorsConfig, everestClient everestClientConnector) (*Operators, error) {
	if c == nil {
		panic("OperatorsConfig is required")
	}

	cli := &Operators{
		config:        c,
		everestClient: everestClient,
		l:             logrus.WithField("component", "install/operators"),
	}

	k, err := kubernetes.New(c.KubeconfigPath, cli.l)
	if err != nil {
		return nil, err
	}
	cli.kubeClient = k
	return cli, nil
}

// Run runs the operators installation process.
func (o *Operators) Run(ctx context.Context) error {
	if err := o.runWizard(); err != nil {
		return err
	}
	if err := o.provisionNamespace(); err != nil {
		return err
	}
	if err := o.provisionAllOperators(ctx); err != nil {
		return err
	}

	return o.connectToEverest(ctx)
}

// runWizard runs installation wizard.
func (o *Operators) runWizard() error {
	if err := o.runEverestWizard(); err != nil {
		return err
	}

	if err := o.runMonitoringWizard(); err != nil {
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

// provisionNamespace provisions a namespace for Everest.
func (o *Operators) provisionNamespace() error {
	o.l.Infof("Creating namespace %s", o.config.Operator.Namespace)
	err := o.kubeClient.CreateNamespace(o.config.Operator.Namespace)
	if err != nil {
		return errors.Wrap(err, "could not provision namespace")
	}

	o.l.Infof("Namespace %s has been created", o.config.Operator.Namespace)
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

	if o.config.Monitoring.Enabled {
		o.l.Info("Started setting up monitoring")
		// if err := c.provisionPMMMonitoring(); err != nil {
		// 	return err
		// }
		o.l.Info("Monitoring using PMM has been provisioned")
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
		// We check if the context has not been cancelled yet to return early
		if err := ctx.Err(); err != nil {
			o.l.Debugf("Cancelled %s operator installation due to context error: %s", operatorName, err)
			return err
		}

		o.l.Infof("Installing %s operator", operatorName)

		params := kubernetes.InstallOperatorRequest{
			Namespace:              o.config.Operator.Namespace,
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
	account := fmt.Sprintf("everest-service-account-%s", uuid.NewString())
	o.l.Info("Creating a new service account in PMM")
	token, err := o.provisionPMM(ctx, account)
	if err != nil {
		return err
	}
	o.l.Info("New token has been generated")
	o.l.Info("Started provisioning monitoring in k8s cluster")
	err = o.kubeClient.ProvisionMonitoring(account, token, o.config.Monitoring.PMM.Endpoint)
	if err != nil {
		o.l.Error("failed provisioning monitoring")
		return err
	}

	return nil
}

//nolint:unused
func (o *Operators) provisionPMM(ctx context.Context, account string) (string, error) {
	token, err := o.createAdminToken(ctx, account, "")
	return token, err
}

// connectToEverest connects the k8s cluster to Everest.
func (o *Operators) connectToEverest(ctx context.Context) error {
	if err := o.prepareServiceAccount(); err != nil {
		return errors.Wrap(err, "could not prepare a service account")
	}

	o.l.Info("Generating kubeconfig")
	kubeconfig, err := o.getServiceAccountKubeConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get a new kubeconfig file for a service account")
	}

	o.l.Info("Connecting your Kubernetes cluster to Everest")

	_, err = o.everestClient.RegisterKubernetesCluster(ctx, client.CreateKubernetesClusterParams{
		Kubeconfig: base64.StdEncoding.EncodeToString([]byte(kubeconfig)),
		Name:       o.config.Name,
	})
	if err != nil {
		return errors.Wrap(err, "could not register a new Kubernetes cluster with Everest")
	}

	return nil
}

func (o *Operators) prepareServiceAccount() error {
	o.l.Info("Creating service account for Everest")
	if err := o.kubeClient.CreateServiceAccount(everestServiceAccount); err != nil {
		return errors.Wrap(err, "could not create service account")
	}

	o.l.Info("Creating role for Everest service account")
	err := o.kubeClient.CreateRole(o.config.Operator.Namespace, everestServiceAccountRole, []rbacv1.PolicyRule{
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
		o.config.Operator.Namespace,
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
func (o *Operators) createAdminToken(ctx context.Context, name string, token string) (string, error) {
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
	client := &http.Client{}
	resp, err := client.Do(req)
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
