package main

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/percona/percona-everest-cli/commands"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	rootCmd := commands.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
