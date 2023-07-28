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

	everestClient "github.com/percona/percona-everest-cli/pkg/everest/client"
	"github.com/percona/percona-everest-cli/pkg/kubernetes"
)

// Operators implements the main logic for commands.
type Operators struct {
	l *logrus.Entry

	config        OperatorsConfig
	everestClient everestClientConnector
	kubeClient    *kubernetes.Kubernetes

	// apiKeySecretID stores name of a secret with PMM API key.
	apiKeySecretID string
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
		// InstanceID stores PMM instance ID from Everest.
		// If provided, Endpoint, Username and Password are ignored.
		InstanceID string `mapstructure:"instance-id"`
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

type pmmErrorMessage struct {
	Message string `json:"message"`
}

const secretNameTemplate = "everest-%s"

// NewOperators returns a new Operators struct.
func NewOperators(c OperatorsConfig) (*Operators, error) {
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

// Run runs the operators installation process.
func (o *Operators) Run(ctx context.Context) error {
	if !o.config.SkipWizard {
		if err := o.runWizard(ctx); err != nil {
			return err
		}
	}

	if err := o.validateConfig(ctx); err != nil {
		return err
	}

	if o.everestClient == nil {
		if err := o.configureEverestConnector(); err != nil {
			return err
		}
	}

	if err := o.provisionNamespace(); err != nil {
		return err
	}
	if err := o.provisionAllOperators(ctx); err != nil {
		return err
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return o.connectToEverest(gCtx)
	})
	g.Go(func() error {
		return o.createEverestBackupStorage(gCtx)
	})

	return g.Wait()
}

func (o *Operators) validateConfig(ctx context.Context) error {
	if o.config.Monitoring.Enable && o.apiKeySecretID == "" {
		if o.config.Monitoring.PMM.InstanceID == "" {
			return errors.New("--monitoring.pmm.instance-id cannot be empty if monitoring is enabled")
		}

		if err := o.setPMMAPIKeySecretIDFromInstanceID(ctx); err != nil {
			return errors.Wrap(err, "could not retrieve PMM instance by its ID from Everest")
		}
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

	if o.config.Monitoring.PMM.InstanceID == "" {
		if err := o.runMonitoringURLWizard(ctx); err != nil {
			return err
		}
	} else {
		if err := o.setPMMAPIKeySecretIDFromInstanceID(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (o *Operators) runMonitoringURLWizard(ctx context.Context) error {
	instances, err := o.everestClient.ListPMMInstances(ctx)
	if err != nil {
		return errors.Wrap(err, "could not retrieve list of PMM instances")
	}

	if len(instances) == 0 {
		return o.runMonitoringNewURLWizard()
	}

	opts := make([]string, 0, len(instances)+1)
	for _, i := range instances {
		opts = append(opts, i.Url)
	}
	opts = append(opts, "Add new PMM instance")

	pInstance := &survey.Select{
		Message: "Select PMM instance:",
		Options: opts,
	}
	ix := 0
	if err := survey.AskOne(pInstance, &ix); err != nil {
		return err
	}

	if ix > len(instances)-1 {
		return o.runMonitoringNewURLWizard()
	}

	pmm := instances[ix]
	o.config.Monitoring.PMM.Endpoint = pmm.Url
	o.apiKeySecretID = pmm.ApiKeySecretId

	return nil
}

func (o *Operators) runMonitoringNewURLWizard() error {
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
	return survey.AskOne(
		pPass,
		&o.config.Monitoring.PMM.Password,
		survey.WithValidator(survey.Required),
	)
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

func (o *Operators) setPMMAPIKeySecretIDFromInstanceID(ctx context.Context) error {
	pmm, err := o.everestClient.GetPMMInstance(ctx, o.config.Monitoring.PMM.InstanceID)
	if err != nil {
		return err
	}

	o.apiKeySecretID = pmm.ApiKeySecretId

	return nil
}

// provisionNamespace provisions a namespace for Everest.
func (o *Operators) provisionNamespace() error {
	o.l.Infof("Creating namespace %s", o.config.Namespace)
	err := o.kubeClient.CreateNamespace(o.config.Namespace)
	if err != nil {
		return errors.Wrap(err, "could not provision namespace")
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
		if err := o.provisionPMMMonitoring(ctx); err != nil {
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

func (o *Operators) provisionPMMMonitoring(ctx context.Context) error {
	l := o.l.WithField("action", "PMM")
	l.Info("Setting up PMM monitoring")

	if o.apiKeySecretID == "" {
		if err := o.provisionNewPMMInstance(ctx, l); err != nil {
			return errors.Wrap(err, "could not create a new PMM instance")
		}
	}

	l.Debugf("Using API key secret ID %s", o.apiKeySecretID)

	l.Info("Provisioning monitoring in k8s cluster")
	err := o.kubeClient.ProvisionMonitoring(
		o.config.Namespace,
		o.apiKeySecretID,
		o.config.Monitoring.PMM.Endpoint,
	)
	if err != nil {
		return errors.Wrap(err, "could not provision PMM Monitoring")
	}

	l.Info("PMM Monitoring has been provisioned successfully")

	return nil
}

func (o *Operators) provisionNewPMMInstance(ctx context.Context, l *logrus.Entry) error {
	if o.config.Monitoring.PMM.Endpoint == "" || o.config.Monitoring.PMM.Username == "" {
		return errors.New("PMM endpoint or username is empty")
	}

	account := fmt.Sprintf("everest-pmm-%s", uuid.NewString())
	l.Info("Creating a new API key in PMM")

	apiKey, err := o.createPMMApiKey(ctx, account, "")
	if err != nil {
		return errors.Wrap(err, "could not create PMM API key")
	}

	l.Infof("New API key with name %q has been created", account)

	l.Info("Creating PMM instance in Everest")
	pmm, err := o.everestClient.CreatePMMInstance(ctx, client.PMMInstanceCreateParams{
		Url:    o.config.Monitoring.PMM.Endpoint,
		ApiKey: apiKey,
	})
	if err != nil {
		return errors.Wrap(err, "could not create PMM instance in Everest")
	}
	l.Infof("PMM instance %s has been created in Everest", *pmm.Id)

	if pmm.Id == nil || *pmm.Id == "" {
		return errors.New("PMM instance ID is empty")
	}

	l.Info("Creating secret in Kubernetes")
	o.apiKeySecretID = fmt.Sprintf(secretNameTemplate, *pmm.Id)
	err = o.kubeClient.CreatePMMSecret(o.config.Namespace, o.apiKeySecretID, map[string][]byte{
		"username": []byte("api_key"),
		"password": []byte(apiKey),
	})
	if err != nil {
		return errors.Wrap(err, "could not create secret in Kubernetes")
	}
	l.Infof("Secret %s has been created", o.apiKeySecretID)

	return nil
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
		Namespace:  &o.config.Namespace,
	})
	if err != nil {
		return errors.Wrap(err, "could not register a new Kubernetes cluster with Everest")
	}

	o.l.Info("Connected Kubernetes cluster to Everest")

	return nil
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
		return errors.Wrap(err, "could not create a new backup storage in Everest")
	}

	o.l.Info("Created a new backup storage in Everest")

	return nil
}

func (o *Operators) prepareServiceAccount() error {
	o.l.Info("Creating service account for Everest")
	if err := o.kubeClient.CreateServiceAccount(everestServiceAccount, o.config.Namespace); err != nil {
		return errors.Wrap(err, "could not create service account")
	}

	o.l.Info("Creating role for Everest service account")
	err := o.kubeClient.CreateRole(o.config.Namespace, everestServiceAccountRole, o.serviceAccountPolicyRules())
	if err != nil {
		return errors.Wrap(err, "could not create role")
	}

	o.l.Info("Binding role to Everest Service account")
	err = o.kubeClient.CreateRoleBinding(
		o.config.Namespace,
		everestServiceAccountRoleBinding,
		everestServiceAccountRole,
		everestServiceAccount,
	)

	return errors.Wrap(err, "could not create cluster role binding")
}

func (o *Operators) serviceAccountPolicyRules() []rbacv1.PolicyRule {
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
			Resources: []string{"objectstorages"},
			Verbs:     []string{"*"},
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
		return "", errors.Wrap(err, "could not get token from secret for a service account")
	}

	return o.kubeClient.GenerateKubeConfigWithToken(everestServiceAccount, secret)
}

func (o *Operators) createPMMApiKey(ctx context.Context, name string, token string) (string, error) {
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

	if resp.StatusCode >= http.StatusBadRequest {
		var pmmErr *pmmErrorMessage
		if err := json.Unmarshal(data, &pmmErr); err != nil {
			return "", errors.Wrapf(err, "PMM returned an unknown error. HTTP status code %d", resp.StatusCode)
		}
		return "", errors.Errorf("PMM returned an error with message: %s", pmmErr.Message)
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
