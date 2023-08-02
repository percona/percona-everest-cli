// Package main is the main file for everest cli.
package main

import (
	"os"

	"github.com/go-logr/zapr"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/percona/percona-everest-cli/commands"
	"github.com/percona/percona-everest-cli/pkg/logger"
)

func main() {
	l := logger.MustInitLogger(false)

	// This is required because controller-runtime requires a logger
	// to be set within 30 seconds of the program initialization.
	log := zapr.NewLogger(l)
	ctrlruntimelog.SetLogger(log)

	rootCmd := commands.NewRootCmd(l.Sugar())
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
