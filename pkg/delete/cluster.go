package delete //nolint:predeclared

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Cluster implements logic for the cluster command.
type Cluster struct {
	config        ClusterConfig
	everestClient everestClientConnector
	l             *logrus.Entry

	// kubernetesID stores ID of the Kubernetes cluster to be removed.
	kubernetesID string
}

// ClusterConfig stores configuration for the ClusterL command.
type ClusterConfig struct {
	// Name is a name of the Kubernetes cluster in Everest
	Name string

	Everest struct {
		// Endpoint stores URL to Everest.
		Endpoint string
	}

	// Force is true when we shall not prompt for removal.
	Force bool
}

// NewCluster returns a new Cluster struct.
func NewCluster(c ClusterConfig, everestClient everestClientConnector) *Cluster {
	cli := &Cluster{
		config:        c,
		everestClient: everestClient,
		l:             logrus.WithField("component", "delete/cluster"),
	}

	return cli
}

// Run runs the cluster command.
func (c *Cluster) Run(ctx context.Context) error {
	if err := c.populateKubernetesID(ctx); err != nil {
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

	c.l.Infof("Deleting %q Kubernetes cluster from Everest", c.config.Name)
	err := c.everestClient.UnregisterKubernetesCluster(ctx, c.kubernetesID, client.UnregisterKubernetesClusterParams{
		Force: &c.config.Force,
	})
	if err != nil {
		return err
	}

	c.l.Infof("Kubernetes cluster %q has been deleted successfully", c.config.Name)

	return nil
}

func (c *Cluster) populateKubernetesID(ctx context.Context) error {
	if c.kubernetesID != "" {
		return nil
	}

	if c.config.Name == "" {
		if err := c.askForKubernetesCluster(ctx); err != nil {
			return err
		}
	}

	if c.kubernetesID == "" {
		id, err := c.lookupKubernetesClusterID(ctx, c.config.Name)
		if err != nil {
			return err
		}

		if id == "" {
			return errors.New("could not find Kubernetes cluster in Everest by its name")
		}
		c.kubernetesID = id
	}

	if c.kubernetesID == "" {
		// This shall not happen but is here in case the logic
		// above is changed and somehow we end up with an empty kubernetesID
		return errors.New("could not find Kubernetes cluster ID in Everest")
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
	c.kubernetesID = cluster.Id

	return nil
}

func (c *Cluster) lookupKubernetesClusterID(ctx context.Context, name string) (string, error) {
	clusters, err := c.everestClient.ListKubernetesClusters(ctx)
	if err != nil {
		return "", err
	}

	for _, i := range clusters {
		if i.Name == name {
			return i.Id, nil
		}
	}

	return "", nil
}
