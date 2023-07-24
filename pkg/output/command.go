// Package output provides utilities to print output in commands.
package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// PrintOutput prints output as a string or json.
func PrintOutput(cmd *cobra.Command, output interface{}) {
	outputJSON, err := cmd.Flags().GetBool("json")
	if err != nil {
		logrus.Errorf("could not parse json global flag. Error: %s", err)
	}

	if !outputJSON {
		fmt.Println(output) //nolint:forbidigo
		return
	}

	out, err := json.Marshal(output)
	if err != nil {
		logrus.Error("Cannot unmarshal output to JSON")
		os.Exit(1)
	}

	fmt.Println(string(out)) //nolint:forbidigo
}
