package upgrade

import (
	"context"
	"errors"
	"net/url"

	"github.com/AlecAivazis/survey/v2"
	"github.com/percona/percona-everest-cli/data"
	"github.com/percona/percona-everest-cli/pkg/kubernetes"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
)

type (
	UpgradeConfig struct {
		// Name of the Kubernetes Cluster
		Name string
		// Namespace defines the namespace operators shall be installed to.
		Namespace string
		// KubeconfigPath is a path to a kubeconfig
		KubeconfigPath string `mapstructure:"kubeconfig"`
		// UpgradeOLM defines do we need to upgrade OLM or not.
		UpgradeOLM bool `mapstructure:"upgrade_olm"`
		// SkipWizard skips wizard during installation.
		SkipWizard bool `mapstructure:"skip-wizard"`
	}
	Upgrade struct {
		l *zap.SugaredLogger

		config     UpgradeConfig
		kubeClient *kubernetes.Kubernetes
	}
)

// NewUpgrade returns a new Upgrade struct.
func NewUpgrade(c UpgradeConfig, l *zap.SugaredLogger) (*Upgrade, error) {
	cli := &Upgrade{
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
func (o *Upgrade) Run(ctx context.Context) error {
	csv, err := o.kubeClient.GetClusterServiceVersion(ctx, types.NamespacedName{
		Name:      "packageserver",
		Namespace: "olm",
	})
	if err != nil {
		return err
	}
	if csv.Spec.Version.String() != data.OLMVersion {
		if !o.config.SkipWizard {
			if err := o.runWizard(ctx); err != nil {
				return err
			}
		}
		if o.config.UpgradeOLM {
			o.l.Info("Upgrading OLM")
			if err := o.kubeClient.InstallOLMOperator(ctx, true); err != nil {
				o.l.Error(err)
			}
			o.l.Info("OLM has been upgraded")
		}
	}
	o.l.Info("Upgrading Percona Catalog")
	if err := o.kubeClient.InstallPerconaCatalog(ctx); err != nil {
		o.l.Error(err)
	}
	o.l.Info("Percona Catalog has been upgraded")
	return nil
}

// runWizard runs installation wizard.
func (o *Upgrade) runWizard(ctx context.Context) error {
	pMonitor := &survey.Confirm{
		Message: "Do you want to upgrade OLM?",
		Default: o.config.UpgradeOLM,
	}
	return survey.AskOne(pMonitor, &o.config.UpgradeOLM)
}
