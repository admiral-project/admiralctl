// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/admiral-project/admiral/admiralctl/internal/output"
	"github.com/spf13/cobra"
)

var storageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Show storage state for instances and nodes",
}

func init() {
	rootCmd.AddCommand(storageCmd)
	storageCmd.AddCommand(storageInstancesCmd)
	storageCmd.AddCommand(storageNodesCmd)
}

var storageInstancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "Show storage state for instances",
	RunE:  runStorageInstances,
}

var storageNodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Show storage state for nodes",
	RunE:  runStorageNodes,
}

func init() {
	storageInstancesCmd.Flags().String("output", "table", "Output format: table or json")
	storageNodesCmd.Flags().String("output", "table", "Output format: table or json")
}

func runStorageInstances(cmd *cobra.Command, _ []string) error {
	apps, err := clientOrNil().GetCustomerApps("")
	if err != nil {
		return err
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(apps)
		return nil
	}

	headers := []string{"INSTANCE ID", "APP", "STATE", "USED", "LIMIT", "USED%", "GRACE ENDS"}
	var rows [][]string
	for _, a := range apps {
		storageState := fmt.Sprintf("%v", a["storage_state"])
		if storageState == "" || storageState == "<nil>" {
			storageState = "-"
		}
		if v, ok := a["storage_exceeded"]; ok {
			if fmt.Sprintf("%v", v) == "true" {
				storageState = "EXCEEDED"
			}
		}
		used := "-"
		if u, ok := a["storage_used_bytes"]; ok && fmt.Sprintf("%v", u) != "0" {
			used = fmt.Sprintf("%v", u)
		}
		limit := "-"
		if l, ok := a["storage_limit_bytes"]; ok && fmt.Sprintf("%v", l) != "0" {
			limit = fmt.Sprintf("%v", l)
		}
		pct := "-"
		if p, ok := a["storage_used_percent"]; ok && fmt.Sprintf("%v", p) != "0" {
			pct = fmt.Sprintf("%.1f%%", p)
		}
		graceEnds := "-"
		if g, ok := a["grace_period_ends_at"]; ok && g != nil {
			graceEnds = fmt.Sprintf("%v", g)
		}
		rows = append(rows, []string{
			fmt.Sprintf("%v", a["id"]),
			fmt.Sprintf("%v", a["app_definition_name"]),
			storageState,
			used,
			limit,
			pct,
			graceEnds,
		})
	}
	output.PrintTable(headers, rows)
	return nil
}

func runStorageNodes(cmd *cobra.Command, _ []string) error {
	nodes, err := clientOrNil().GetNodes()
	if err != nil {
		return err
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(nodes)
		return nil
	}

	headers := []string{"NODE ID", "HOSTNAME", "DISK USED", "DISK TOTAL", "RAM USED", "RAM TOTAL", "RAM COMMIT", "DISK COMMIT", "HEALTH", "STORAGE STATE"}
	var rows [][]string
	for _, n := range nodes {
		diskUsed := "-"
		if d, ok := n["disk_used_bytes"]; ok && fmt.Sprintf("%v", d) != "0" {
			diskUsed = fmt.Sprintf("%v", d)
		}
		diskTotal := "-"
		if d, ok := n["disk_total_bytes"]; ok && fmt.Sprintf("%v", d) != "0" {
			diskTotal = fmt.Sprintf("%v", d)
		}
		ramUsed := "-"
		if r, ok := n["ram_used_bytes"]; ok && fmt.Sprintf("%v", r) != "0" {
			ramUsed = fmt.Sprintf("%v", r)
		}
		ramTotal := "-"
		if r, ok := n["ram_total_bytes"]; ok && fmt.Sprintf("%v", r) != "0" {
			ramTotal = fmt.Sprintf("%v", r)
		}
		ramCommit := "-"
		if r, ok := n["committed_ram_bytes"]; ok && fmt.Sprintf("%v", r) != "0" {
			ramCommit = fmt.Sprintf("%v", r)
		}
		diskCommit := "-"
		if d, ok := n["committed_disk_bytes"]; ok && fmt.Sprintf("%v", d) != "0" {
			diskCommit = fmt.Sprintf("%v", d)
		}
		health := fmt.Sprintf("%v", n["health_status"])
		if health == "" || health == "<nil>" {
			health = "-"
		}
		stState := fmt.Sprintf("%v", n["storage_state"])
		if stState == "" || stState == "<nil>" {
			stState = "ok"
		}
		rows = append(rows, []string{
			fmt.Sprintf("%v", n["id"]),
			fmt.Sprintf("%v", n["hostname"]),
			diskUsed,
			diskTotal,
			ramUsed,
			ramTotal,
			ramCommit,
			diskCommit,
			health,
			stState,
		})
	}
	output.PrintTable(headers, rows)
	return nil
}
