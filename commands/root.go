/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/

// Package commands implements main logic for cli commands.
package commands

import "github.com/spf13/cobra"

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
	}

	rootCmd.AddCommand(newInstallCmd())

	return rootCmd
}
