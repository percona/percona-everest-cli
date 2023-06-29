package provision

import (
	"context"
	"encoding/json"
	"fmt"

	dbclusterv1 "github.com/percona/dbaas-operator/api/v1"
	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MySQL implements logic for the MySQL command.
type MySQL struct {
	config        *MySQLConfig
	everestClient everestClientConnector
	l             *logrus.Entry
}

// MySQLConfig stores configuration for the MySQL command.
type MySQLConfig struct {
	Name         string
	Namespace    string
	KubernetesID string `mapstructure:"kubernetes-id"`

	Everest struct {
		// Endpoint stores URL to Everest.
		Endpoint string
	}

	DB struct {
		Version string
	}

	Nodes  int
	CPU    string
	Memory string
	Disk   string

	ExternalAccess bool `mapstructure:"external-access"`

	Monitoring struct {
		// Enable is true if monitoring shall be enabled.
		Enable bool
		// Endpoint stores URL to PMM.
		Endpoint string
		// Username stores username for authentication against PMM.
		Username string
		// Password stores password for authentication against PMM.
		Password string
	}
}

// NewMySQL returns a new MySQL struct.
func NewMySQL(c *MySQLConfig, everestClient everestClientConnector) *MySQL {
	if c == nil {
		logrus.Panic("MySQLConfig is required")
	}

	cli := &MySQL{
		config:        c,
		everestClient: everestClient,
		l:             logrus.WithField("component", "provision/mysql"),
	}

	return cli
}

// Run runs the MySQL command.
func (m *MySQL) Run(ctx context.Context) error {
	m.l.Info("Preparing cluster config")
	body, err := m.prepareBody()
	if err != nil {
		return err
	}

	m.l.Infof("Creating %q database cluster", m.config.Name)
	_, err = m.everestClient.CreateDBCluster(ctx, m.config.KubernetesID, *body)
	if err != nil {
		return err
	}

	m.l.Infof("Database cluster %q has been scheduled to Kubernetes", m.config.Name)

	return nil
}

func (m *MySQL) prepareBody() (*client.DatabaseCluster, error) {
	cpu, err := resource.ParseQuantity(m.config.CPU)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse cpu")
	}

	memory, err := resource.ParseQuantity(m.config.Memory)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse memory")
	}

	disk, err := resource.ParseQuantity(m.config.Disk)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse disk storage")
	}

	payload := dbclusterv1.DatabaseCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "dbaas.percona.com/v1",
			Kind:       "DatabaseCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.config.Name,
			Namespace: m.config.Namespace,
		},
		Spec: dbclusterv1.DatabaseSpec{
			Database:      dbclusterv1.PXCEngine,
			DatabaseImage: fmt.Sprintf("percona/percona-xtradb-cluster:%s", m.config.DB.Version),
			ClusterSize:   int32(m.config.Nodes),
			DBInstance: dbclusterv1.DBInstanceSpec{
				CPU:      cpu,
				Memory:   memory,
				DiskSize: disk,
			},
			LoadBalancer: dbclusterv1.LoadBalancerSpec{
				Type:       dbclusterv1.LoadBalancerHAProxy,
				ExposeType: corev1.ServiceTypeClusterIP,
				Size:       int32(m.config.Nodes),
				Image:      "percona/percona-xtradb-cluster-operator:1.12.0-haproxy",
			},
		},
	}

	if m.config.ExternalAccess {
		m.l.Debug("Enabling external access")
		payload.Spec.LoadBalancer = dbclusterv1.LoadBalancerSpec{
			Size:       int32(m.config.Nodes),
			ExposeType: corev1.ServiceTypeLoadBalancer,
		}
	}

	return m.convertPayload(payload)
}

func (m *MySQL) convertPayload(payload dbclusterv1.DatabaseCluster) (*client.DatabaseCluster, error) {
	bodyJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal payload to json")
	}

	m.l.Debug(string(bodyJSON))

	body := &client.DatabaseCluster{}
	err = json.Unmarshal(bodyJSON, body)
	if err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal payload back to json")
	}

	return body, nil
}
