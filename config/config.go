package config

import "github.com/spf13/viper"

const MonitoringTypePMM = "pmm"

type (
	MonitoringType string
	AppConfig      struct {
		Monitoring   MonitoringConfig `mapstructure:"monitoring"`
		Kubeconfig   string           `mapstructure:"kubeconfig"`
		EnableBackup bool             `mapstructure:"enable_backup"`
		InstallOLM   bool             `mapstructure:"install_olm"`
	}
	MonitoringConfig struct {
		Enabled bool           `mapstructure:"enabled"`
		Type    MonitoringType `mapstructure:"type"`
		PMM     *PMMConfig     `mapstructure:"pmm"`
	}
	PMMConfig struct {
		Endpoint string `mapstructure:"endpoint"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
	}
)

func ParseConfig() (*AppConfig, error) {
	viper.SetConfigType("yaml")
	c := &AppConfig{}
	err := viper.Unmarshal(c)
	return c, err
}
