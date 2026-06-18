// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/admiral-project/admiral/admiralctl/internal/output"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get status details from control plane",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().String("output", "table", "Output format: table or json")
}

func runStatus(cmd *cobra.Command, _ []string) error {
	status, err := clientOrNil().GetStatus()
	if err != nil {
		return fmt.Errorf("contact control plane: %w", err)
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(status)
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Admiral PaaS Status:")
	fmt.Fprintf(cmd.OutOrStdout(), "  API connection:   online\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  Control Plane:    %v\n", status["status"])
	return nil
}
