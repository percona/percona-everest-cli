package delete //nolint:predeclared

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/percona-everest-cli/pkg/kubernetes"
)

// Cluster implements logic for the cluster command.
type Cluster struct {
	config        ClusterConfig
	everestClient everestClientConnector
	kubeClient    *kubernetes.Kubernetes
	l             *logrus.Entry

	kubernetes *k8sCluster
}

type k8sCluster struct {
	// id stores ID of the Kubernetes cluster to be removed.
	id string
	// namespace stores everest namespace in the k8s cluster.
	namespace string
}

// ClusterConfig stores configuration for the ClusterL command.
type ClusterConfig struct {
	// Name is a name of the Kubernetes cluster in Everest
	Name string

	Everest struct {
		// Endpoint stores URL to Everest.
		Endpoint string
	}

	// KubeconfigPath is a path to a kubeconfig
	KubeconfigPath string `mapstructure:"kubeconfig"`

	// Force is true when we shall not prompt for removal.
	Force bool
}

// NewCluster returns a new Cluster struct.
func NewCluster(c ClusterConfig, everestClient everestClientConnector) (*Cluster, error) {
	l := logrus.WithField("component", "delete/cluster")
	kubeClient, err := kubernetes.New(c.KubeconfigPath, l)
	if err != nil {
		return nil, err
	}

	cli := &Cluster{
		config:        c,
		everestClient: everestClient,
		kubeClient:    kubeClient,
		l:             l,
	}

	return cli, nil
}

// Run runs the cluster command.
func (c *Cluster) Run(ctx context.Context) error {
	if err := c.populateKubernetesCluster(ctx); err != nil {
		return err
	}

	if !c.config.Force {
		confirm := &survey.Confirm{
			Message: fmt.Sprintf("Are you sure you want to delete the %q Kubernetes cluster in Everest?", c.config.Name),
		}
		prompt := false
		if err := survey.AskOne(confirm, &prompt); err != nil {
			return err
		}

		if !prompt {
			c.l.Info("Exiting")
			return nil
		}
	}

	if c.kubernetes == nil {
		// This shall not happen but it's here in case the logic
		// above becomes broken and somehow we end up with an empty kubernetes field.
		return errors.New("could not find Kubernetes cluster in Everest")
	}

	c.l.Infof("Deleting all Kubernetes monitoring resources in Kubernetes cluster %q", c.config.Name)
	if err := c.kubeClient.DeleteAllMonitoringResources(ctx, c.kubernetes.namespace); err != nil {
		return errors.Wrap(err, "could not delete monitoring resources from the Kubernetes cluster")
	}

	c.l.Infof("Deleting Kubernetes cluster %q from Everest", c.config.Name)
	err := c.everestClient.UnregisterKubernetesCluster(ctx, c.kubernetes.id, client.UnregisterKubernetesClusterParams{
		Force: &c.config.Force,
	})
	if err != nil {
		return err
	}

	c.l.Infof("Kubernetes cluster %q has been deleted successfully", c.config.Name)

	return nil
}

func (c *Cluster) populateKubernetesCluster(ctx context.Context) error {
	if c.kubernetes != nil {
		return nil
	}

	if c.config.Name == "" {
		if err := c.askForKubernetesCluster(ctx); err != nil {
			return err
		}
	}

	if c.kubernetes == nil {
		cluster, err := c.lookupKubernetesCluster(ctx, c.config.Name)
		if err != nil {
			return err
		}

		c.kubernetes = &k8sCluster{
			id:        cluster.Id,
			namespace: cluster.Namespace,
		}
	}

	return nil
}

func (c *Cluster) askForKubernetesCluster(ctx context.Context) error {
	clusters, err := c.everestClient.ListKubernetesClusters(ctx)
	if err != nil {
		return err
	}

	opts := make([]string, 0, len(clusters)+1)
	for _, i := range clusters {
		opts = append(opts, i.Name)
	}

	pCluster := &survey.Select{
		Message: "Select a Kubernetes cluster to delete:",
		Options: opts,
	}
	ix := 0
	if err := survey.AskOne(pCluster, &ix); err != nil {
		return err
	}

	cluster := clusters[ix]
	c.config.Name = cluster.Name
	c.kubernetes = &k8sCluster{
		id:        cluster.Id,
		namespace: cluster.Namespace,
	}

	return nil
}

func (c *Cluster) lookupKubernetesCluster(ctx context.Context, name string) (*client.KubernetesCluster, error) {
	clusters, err := c.everestClient.ListKubernetesClusters(ctx)
	if err != nil {
		return nil, err
	}

	for _, i := range clusters {
		if i.Name == name {
			return &i, nil
		}
	}

	return nil, errors.New("could not find Kubernetes cluster in Everest by its name")
}
