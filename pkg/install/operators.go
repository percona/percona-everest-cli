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
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/uuid"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/percona/percona-everest-cli/pkg/kubernetes"
)

// Operators implements the main logic for commands.
type Operators struct {
	config     *OperatorsConfig
	kubeClient *kubernetes.Kubernetes
	l          *logrus.Entry
}

const (
	namespace              = "default"
	catalogSourceNamespace = "olm"
	operatorGroup          = "percona-operators-group"
	catalogSource          = "percona-dbaas-catalog"
)

type (
	// MonitoringType identifies type of monitoring to be used.
	MonitoringType string

	// OperatorsConfig stores configuration for the operators.
	OperatorsConfig struct {
		Operator OperatorConfig `mapstructure:"operator"`
		// Monitoring stores config for monitoring.
		Monitoring MonitoringConfig `mapstructure:"monitoring"`
		// KubeconfigPath stores path to a kube config.
		KubeconfigPath string `mapstructure:"kubeconfig"`
		// EnableBackup is true if backup shall be enabled.
		EnableBackup bool `mapstructure:"enable_backup"`
		// InstallOLM is true if OLM shall be installed.
		InstallOLM bool `mapstructure:"install_olm"`
		// Channel stores configuration for operator channels.
		Channel ChannelConfig `mapstructure:"channel"`
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

	// OperatorConfig identifies which operators shall be installed
	OperatorConfig struct {
		// PG stores if PostgresSQL shall be installed
		PG bool `mapstructure:"postgresql"`
		// PSMDB stores if MongoDB shall be installed
		PSMDB bool `mapstructure:"mongodb"`
		// PXC stores if XtraDB Cluster shall be installed
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

	if err := o.runOperatorsWizard(); err != nil {
		return err
	}

	return nil
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
	if err := survey.AskOne(pBackup, &o.config.EnableBackup); err != nil {
		return err
	}

	return nil
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
	g.SetLimit(1)

	if o.config.Monitoring.Enabled {
		g.Go(o.installOperator(gCtx, o.config.Channel.VictoriaMetrics, "victoriametrics-operator"))
	}

	if o.config.Operator.PXC {
		g.Go(o.installOperator(gCtx, o.config.Channel.PXC, "percona-xtradb-cluster-operator"))
	}
	if o.config.Operator.PSMDB {
		g.Go(o.installOperator(gCtx, o.config.Channel.PSMDB, "percona-server-mongodb-operator"))
	}
	if o.config.Operator.PG {
		g.Go(o.installOperator(gCtx, o.config.Channel.PG, "percona-postgresql-operator"))
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return o.installOperator(ctx, o.config.Channel.Everest, "dbaas-operator")()
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

// ConnectToEverest connects the k8s cluster to Everest.
//
//nolint:unparam
func (o *Operators) ConnectToEverest() error {
	o.l.Info("Generating service account and connecting with Everest")
	// TODO: Remove this after Percona Everest will be enabled
	//nolint:godox,revive
	return nil
	//nolint:govet
	data, err := os.ReadFile("/Users/gen1us2k/.kube/config")
	if err != nil {
		o.l.Error("failed generating kubeconfig")
		return err
	}
	enc := base64.StdEncoding.EncodeToString(data)
	payload := map[string]string{
		"name":       "minikube",
		"kubeconfig": enc,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		o.l.Error("failed marshaling JSON")
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/kubernetes", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("non 200 status code")
	}
	o.l.Info("DBaaS has been connected")
	return nil
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

	defer resp.Body.Close() //nolint:errcheck,gosec
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
