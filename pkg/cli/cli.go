package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"

	"github.com/gen1us2k/everest-provisioner/config"
	"github.com/gen1us2k/everest-provisioner/kubernetes"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/sirupsen/logrus"
)

type CLI struct {
	config     *config.AppConfig
	kubeClient *kubernetes.Kubernetes
	l          *logrus.Entry
}

const (
	namespace              = "default"
	catalogSourceNamespace = "olm"
	operatorGroup          = "percona-operators-group"
	catalogSource          = "percona-dbaas-catalog"
)

func New(c *config.AppConfig) (*CLI, error) {
	cli := &CLI{config: c}
	k, err := kubernetes.New(c.Kubeconfig)
	if err != nil {
		return nil, err
	}
	cli.kubeClient = k
	cli.l = logrus.WithField("component", "cli")
	return cli, nil
}

func (c *CLI) ProvisionCluster() error {
	c.l.Info("started provisioning the cluster")
	ctx := context.TODO()
	if c.config.InstallOLM {
		c.l.Info("Installing Operator Lifecycle Manager")
		if err := c.kubeClient.InstallOLMOperator(ctx); err != nil {
			c.l.Error("failed installing OLM")
			return err
		}
	}
	c.l.Info("OLM has been installed")
	c.l.Info("installing Victoria Metrics operator")
	channel, ok := os.LookupEnv("DBAAS_VM_OP_CHANNEL")
	if !ok || channel == "" {
		channel = "stable-v0"
	}
	params := kubernetes.InstallOperatorRequest{
		Namespace:              namespace,
		Name:                   "victoriametrics-operator",
		OperatorGroup:          operatorGroup,
		CatalogSource:          catalogSource,
		CatalogSourceNamespace: catalogSourceNamespace,
		Channel:                channel,
		InstallPlanApproval:    v1alpha1.ApprovalManual,
	}

	if err := c.kubeClient.InstallOperator(ctx, params); err != nil {
		c.l.Error("failed installing victoria metrics operator")
		return err
	}
	c.l.Info("Victoria metrics operator has been installed")
	c.l.Info("Installing PXC operator")
	channel, ok = os.LookupEnv("DBAAS_PXC_OP_CHANNEL")
	if !ok || channel == "" {
		channel = "stable-v1"
	}

	if err := c.kubeClient.InstallOperator(ctx, params); err != nil {
		c.l.Error("failed installing PXC operator")
		return err
	}
	c.l.Info("PXC operator has been installed")
	c.l.Info("Installing PSMDB operator")
	channel, ok = os.LookupEnv("DBAAS_PSMDB_OP_CHANNEL")
	if !ok || channel == "" {
		channel = "stable-v1"
	}
	params.Name = "percona-server-mongodb-operator"
	params.Channel = channel
	if err := c.kubeClient.InstallOperator(ctx, params); err != nil {
		c.l.Error("failed installing PSMDB operator")
		return err
	}
	c.l.Info("PSMDB operator has been installed")
	c.l.Info("Installing DBaaS operator")
	channel, ok = os.LookupEnv("DBAAS_DBAAS_OP_CHANNEL")
	if !ok || channel == "" {
		channel = "stable-v0"
	}
	params.Name = "dbaas-operator"
	params.Channel = channel
	if err := c.kubeClient.InstallOperator(ctx, params); err != nil {
		c.l.Error("failed installing DBaaS operator")
		return err
	}
	c.l.Info("DBaaS operator has been installed")
	//c.l.Info("Installing PG operator")
	//channel, ok = os.LookupEnv("DBAAS_PG_OP_CHANNEL")
	//if !ok || channel == "" {
	//	channel = "stable-v2"
	//}
	//params.Name = "percona-postgresql-operator"
	//params.Channel = channel
	//if err := c.kubeClient.InstallOperator(ctx, params); err != nil {
	//	c.l.Error("failed installing PG operator")
	//	return err
	//}
	//c.l.Info("PG operator has been installed")
	if c.config.Monitoring.Enabled {
		c.l.Info("Started setting up monitoring")
		//if err := c.provisionPMMMonitoring(); err != nil {
		//	return err
		//}
		c.l.Info("Monitoring using PMM has been provisioned")
	}
	return nil
}
func (c *CLI) provisionPMMMonitoring() error {
	account := fmt.Sprintf("dbaas-service-account-%d", rand.Int63())
	c.l.Info("Creating a new service account in PMM")
	token, err := c.provisionPMM(account)
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
func (c *CLI) provisionPMM(account string) (string, error) {
	token, err := c.createAdminToken(account, "")
	return token, err
}
func (c *CLI) ConnectDBaaS() error {
	c.l.Info("Generating service account and connecting with DBaaS")
	data, err := ioutil.ReadFile("/Users/gen1us2k/.kube/config")
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
func (c *CLI) createAdminToken(name string, token string) (string, error) {
	apiKey := map[string]string{
		"name": name,
		"role": "Admin",
	}
	b, err := json.Marshal(apiKey)
	if err != nil {
		return "", err
	}
	fmt.Println(string(b))
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/graph/api/auth/keys", c.config.Monitoring.PMM.Endpoint), bytes.NewReader(b))
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
	fmt.Println(resp.StatusCode)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	fmt.Println(string(data))
	if err != nil {
		return "", err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return "", err
	}
	return m["key"].(string), nil

}
