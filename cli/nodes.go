// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/admiral-project/admiral/admiralctl/internal/output"
	"github.com/admiral-project/admiral/admirald/pkg/admiral"
	"github.com/spf13/cobra"
)

var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "List or register worker nodes",
}

func init() {
	rootCmd.AddCommand(nodesCmd)
	nodesCmd.AddCommand(nodesListCmd)
	nodesCmd.AddCommand(nodesRegisterCmd)
	nodesCmd.AddCommand(nodesShowCmd)
	nodesCmd.AddCommand(nodesEnableCmd)
	nodesCmd.AddCommand(nodesDisableCmd)
}

var nodesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered worker nodes",
	RunE:  runNodesList,
}

var nodesRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new worker node",
	RunE:  runNodesRegister,
}

var nodesShowCmd = &cobra.Command{
	Use:   "show <node_id>",
	Short: "Show details for a specific node",
	Args:  cobra.ExactArgs(1),
	RunE:  runNodesShow,
}

var nodesEnableCmd = &cobra.Command{
	Use:   "enable <node_id>",
	Short: "Enable a worker node",
	Args:  cobra.ExactArgs(1),
	RunE:  runNodesEnable,
}

var nodesDisableCmd = &cobra.Command{
	Use:   "disable <node_id>",
	Short: "Disable a worker node",
	Args:  cobra.ExactArgs(1),
	RunE:  runNodesDisable,
}

func init() {
	nodesListCmd.Flags().String("output", "table", "Output format: table or json")

	nodesRegisterCmd.Flags().String("id", "", "Unique Node ID (required)")
	_ = nodesRegisterCmd.MarkFlagRequired("id")
	nodesRegisterCmd.Flags().String("hostname", "", "Node hostname (required)")
	_ = nodesRegisterCmd.MarkFlagRequired("hostname")
	nodesRegisterCmd.Flags().String("ip", "", "Node IP address (required)")
	_ = nodesRegisterCmd.MarkFlagRequired("ip")
	nodesRegisterCmd.Flags().String("wireguard-ip", "", "WireGuard VPN IP address")
	nodesRegisterCmd.Flags().String("role", "worker", "Node role: admin, worker, or portal")
	nodesRegisterCmd.Flags().String("public-ip", "", "Public IP address for remote connectivity")
	nodesRegisterCmd.Flags().String("os", "linux", "Operating System")
	nodesRegisterCmd.Flags().String("podman", "4.9.0", "Podman Version")
	nodesRegisterCmd.Flags().String("token", "", "Pre-generated node token for single-node mode")

	nodesEnableCmd.Flags().Bool("force", false, "Skip confirmation prompt")
	nodesDisableCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

func runNodesList(cmd *cobra.Command, _ []string) error {
	nodes, err := clientOrNil().GetNodes()
	if err != nil {
		return err
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(nodes)
		return nil
	}

	headers := []string{"NODE ID", "HOSTNAME", "ROLE", "STATUS", "HEALTH", "AVAILABLE", "WG IP", "PUBLIC IP"}
	var rows [][]string
	for _, n := range nodes {
		health := fmt.Sprintf("%v", n["health_status"])
		if health == "" || health == "<nil>" {
			health = "-"
		}
		avail := fmt.Sprintf("%v", n["available_for_provisioning"])
		if avail == "" || avail == "<nil>" {
			avail = "-"
		}
		role := fmt.Sprintf("%v", n["node_role"])
		if role == "" || role == "<nil>" {
			role = "worker"
		}
		wgIP := fmt.Sprintf("%v", n["wireguard_ip"])
		if wgIP == "" || wgIP == "<nil>" {
			wgIP = "-"
		}
		pubIP := fmt.Sprintf("%v", n["public_ip"])
		if pubIP == "" || pubIP == "<nil>" {
			pubIP = "-"
		}
		rows = append(rows, []string{
			fmt.Sprintf("%v", n["id"]),
			fmt.Sprintf("%v", n["hostname"]),
			role,
			fmt.Sprintf("%v", n["status"]),
			health,
			avail,
			wgIP,
			pubIP,
		})
	}
	output.PrintTable(headers, rows)
	return nil
}

func runNodesRegister(cmd *cobra.Command, _ []string) error {
	id, _ := cmd.Flags().GetString("id")
	hostname, _ := cmd.Flags().GetString("hostname")
	ip, _ := cmd.Flags().GetString("ip")
	wgIP, _ := cmd.Flags().GetString("wireguard-ip")
	role, _ := cmd.Flags().GetString("role")
	publicIP, _ := cmd.Flags().GetString("public-ip")
	osType, _ := cmd.Flags().GetString("os")
	podmanV, _ := cmd.Flags().GetString("podman")
	token, _ := cmd.Flags().GetString("token")

	req := admiral.RegisterNodeRequest{
		NodeID:      id,
		Hostname:    hostname,
		IP:          ip,
		WireguardIP: wgIP,
		NodeRole:    role,
		PublicIP:    publicIP,
		OS:          osType,
		PodmanV:     podmanV,
		Token:       token,
	}

	if err := clientOrNil().RegisterNode(req); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Node %q registered successfully!\n", id)
	return nil
}

func runNodesShow(cmd *cobra.Command, args []string) error {
	node, err := clientOrNil().GetNode(args[0])
	if err != nil {
		return err
	}
	output.PrintJSON(node)
	return nil
}

func runNodesEnable(cmd *cobra.Command, args []string) error {
	if err := clientOrNil().EnableNode(args[0]); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Node %q enabled.\n", args[0])
	return nil
}

func runNodesDisable(cmd *cobra.Command, args []string) error {
	if !confirmDestructive(cmd, "disable", fmt.Sprintf("node %q", args[0])) {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}
	if err := clientOrNil().DisableNode(args[0]); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Node %q disabled.\n", args[0])
	return nil
}
