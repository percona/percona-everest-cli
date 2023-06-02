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
		// Monitoring stores config for monitoring.
		Monitoring MonitoringConfig `mapstructure:"monitoring"`
		// KubeconfigPath stores path to a kube config.
		KubeconfigPath string `mapstructure:"kubeconfig"`
		// EnableBackup is true if backup shall be enabled.
		EnableBackup bool `mapstructure:"enable_backup"`
		// InstallOLM is true if OLM shall be installed.
		InstallOLM bool `mapstructure:"install_olm"`
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

	// PMMConfig stores configuration for PMM monitoring type.
	PMMConfig struct {
		// Endpoint stores URL to PMM.
		Endpoint string `mapstructure:"endpoint"`
		// Username stores username for authentication against PMM.
		Username string `mapstructure:"username"`
		// Password stores password for authentication against PMM.
		Password string `mapstructure:"password"`
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

// ProvisionOperators provisions all configured operators to a k8s cluster.
func (c *Operators) ProvisionOperators() error {
	c.l.Info("Started provisioning the cluster")
	ctx := context.TODO()

	if err := c.provisionOLM(ctx); err != nil {
		return err
	}

	if err := c.provisionOperators(ctx); err != nil {
		return err
	}

	if c.config.Monitoring.Enabled {
		c.l.Info("Started setting up monitoring")
		// if err := c.provisionPMMMonitoring(); err != nil {
		// 	return err
		// }
		c.l.Info("Monitoring using PMM has been provisioned")
	}

	return nil
}

func (c *Operators) provisionOLM(ctx context.Context) error {
	if c.config.InstallOLM {
		c.l.Info("Installing Operator Lifecycle Manager")
		if err := c.kubeClient.InstallOLMOperator(ctx); err != nil {
			c.l.Error("failed installing OLM")
			return err
		}
	}
	c.l.Info("OLM has been installed")

	return nil
}

func (c *Operators) provisionOperators(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)
	// We set the limit to 1 since operator installation
	// requires an update to the same installation plan which
	// results in race-conditions with a higher limit.
	// The limit can be removed after it's refactored.
	g.SetLimit(1)

	if c.config.Monitoring.Enabled {
		g.Go(c.installOperator(gCtx, "DBAAS_VM_OP_CHANNEL", "victoriametrics-operator", "stable-v0"))
	}

	g.Go(c.installOperator(gCtx, "DBAAS_PXC_OP_CHANNEL", "percona-xtradb-cluster-operator", "stable-v1"))
	g.Go(c.installOperator(gCtx, "DBAAS_PSMDB_OP_CHANNEL", "percona-server-mongodb-operator", "stable-v1"))
	g.Go(c.installOperator(gCtx, "DBAAS_PG_OP_CHANNEL", "percona-postgresql-operator", "fast-v2"))

	if err := g.Wait(); err != nil {
		return err
	}

	return c.installOperator(ctx, "DBAAS_DBAAS_OP_CHANNEL", "dbaas-operator", "stable-v0")()
}

func (c *Operators) installOperator(ctx context.Context, envName, operatorName, defaultChannel string) func() error {
	return func() error {
		c.l.Infof("Installing %s operator", operatorName)

		channel, ok := os.LookupEnv(envName)
		if !ok || channel == "" {
			channel = defaultChannel
		}
		params := kubernetes.InstallOperatorRequest{
			Namespace:              namespace,
			Name:                   operatorName,
			OperatorGroup:          operatorGroup,
			CatalogSource:          catalogSource,
			CatalogSourceNamespace: catalogSourceNamespace,
			Channel:                channel,
			InstallPlanApproval:    v1alpha1.ApprovalManual,
		}

		if err := c.kubeClient.InstallOperator(ctx, params); err != nil {
			c.l.Errorf("failed installing %s operator", operatorName)
			return err
		}
		c.l.Infof("%s operator has been installed", operatorName)

		return nil
	}
}

//nolint:unused
func (c *Operators) provisionPMMMonitoring(ctx context.Context) error {
	account := fmt.Sprintf("everest-service-account-%s", uuid.NewString())
	c.l.Info("Creating a new service account in PMM")
	token, err := c.provisionPMM(ctx, account)
	if err != nil {
		return err
	}
	c.l.Info("New token has been generated")
	c.l.Info("Started provisioning monitoring in k8s cluster")
	err = c.kubeClient.ProvisionMonitoring(account, token, c.config.Monitoring.PMM.Endpoint)
	if err != nil {
		c.l.Error("failed provisioning monitoring")
		return err
	}

	return nil
}

//nolint:unused
func (c *Operators) provisionPMM(ctx context.Context, account string) (string, error) {
	token, err := c.createAdminToken(ctx, account, "")
	return token, err
}

// ConnectToEverest connects the k8s cluster to Everest.
//
//nolint:unparam
func (c *Operators) ConnectToEverest() error {
	c.l.Info("Generating service account and connecting with Everest")
	// TODO: Remove this after Percona Everest will be enabled
	//nolint:godox,revive
	return nil
	//nolint:govet
	data, err := os.ReadFile("/Users/gen1us2k/.kube/config")
	if err != nil {
		c.l.Error("failed generating kubeconfig")
		return err
	}
	enc := base64.StdEncoding.EncodeToString(data)
	payload := map[string]string{
		"name":       "minikube",
		"kubeconfig": enc,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		c.l.Error("failed marshaling JSON")
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
	c.l.Info("DBaaS has been connected")
	return nil
}

//nolint:unused
func (c *Operators) createAdminToken(ctx context.Context, name string, token string) (string, error) {
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
		fmt.Sprintf("%s/graph/api/auth/keys", c.config.Monitoring.PMM.Endpoint),
		bytes.NewReader(b),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if token == "" {
		req.SetBasicAuth(c.config.Monitoring.PMM.Username, c.config.Monitoring.PMM.Password)
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
