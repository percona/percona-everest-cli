// Package output provides utilities to print output in commands.
package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// PrintOutput prints output as a string or json.
func PrintOutput(cmd *cobra.Command, l *zap.SugaredLogger, output interface{}) {
	outputJSON, err := cmd.Flags().GetBool("json")
	if err != nil {
		l.Errorf("could not parse json global flag. Error: %s", err)
	}

	if !outputJSON {
		fmt.Println(output) //nolint:forbidigo
		return
	}

	out, err := json.Marshal(output)
	if err != nil {
		l.Error("Cannot unmarshal output to JSON")
		os.Exit(1)
	}

	fmt.Println(string(out)) //nolint:forbidigo
}
