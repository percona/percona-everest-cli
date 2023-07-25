// Package main is the main file for everest cli.
package main

import (
	"os"

	"github.com/bombsimon/logrusr/v4"
	"github.com/sirupsen/logrus"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/percona/percona-everest-cli/commands"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// This is required because controller-runtime requires a logger
	// to be set within 30 seconds of the program initialization.
	log := logrusr.New(logrus.StandardLogger())
	ctrlruntimelog.SetLogger(log)

	rootCmd := commands.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
