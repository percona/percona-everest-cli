// Package list holds the main logic for list commands.
package list

import (
	"context"
	"fmt"
	"strings"

	everestv1alpha1 "github.com/percona/everest-operator/api/v1alpha1"
	"github.com/sirupsen/logrus"
)

// DBEngines implements the main logic for commands.
type DBEngines struct {
	config        DBEnginesConfig
	everestClient everestClientConnector
	l             *logrus.Entry
}

type (
	// DBEnginesConfig stores configuration for the database engines.
	DBEnginesConfig struct {
		KubernetesID string `mapstructure:"kubernetes-id"`
		Everest      EverestConfig
	}

	// EverestConfig stores config for Everest.
	EverestConfig struct {
		// Endpoint stores URL to Everest.
		Endpoint string
	}
)

type (
	// DBEnginesList stores list of database engines.
	DBEnginesList map[everestv1alpha1.EngineType]DBEngine
	// DBEngine stores information about a database engine.
	DBEngine struct {
		Version string `json:"version"`
	}
)

// String returns string result of database engines list.
func (d DBEnginesList) String() string {
	out := make([]string, 0, len(d))
	for engine, e := range d {
		out = append(out, fmt.Sprintf("%s %s", engine, e.Version))
	}

	return strings.Join(out, "\n")
}

// NewDatabaseEngines returns a new DBEngines struct.
func NewDatabaseEngines(c DBEnginesConfig, everestClient everestClientConnector) *DBEngines {
	cli := &DBEngines{
		config:        c,
		everestClient: everestClient,
		l:             logrus.WithField("component", "list/databaseengines"),
	}

	return cli
}

// Run runs the database engines list command.
func (d *DBEngines) Run(ctx context.Context) (DBEnginesList, error) {
	dbEngines, err := d.everestClient.ListDatabaseEngines(ctx, d.config.KubernetesID)
	if err != nil {
		return nil, err
	}

	res := make(DBEnginesList)

	if dbEngines.Items == nil {
		return res, nil
	}

	for _, db := range *dbEngines.Items {
		if db.Spec == nil {
			continue
		}

		if db.Status == nil {
			continue
		}

		e := everestv1alpha1.EngineType(db.Spec.Type)
		res[e] = DBEngine{
			Version: *db.Status.OperatorVersion,
		}
	}

	return res, nil
}
