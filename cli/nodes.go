// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"strings"

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
	nodesCmd.AddCommand(nodesRemoveCmd)
	nodesCmd.AddCommand(nodesReadyCmd)
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

var nodesRemoveCmd = &cobra.Command{
	Use:   "remove <node_id>",
	Short: "Remove a registered node",
	Long: `Remove a node from the platform.

This removes the node record, its routes, backups, and customer apps
from the database. The operation will be refused if the node has
active instances, unless --force is used.`,
	Args: cobra.ExactArgs(1),
	RunE: runNodesRemove,
}

var nodesDisableCmd = &cobra.Command{
	Use:   "disable <node_id>",
	Short: "Disable a worker node",
	Args:  cobra.ExactArgs(1),
	RunE:  runNodesDisable,
}

var nodesReadyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Check if a worker node is ready and reachable",
	Long: `Check the reachability and readiness of a registered worker node.

The node agent must respond to the readiness probe to be considered ready.`,
	RunE: runNodesReady,
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
	nodesRegisterCmd.Flags().String("token", "", "Pre-generated node token (prefer ADMIRAL_NODE_TOKEN or the secure prompt)")

	nodesEnableCmd.Flags().Bool("force", false, "Skip confirmation prompt")
	nodesDisableCmd.Flags().Bool("force", false, "Skip confirmation prompt")
	nodesRemoveCmd.Flags().Bool("force", false, "Skip confirmation prompt and remove even with active instances")

	nodesReadyCmd.Flags().String("node", "", "Node ID (required)")
	_ = nodesReadyCmd.MarkFlagRequired("node")
}

func runNodesList(cmd *cobra.Command, _ []string) error {
	nodes, err := clientOrNil().GetNodes()
	if err != nil {
		return err
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(cmd.OutOrStdout(), nodes)
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
	output.PrintTable(cmd.OutOrStdout(), headers, rows)
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
	tokenFlag, _ := cmd.Flags().GetString("token")
	token := resolveToken(cmd, tokenFlag, os.Getenv("ADMIRAL_NODE_TOKEN"))

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
	output.PrintJSON(cmd.OutOrStdout(), node)
	return nil
}

func runNodesEnable(cmd *cobra.Command, args []string) error {
	if err := clientOrNil().EnableNode(args[0]); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Node %q enabled.\n", args[0])
	return nil
}

func runNodesRemove(cmd *cobra.Command, args []string) error {
	if !confirmDestructive(cmd, "remove", fmt.Sprintf("node %q", args[0])) {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}
	force, _ := cmd.Flags().GetBool("force")
	if err := clientOrNil().RemoveNode(args[0]); err != nil {
		if force {
			return err
		}
		if strings.Contains(err.Error(), "has active instance") {
			cmd.PrintErrln("Tip: use --force to remove the node and all associated resources.")
		}
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Node %q removed.\n", args[0])
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

func runNodesReady(cmd *cobra.Command, _ []string) error {
	nodeID, _ := cmd.Flags().GetString("node")

	result, err := clientOrNil().NodeReady(nodeID)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "✗ node %s unreachable: %s\n", nodeID, err.Error())
		os.Exit(1)
	}

	role := fmt.Sprintf("%v", result["role"])
	if role == "" || role == "<nil>" {
		role = "worker"
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ node %s (%s) ready\n", nodeID, role)
	output.PrintJSON(cmd.OutOrStdout(), result)
	return nil
}
