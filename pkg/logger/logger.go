// Package logger provides functionality related to logging.
package logger

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// MustInitLogger initializes logger and panics in case of an error.
func MustInitLogger(json bool) *zap.Logger {
	lCfg := zap.NewProductionConfig()
	lCfg.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	if !json {
		lCfg.Encoding = "console"
	}

	l, err := lCfg.Build()
	if err != nil {
		panic(fmt.Sprintf("Cannot initialize logger: %s", err))
	}

	return l
}

// InitLoggerInRootCmd inits the provided logger instance based on command's flags.
// This is meant to be run by the root command in PersistentPreRun step.
func InitLoggerInRootCmd(cmd *cobra.Command, l *zap.SugaredLogger) {
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		l.Warn(`Could not parse "verbose" flag`)
	}

	json, err := cmd.Flags().GetBool("json")
	if err != nil {
		l.Warn(`Could not parse "json" flag`)
	}

	if json {
		*l = *MustInitLogger(true).Sugar()
	}

	if !verbose {
		*l = *l.WithOptions(zap.IncreaseLevel(zap.InfoLevel))
	}
}
