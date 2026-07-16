// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/admiral-project/admiral/admiralctl/internal/output"
	"github.com/spf13/cobra"
)

var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "List, show, sync, enable, or disable public routes",
}

func init() {
	rootCmd.AddCommand(routesCmd)
	routesCmd.AddCommand(routesListCmd)
	routesCmd.AddCommand(routesShowCmd)
	routesCmd.AddCommand(routesSyncCmd)
	routesCmd.AddCommand(routesEnableCmd)
	routesCmd.AddCommand(routesDisableCmd)
}

var routesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List public routes",
	RunE:  runRoutesList,
}

var routesShowCmd = &cobra.Command{
	Use:   "show <hostname>",
	Short: "Show details for a specific route",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoutesShow,
}

var routesSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize routes with the control plane",
	RunE:  runRoutesSync,
}

var routesEnableCmd = &cobra.Command{
	Use:   "enable <hostname>",
	Short: "Enable a public route",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoutesEnable,
}

var routesDisableCmd = &cobra.Command{
	Use:   "disable <hostname>",
	Short: "Disable a public route",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoutesDisable,
}

func init() {
	routesListCmd.Flags().String("output", "table", "Output format: table or json")
	routesDisableCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

func runRoutesList(cmd *cobra.Command, _ []string) error {
	routes, err := clientOrNil().GetRoutes()
	if err != nil {
		return err
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(cmd.OutOrStdout(), routes)
		return nil
	}

	headers := []string{"HOSTNAME", "KIND", "INSTANCE", "SERVICE", "TARGET", "STATUS"}
	var rows [][]string
	for _, route := range routes {
		target := fmt.Sprintf("%v", route["target_url"])
		if target == "" {
			target = fmt.Sprintf("%v:%v", route["target_host"], route["target_port"])
		}
		rows = append(rows, []string{
			fmt.Sprintf("%v", route["hostname"]),
			fmt.Sprintf("%v", route["route_kind"]),
			fmt.Sprintf("%v", route["app_instance_id"]),
			fmt.Sprintf("%v", route["service_name"]),
			target,
			fmt.Sprintf("%v", route["status"]),
		})
	}
	output.PrintTable(cmd.OutOrStdout(), headers, rows)
	return nil
}

func runRoutesShow(cmd *cobra.Command, args []string) error {
	route, err := clientOrNil().GetRoute(args[0])
	if err != nil {
		return fmt.Errorf("retrieve route: %w", err)
	}
	output.PrintJSON(cmd.OutOrStdout(), route)
	return nil
}

func runRoutesSync(cmd *cobra.Command, _ []string) error {
	if err := clientOrNil().SyncRoutes(); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Routes synchronized successfully.")
	return nil
}

func runRoutesEnable(cmd *cobra.Command, args []string) error {
	if err := clientOrNil().EnableRoute(args[0]); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Route %s successfully.\n", args[0])
	return nil
}

func runRoutesDisable(cmd *cobra.Command, args []string) error {
	if !confirmDestructive(cmd, "disable", fmt.Sprintf("route %q", args[0])) {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}
	if err := clientOrNil().DisableRoute(args[0]); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Route %s successfully.\n", args[0])
	return nil
}
