// percona-everest-cli
// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package monitoring holds the main logic for provision monitoring
package monitoring

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/percona/percona-everest-backend/client"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/percona/percona-everest-cli/commands/common"
	everestClient "github.com/percona/percona-everest-cli/pkg/everest/client"
	"github.com/percona/percona-everest-cli/pkg/kubernetes"
)

const (
	everestBackendServiceName    = "percona-everest-backend"
	everestBackendDeploymentName = "percona-everest"
	vmOperatorName               = "victoriametrics-operator"
)

// Monitoring implements the logic for provisioning monitoring.
type Monitoring struct {
	l *zap.SugaredLogger

	config        Config
	everestClient everestClientConnector
	kubeClient    *kubernetes.Kubernetes

	// monitoringInstanceName stores the resolved monitoring instance name.
	monitoringInstanceName string
}

type (
	// monitoringType identifies type of monitoring to be used.
	monitoringType string

	// Config stores configuration for the operators.
	Config struct {
		// Namespace defines the namespace operators shall be installed to.
		Namespace string
		// SkipWizard skips wizard during installation.
		SkipWizard bool `mapstructure:"skip-wizard"`
		// KubeconfigPath is a path to a kubeconfig
		KubeconfigPath string `mapstructure:"kubeconfig"`

		// EverestToken defines a token to connect to Everest
		EverestToken string `mapstructure:"everest-token"`
		// EverestURL defines an URL to connect to Everest
		EverestURL string `mapstructure:"everest-url"`

		// InstanceName stores monitoring instance name from Everest.
		// If provided, the other monitoring configuration is ignored.
		InstanceName string `mapstructure:"instance-name"`
		// NewInstanceName defines name for a new monitoring instance
		// if it's created.
		NewInstanceName string `mapstructure:"new-instance-name"`
		// Type stores the type of monitoring to be used.
		Type monitoringType
		// PMM stores configuration for PMM monitoring type.
		PMM *PMMConfig
	}

	// PMMConfig stores configuration for PMM monitoring type.
	PMMConfig struct {
		// Endpoint stores URL to PMM.
		Endpoint string
		// Username stores username for authentication against PMM.
		Username string
		// Password stores password for authentication against PMM.
		Password string
	}
)

