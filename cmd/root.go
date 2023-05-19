/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/gen1us2k/everest-provisioner/config"
	"github.com/gen1us2k/everest-provisioner/pkg/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "everest-provisioner",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		c, err := config.ParseConfig()
		if err != nil {
			os.Exit(1)
		}
		cli, err := cli.New(c)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := cli.ProvisionCluster(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := cli.ConnectDBaaS(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.everest-provisioner.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("monitoring.enabled", "m", true, "Enable monitoring")
	viper.BindPFlag("monitoring.enabled", rootCmd.Flags().Lookup("monitoring.enabled"))
	rootCmd.Flags().StringP("monitoring.type", "", "pmm", "Monitoring type")
	viper.BindPFlag("monitoring.type", rootCmd.Flags().Lookup("monitoring.type"))
	rootCmd.Flags().StringP("monitoring.pmm.endpoint", "", "http://127.0.0.1", "PMM endpoint URL")
	viper.BindPFlag("monitoring.pmm.endpoint", rootCmd.Flags().Lookup("monitoring.pmm.endpoint"))
	rootCmd.Flags().StringP("monitoring.pmm.username", "", "admin", "PMM username")
	viper.BindPFlag("monitoring.pmm.username", rootCmd.Flags().Lookup("monitoring.pmm.username"))
	rootCmd.Flags().StringP("monitoring.pmm.password", "", "password", "PMM password")
	viper.BindPFlag("monitoring.pmm.password", rootCmd.Flags().Lookup("monitoring.pmm.password"))
	rootCmd.Flags().BoolP("enable_backup", "b", false, "Enable backups")
	viper.BindPFlag("enable_backup", rootCmd.Flags().Lookup("enable_backup"))
	rootCmd.Flags().BoolP("install_olm", "o", true, "Install OLM")
	viper.BindPFlag("install_olm", rootCmd.Flags().Lookup("install_olm"))
	rootCmd.Flags().StringP("kubeconfig", "k", "~/.kube/config", "specify kubeconfig")
	viper.BindPFlag("kubeconfig", rootCmd.Flags().Lookup("kubeconfig"))
}
