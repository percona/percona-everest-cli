// Package config stores configuration passed to the commands.
package config

import "github.com/spf13/viper"

type (
	// MonitoringType identifies type of monitoring to be used.
	MonitoringType string

	// AppConfig stores configuration for the root cli command.
	AppConfig struct {
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

// ParseConfig parses configuration.
func ParseConfig() (*AppConfig, error) {
	c := &AppConfig{}
	err := viper.Unmarshal(c)
	return c, err
}
