/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/

// Package cmd implements main logic for cli commands.
package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/percona/percona-everest-cli/config"
	"github.com/percona/percona-everest-cli/pkg/cli"
)

// NewRootCmd creates a new root command for the cli.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "everest-cli",
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := config.ParseConfig()
			if err != nil {
				os.Exit(1)
			}
			cli, err := cli.New(c)
			if err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
			if err := cli.ProvisionCluster(); err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
			if err := cli.ConnectToEverest(); err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
		},
	}

	initFlags(rootCmd)

	return rootCmd
}

func initFlags(rootCmd *cobra.Command) {
	rootCmd.Flags().BoolP("monitoring.enabled", "m", true, "Enable monitoring")
	rootCmd.Flags().StringP("monitoring.type", "", "pmm", "Monitoring type")
	rootCmd.Flags().StringP("monitoring.pmm.endpoint", "", "http://127.0.0.1", "PMM endpoint URL")
	rootCmd.Flags().StringP("monitoring.pmm.username", "", "admin", "PMM username")
	rootCmd.Flags().StringP("monitoring.pmm.password", "", "password", "PMM password")
	rootCmd.Flags().BoolP("enable_backup", "b", false, "Enable backups")
	rootCmd.Flags().BoolP("install_olm", "o", true, "Install OLM")
	rootCmd.Flags().StringP("kubeconfig", "k", "~/.kube/config", "specify kubeconfig")

	viper.BindPFlag("monitoring.enabled", rootCmd.Flags().Lookup("monitoring.enabled"))           //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.type", rootCmd.Flags().Lookup("monitoring.type"))                 //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.pmm.endpoint", rootCmd.Flags().Lookup("monitoring.pmm.endpoint")) //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.pmm.username", rootCmd.Flags().Lookup("monitoring.pmm.username")) //nolint:errcheck,gosec
	viper.BindPFlag("monitoring.pmm.password", rootCmd.Flags().Lookup("monitoring.pmm.password")) //nolint:errcheck,gosec
	viper.BindPFlag("enable_backup", rootCmd.Flags().Lookup("enable_backup"))                     //nolint:errcheck,gosec
	viper.BindPFlag("install_olm", rootCmd.Flags().Lookup("install_olm"))                         //nolint:errcheck,gosec
	viper.BindPFlag("kubeconfig", rootCmd.Flags().Lookup("kubeconfig"))                           //nolint:errcheck,gosec
}
