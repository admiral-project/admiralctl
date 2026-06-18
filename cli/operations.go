// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/admiral-project/admiral/admiralctl/internal/output"
	"github.com/spf13/cobra"
)

var operationsCmd = &cobra.Command{
	Use:   "operations",
	Short: "Query states of background PaaS operations",
}

func init() {
	rootCmd.AddCommand(operationsCmd)
	operationsCmd.AddCommand(operationsListCmd)
	operationsCmd.AddCommand(operationsShowCmd)
	operationsCmd.AddCommand(operationsRetryCmd)

	// Legacy singular alias: "admiralctl operation status <id>".
	operationCmd := &cobra.Command{
		Use:     "operation",
		Short:   "Query one operation status directly",
		Aliases: []string{},
	}
	operationStatusCmd := &cobra.Command{
		Use:   "status <operation_id>",
		Short: "Show operation status",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperationsShow,
	}
	operationCmd.AddCommand(operationStatusCmd)
	rootCmd.AddCommand(operationCmd)
}

var operationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List operations",
	RunE:  runOperationsList,
}

var operationsShowCmd = &cobra.Command{
	Use:   "show <operation_id>",
	Short: "Show details for a specific operation",
	Args:  cobra.ExactArgs(1),
	RunE:  runOperationsShow,
}

var operationsRetryCmd = &cobra.Command{
	Use:   "retry <operation_id>",
	Short: "Retry a failed operation",
	Args:  cobra.ExactArgs(1),
	RunE:  runOperationsRetry,
}

func init() {
	operationsListCmd.Flags().String("output", "table", "Output format: table or json")
}

func runOperationsList(cmd *cobra.Command, _ []string) error {
	ops, err := clientOrNil().GetOperations()
	if err != nil {
		return err
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(ops)
		return nil
	}

	headers := []string{"OPERATION ID", "INSTANCE ID", "ACTION", "STATUS", "LAST UPDATE"}
	var rows [][]string
	for _, o := range ops {
		rows = append(rows, []string{
			fmt.Sprintf("%v", o["id"]),
			fmt.Sprintf("%v", o["instance_id"]),
			fmt.Sprintf("%v", o["action"]),
			fmt.Sprintf("%v", o["status"]),
			fmt.Sprintf("%v", o["updated_at"]),
		})
	}
	output.PrintTable(headers, rows)
	return nil
}

func runOperationsShow(cmd *cobra.Command, args []string) error {
	op, err := clientOrNil().GetOperation(args[0])
	if err != nil {
		return fmt.Errorf("retrieve operation: %w", err)
	}
	output.PrintJSON(op)
	return nil
}

func runOperationsRetry(cmd *cobra.Command, args []string) error {
	res, err := clientOrNil().RetryOperation(args[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Operation %s retried.\n", res["operation_id"])
	return nil
}
