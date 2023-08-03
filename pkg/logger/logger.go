// percona-everest-backend
// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package logger provides functionality related to logging.
package logger

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// MustInitLogger initializes a logger and panics in case of an error.
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

// MustInitVerboseLogger initializes a verbose logger and panics in case of an error.
func MustInitVerboseLogger(json bool) *zap.Logger {
	lCfg := zap.NewDevelopmentConfig()
	if json {
		lCfg.Encoding = "json"
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

	if verbose {
		*l = *MustInitVerboseLogger(json).Sugar()
	} else if json {
		*l = *MustInitLogger(true).Sugar()
	}
}