// NewMonitoring returns a new Monitoring struct.
func NewMonitoring(c Config, l *zap.SugaredLogger) (*Monitoring, error) {
	cli := &Monitoring{
		config: c,
		l:      l.With("component", "monitoring/enable"),
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
func (m *Monitoring) Run(ctx context.Context) error {
	if err := m.populateConfig(ctx); err != nil {
		return err
	}
	if err := m.provisionNamespace(ctx); err != nil {
		return err
	}
	if err := m.provisionMonitoring(ctx); err != nil {
		return err
	}

	return nil
}

func (m *Monitoring) populateConfig(ctx context.Context) error {
	if !m.config.SkipWizard {
		if err := m.runEverestWizard(ctx); err != nil {
			return err
		}
		if err := m.runMonitoringWizard(); err != nil {
			return err
		}
	}
	m.config.EverestURL = strings.TrimSpace(m.config.EverestURL)
	m.config.EverestToken = strings.TrimSpace(m.config.EverestToken)

	if err := m.configureEverestConnector(); err != nil {
		return err
	}
	if err := m.checkEverestConnection(ctx); err != nil {
		return err
	}

	return nil
}

// provisionNamespace provisions a namespace for Everest.
func (m *Monitoring) provisionNamespace(ctx context.Context) error {
	_, err := m.kubeClient.GetNamespace(ctx, m.config.Namespace)
	if err != nil && k8serrors.IsNotFound(err) {
		return fmt.Errorf("namespace %s is not found", m.config.Namespace)
	}
	return err
}

func (m *Monitoring) installVMOperator(ctx context.Context) error {
	m.l.Infof("Installing %s operator", vmOperatorName)

	params := kubernetes.InstallOperatorRequest{
		Namespace:              m.config.Namespace,
		Name:                   vmOperatorName,
		OperatorGroup:          kubernetes.OperatorGroup,
		CatalogSource:          kubernetes.CatalogSource,
		CatalogSourceNamespace: kubernetes.CatalogSourceNamespace,
		Channel:                "stable-v0",
		InstallPlanApproval:    v1alpha1.ApprovalManual,
	}

	if err := m.kubeClient.InstallOperator(ctx, params); err != nil {
		m.l.Errorf("failed installing %s operator", vmOperatorName)
		return err
	}
	m.l.Infof("%s operator has been installed", vmOperatorName)
	return nil
}

func (m *Monitoring) provisionMonitoring(ctx context.Context) error {
	l := m.l.With("action", "monitoring")
	l.Info("Preparing k8s cluster for monitoring")
	if err := m.installVMOperator(ctx); err != nil {
		return err
	}
	if err := m.kubeClient.ProvisionMonitoring(m.config.Namespace); err != nil {
		return errors.Join(err, errors.New("could not provision monitoring configuration"))
	}

	l.Info("K8s cluster monitoring has been provisioned successfully")
	if err := m.resolveMonitoringInstanceName(ctx); err != nil {
		return err
	}
	m.l.Info("Deploying VMAgent to k8s cluster")
	if err := m.kubeClient.RestartEverest(ctx, everestBackendServiceName, m.config.Namespace); err != nil {
		return err
	}
	if err := m.kubeClient.WaitForRollout(ctx, everestBackendDeploymentName, m.config.Namespace); err != nil {
		return errors.Join(err, errors.New("failed waiting for Everest to be ready"))
	}

	if err := m.waitForEverestConnection(ctx); err != nil {
		return err
	}

	// We retry for a bit since the MonitoringConfig may not be properly
	// deployed yet and we get a HTTP 500 in this case.
	err := wait.PollUntilContextTimeout(ctx, 3*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		m.l.Debug("Trying to enable Kubernetes cluster monitoring")
		err := m.everestClient.SetKubernetesClusterMonitoring(ctx, client.KubernetesClusterMonitoring{
			Enable:                 true,
			MonitoringInstanceName: m.monitoringInstanceName,
		})
		if err != nil {
			m.l.Debug(errors.Join(err, errors.New("could not enable Kubernetes cluster monitoring")))
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return errors.Join(err, errors.New("could not enable Kubernetes cluster monitoring"))
	}

	m.l.Info("VMAgent deployed successfully")
	return nil
}

func (m *Monitoring) waitForEverestConnection(ctx context.Context) error {
	sleep := time.Second
	for i := 0; i < 3; i++ {
		time.Sleep(sleep)
		sleep *= 2
		if err := m.checkEverestConnection(ctx); err != nil {
			if i != 2 {
				continue
			}
			var u *url.Error
			if errors.As(err, &u) {
				m.l.Debug(err)

				l := m.l.WithOptions(zap.AddStacktrace(zap.DPanicLevel))
				l.Error("Could not connect to Everest. " +
					"Make sure Everest is running and is accessible from this machine.",
				)
				return common.ErrExitWithError
			}

			return errors.Join(err, errors.New("could not check connection to Everest"))
		}
	}
	return nil
}

func (m *Monitoring) resolveMonitoringInstanceName(ctx context.Context) error {
	if m.config.InstanceName != "" {
		i, err := m.everestClient.GetMonitoringInstance(ctx, m.config.InstanceName)
		if err != nil {
			return errors.Join(err, fmt.Errorf("could not get monitoring instance with name %s from Everest", m.config.InstanceName))
		}
		m.monitoringInstanceName = i.Name
		return nil
	}

	if m.config.NewInstanceName == "" && m.monitoringInstanceName == "" {
		return errors.New("new-instance-name is required when creating a new monitoring instance")
	}

	err := m.createPMMMonitoringInstance(
		ctx, m.config.NewInstanceName, m.config.PMM.Endpoint,
		m.config.PMM.Username, m.config.PMM.Password,
	)
	if err != nil {
		return errors.Join(err, errors.New("could not create a new PMM monitoring instance in Everest"))
	}

	m.monitoringInstanceName = m.config.NewInstanceName

	return nil
}

func (m *Monitoring) createPMMMonitoringInstance(ctx context.Context, name, url, username, password string) error {
	_, err := m.everestClient.CreateMonitoringInstance(ctx, client.MonitoringInstanceCreateParams{
		Type: client.MonitoringInstanceCreateParamsTypePmm,
		Name: name,
		Url:  url,
		Pmm: &client.PMMMonitoringInstanceSpec{
			User:     username,
			Password: password,
		},
	})
	if err != nil {
		return errors.Join(err, errors.New("could not create a new monitoring instance"))
	}

	return nil
}

func (m *Monitoring) configureEverestConnector() error {
	e, err := everestClient.NewEverestFromURL(m.config.EverestURL, m.config.EverestToken)
	if err != nil {
		return err
	}
	m.everestClient = e
	return nil
}

func (m *Monitoring) runEverestWizard(ctx context.Context) error {
	pURL := &survey.Input{
		Message: "Everest URL endpoint",
		Default: m.config.EverestURL,
	}
	if err := survey.AskOne(
		pURL,
		&m.config.EverestURL,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}
	pToken := &survey.Password{Message: "Everest Token"}
	if err := survey.AskOne(
		pToken,
		&m.config.EverestToken,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}
	return nil
}

func (m *Monitoring) runMonitoringWizard() error {
	if m.config.PMM == nil {
		m.config.PMM = &PMMConfig{}
	}
	pName := &survey.Input{
		Message: "Registered instance name",
	}
	if err := survey.AskOne(
		pName,
		&m.config.InstanceName,
	); err != nil {
		return err
	}

	if m.config.InstanceName == "" {
		if err := m.runMonitoringNewURLWizard(); err != nil {
			return err
		}
	}

	return nil
}

func (m *Monitoring) runMonitoringNewURLWizard() error {
	pURL := &survey.Input{
		Message: "PMM URL Endpoint",
		Default: m.config.PMM.Endpoint,
	}
	if err := survey.AskOne(
		pURL,
		&m.config.PMM.Endpoint,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}
	m.config.PMM.Endpoint = strings.TrimSpace(m.config.PMM.Endpoint)

	pUser := &survey.Input{
		Message: "Username",
		Default: m.config.PMM.Username,
	}
	if err := survey.AskOne(
		pUser,
		&m.config.PMM.Username,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}

	pPass := &survey.Password{Message: "Password"}
	if err := survey.AskOne(
		pPass,
		&m.config.PMM.Password,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}

	pName := &survey.Input{
		Message: "Name for the new monitoring instance",
		Default: m.config.NewInstanceName,
	}
	if err := survey.AskOne(
		pName,
		&m.config.NewInstanceName,
		survey.WithValidator(survey.Required),
	); err != nil {
		return err
	}

	return nil
}

func (m *Monitoring) checkEverestConnection(ctx context.Context) error {
	_, err := m.everestClient.ListMonitoringInstances(ctx)
	return err
}
