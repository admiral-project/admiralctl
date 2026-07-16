// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/admiral-project/admiral/admiralctl/internal/client"
	"github.com/admiral-project/admiral/admiralctl/internal/output"
	"github.com/admiral-project/admiral/admirald/pkg/admiral"
	"github.com/spf13/cobra"
)

var instancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "Manage provisioning, pausing, or deleting customer applications",
}

func init() {
	rootCmd.AddCommand(instancesCmd)
	instancesCmd.AddCommand(instancesListCmd)
	instancesCmd.AddCommand(instancesShowCmd)
	instancesCmd.AddCommand(instancesInspectCmd)
	instancesCmd.AddCommand(instancesProvisionCmd)
	instancesCmd.AddCommand(instancesCredentialsCmd)
	instancesCmd.AddCommand(instancesPauseCmd)
	instancesCmd.AddCommand(instancesResumeCmd)
	instancesCmd.AddCommand(instancesReactivateCmd)
	instancesCmd.AddCommand(instancesStartCmd)
	instancesCmd.AddCommand(instancesStopCmd)
	instancesCmd.AddCommand(instancesRestartCmd)
	instancesCmd.AddCommand(instancesBackupCmd)
	instancesCmd.AddCommand(instancesDeprovisionCmd)
	instancesCmd.AddCommand(instancesResizeCmd)
	instancesCmd.AddCommand(instancesMigrateCmd)
}

var instancesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List customer application instances",
	RunE:  runInstancesList,
}

var instancesShowCmd = &cobra.Command{
	Use:   "show <instance_id>",
	Short: "Show details for a specific instance",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstancesShow,
}

var instancesInspectCmd = &cobra.Command{
	Use:   "inspect <instance_id>",
	Short: "Trigger or show an inspect operation for an instance",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstancesInspect,
}

var instancesProvisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Provision a new customer application instance",
	RunE:  runInstancesProvision,
}

var instancesCredentialsCmd = &cobra.Command{
	Use:   "credentials <instance_id>",
	Short: "Show exposed credentials for an instance",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstancesCredentials,
}

var instancesPauseCmd = &cobra.Command{
	Use:   "pause <instance_id>",
	Short: "Pause an instance",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstancesAction("pause"),
}

var instancesResumeCmd = &cobra.Command{
	Use:   "resume <instance_id>",
	Short: "Resume an instance",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstancesAction("resume"),
}

var instancesReactivateCmd = &cobra.Command{
	Use:   "reactivate <instance_id>",
	Short: "Reactivate an instance",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstancesAction("reactivate"),
}

var instancesStartCmd = &cobra.Command{
	Use:   "start <instance_id>",
	Short: "Start an instance",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstancesAction("start"),
}

var instancesStopCmd = &cobra.Command{
	Use:   "stop <instance_id>",
	Short: "Stop an instance",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstancesDestructiveAction("stop"),
}

var instancesRestartCmd = &cobra.Command{
	Use:   "restart <instance_id>",
	Short: "Restart an instance",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstancesRestart,
}

var instancesBackupCmd = &cobra.Command{
	Use:   "backup <instance_id>",
	Short: "Trigger a backup for an instance",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstancesBackup,
}

var instancesDeprovisionCmd = &cobra.Command{
	Use:     "deprovision <instance_id>",
	Aliases: []string{"destroy"},
	Short:   "Deprovision an instance",
	Args:    cobra.ExactArgs(1),
	RunE:    runInstancesDestructiveAction("deprovision"),
}

var instancesResizeCmd = &cobra.Command{
	Use:   "resize <instance_id>",
	Short: "Resize an instance to a different tier",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstancesResize,
}

var instancesMigrateCmd = &cobra.Command{
	Use:   "migrate <instance_id>",
	Short: "Migrate an instance to another node",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstancesMigrate,
}

func init() {
	instancesListCmd.Flags().String("customer", "", "Filter by customer ID")
	instancesListCmd.Flags().String("output", "table", "Output format: table or json")
	instancesCredentialsCmd.Flags().String("output", "table", "Output format: table or json")

	instancesInspectCmd.Flags().Bool("result", false, "Show the last inspect result instead of triggering a new inspect")

	instancesProvisionCmd.Flags().String("app", "", "Name of application definition (required)")
	_ = instancesProvisionCmd.MarkFlagRequired("app")
	instancesProvisionCmd.Flags().String("tier", "", "Name of the service tier (required)")
	_ = instancesProvisionCmd.MarkFlagRequired("tier")
	instancesProvisionCmd.Flags().String("customer", "", "Unique Customer ID (required)")
	_ = instancesProvisionCmd.MarkFlagRequired("customer")
	instancesProvisionCmd.Flags().String("node", "", "Explicit node ID to target")
	instancesProvisionCmd.Flags().String("logical-instance-id", "", "Preserve logical instance identity for migration")
	instancesProvisionCmd.Flags().String("output", "table", "Output format: table or json")
	instancesProvisionCmd.Flags().Bool("wait", false, "Wait until the operation reaches a terminal state")
	instancesProvisionCmd.Flags().Bool("quiet", false, "Suppress credential output")

	addWaitAndForceFlags := func(cmd *cobra.Command) {
		cmd.Flags().Bool("wait", false, "Wait until the operation reaches a terminal state")
		cmd.Flags().Bool("force", false, "Skip confirmation prompt")
	}
	addWaitAndForceFlags(instancesPauseCmd)
	addWaitAndForceFlags(instancesResumeCmd)
	addWaitAndForceFlags(instancesReactivateCmd)
	addWaitAndForceFlags(instancesStartCmd)
	addWaitAndForceFlags(instancesStopCmd)
	instancesRestartCmd.Flags().Bool("force", false, "Skip confirmation prompt")
	instancesBackupCmd.Flags().Bool("wait", false, "Wait until the operation reaches a terminal state")
	instancesBackupCmd.Flags().String("service", "", "Service name for backup actions (required)")
	_ = instancesBackupCmd.MarkFlagRequired("service")
	addWaitAndForceFlags(instancesDeprovisionCmd)
	instancesResizeCmd.Flags().Bool("wait", false, "Wait until the operation reaches a terminal state")
	instancesResizeCmd.Flags().String("tier", "", "Target tier name (required)")
	_ = instancesResizeCmd.MarkFlagRequired("tier")
	instancesMigrateCmd.Flags().Bool("wait", false, "Wait until the operation completes")
	instancesMigrateCmd.Flags().String("target-node", "", "Target node ID (required)")
	_ = instancesMigrateCmd.MarkFlagRequired("target-node")
}

func runInstancesList(cmd *cobra.Command, _ []string) error {
	customerID, _ := cmd.Flags().GetString("customer")
	apps, err := clientOrNil().GetCustomerApps(customerID)
	if err != nil {
		return err
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(apps)
		return nil
	}

	headers := []string{"INSTANCE ID", "CUSTOMER ID", "APP TEMPLATE", "TIER", "NODE", "COMMERCIAL", "TECHNICAL", "STORAGE"}
	var rows [][]string
	for _, a := range apps {
		node := "-"
		if a["node_id"] != nil {
			node = fmt.Sprintf("%v", a["node_id"])
		}
		storageState := fmt.Sprintf("%v", a["storage_state"])
		if storageState == "" || storageState == "<nil>" {
			storageState = "-"
		}
		if v, ok := a["storage_exceeded"]; ok {
			if fmt.Sprintf("%v", v) == "true" {
				storageState = "EXCEEDED"
			}
		}
		rows = append(rows, []string{
			fmt.Sprintf("%v", a["id"]),
			fmt.Sprintf("%v", a["customer_id"]),
			fmt.Sprintf("%v", a["app_definition_name"]),
			fmt.Sprintf("%v", a["tier_name"]),
			node,
			fmt.Sprintf("%v", a["commercial_status"]),
			fmt.Sprintf("%v", a["technical_status"]),
			storageState,
		})
	}
	output.PrintTable(headers, rows)
	return nil
}

func runInstancesShow(cmd *cobra.Command, args []string) error {
	instance, err := clientOrNil().GetCustomerApp(args[0])
	if err != nil {
		return err
	}
	output.PrintJSON(instance)
	return nil
}

func runInstancesCredentials(cmd *cobra.Command, args []string) error {
	credentials, err := clientOrNil().GetCredentials(args[0])
	if err != nil {
		return err
	}

	outputFlag, _ := cmd.Flags().GetString("output")
	if outputFlag == "json" {
		output.PrintJSON(credentials)
		return nil
	}

	if len(credentials) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No credentials exposed for this instance.")
		return nil
	}

	for _, cred := range credentials {
		if cred.Kind == "notice" {
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", cred.Name, cred.Value)
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s.%s: %s\n", cred.Service, cred.Name, cred.Value)
	}
	return nil
}

func runInstancesInspect(cmd *cobra.Command, args []string) error {
	showResult, _ := cmd.Flags().GetBool("result")
	if showResult {
		result, err := clientOrNil().GetInspectResult(args[0])
		if err != nil {
			return err
		}
		output.PrintJSON(result)
		return nil
	}
	opID, err := clientOrNil().TriggerInspect(args[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Inspect task queued: operation_id=%s\n", opID)
	return nil
}

func runInstancesProvision(cmd *cobra.Command, _ []string) error {
	app, _ := cmd.Flags().GetString("app")
	tier, _ := cmd.Flags().GetString("tier")
	customer, _ := cmd.Flags().GetString("customer")
	nodeID, _ := cmd.Flags().GetString("node")
	logicalInstanceID, _ := cmd.Flags().GetString("logical-instance-id")
	outputFlag, _ := cmd.Flags().GetString("output")
	wait, _ := cmd.Flags().GetBool("wait")
	quiet, _ := cmd.Flags().GetBool("quiet")

	req := admiral.ProvisionRequest{
		AppDefinitionName: app,
		TierName:          tier,
		CustomerID:        customer,
		NodeID:            nodeID,
		LogicalInstanceID: logicalInstanceID,
	}

	res, err := clientOrNil().ProvisionApp(req)
	if err != nil {
		var rejected *client.ProvisionRejectedError
		if errors.As(err, &rejected) {
			if outputFlag == "json" {
				output.PrintJSON(rejected.Response)
			} else {
				printPolicyRejected(rejected.Response)
			}
			return fmt.Errorf("provision rejected")
		}
		return err
	}

	if outputFlag == "json" {
		output.PrintJSON(res)
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Provisioning queued successfully!\nOperation ID: %s\nRun 'admiralctl operations show %s' to monitor status.\n", res.OperationID, res.OperationID)
	if wait {
		op, err := waitForOperation(cmd, res.OperationID)
		if err != nil {
			return err
		}
		instanceID, _ := op["instance_id"].(string)
		if instanceID != "" {
			printFinalAccessData(cmd, instanceID)
		}
		return nil
	}
	if !quiet && len(res.Credentials) > 0 {
		printProvisionAccessData(cmd, res.Credentials)
	}
	return nil
}

func printFinalAccessData(cmd *cobra.Command, instanceID string) {
	instance, err := clientOrNil().GetCustomerApp(instanceID)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not fetch instance details: %v\n", err)
	}
	if hostname, ok := instance["hostname"].(string); ok && hostname != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Hostname: %s\n", hostname)
	}

	credentials, err := clientOrNil().GetCredentials(instanceID)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not fetch credentials: %v\n", err)
		return
	}
	if len(credentials) == 0 {
		return
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Access credentials:")
	printProvisionAccessData(cmd, credentials)
}

func printProvisionAccessData(cmd *cobra.Command, credentials []admiral.Credential) {
	credentialCount := 0
	for _, cred := range credentials {
		if cred.Kind != "notice" {
			credentialCount++
		}
	}
	if credentialCount > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Initial credentials:")
		for _, cred := range credentials {
			if cred.Kind == "notice" {
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  %s.%s: %s\n", cred.Service, cred.Name, cred.Value)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Warning: save these credentials securely. They are displayed only once.")
	}
	for _, cred := range credentials {
		if cred.Kind == "notice" {
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", cred.Name, cred.Value)
		}
	}
}

func runInstancesAction(action string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		wait, _ := cmd.Flags().GetBool("wait")
		opID, err := clientOrNil().TriggerAction(args[0], action)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Action %s queued successfully!\nOperation ID: %s\n", action, opID)
		if wait {
			if _, err := waitForOperation(cmd, opID); err != nil {
				return err
			}
		}
		return nil
	}
}

func runInstancesDestructiveAction(action string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if !confirmDestructive(cmd, action, fmt.Sprintf("instance %q", args[0])) {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}
		return runInstancesAction(action)(cmd, args)
	}
}

func runInstancesRestart(cmd *cobra.Command, args []string) error {
	if !confirmDestructive(cmd, "restart", fmt.Sprintf("instance %q", args[0])) {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}
	stopOpID, err := clientOrNil().TriggerAction(args[0], "stop")
	if err != nil {
		return err
	}
	startOpID, err := clientOrNil().TriggerAction(args[0], "start")
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Instance %s restarted (stop: %s / start: %s)\n", args[0], stopOpID, startOpID)
	return nil
}

func runInstancesBackup(cmd *cobra.Command, args []string) error {
	service, _ := cmd.Flags().GetString("service")
	wait, _ := cmd.Flags().GetBool("wait")
	opID, err := clientOrNil().TriggerActionWithService(args[0], "backup", service)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Action backup queued successfully!\nOperation ID: %s\n", opID)
	if wait {
		if _, err := waitForOperation(cmd, opID); err != nil {
			return err
		}
	}
	return nil
}

func runInstancesResize(cmd *cobra.Command, args []string) error {
	tier, _ := cmd.Flags().GetString("tier")
	wait, _ := cmd.Flags().GetBool("wait")

	if !confirmDestructive(cmd, "resize", fmt.Sprintf("instance %q to tier %q", args[0], tier)) {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}

	opID, err := clientOrNil().TriggerActionWithTier(args[0], "resize", tier)
	if err != nil {
		var rejected *client.ProvisionRejectedError
		if errors.As(err, &rejected) {
			printPolicyRejected(rejected.Response)
			return fmt.Errorf("resize rejected")
		}
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Resize queued successfully!\nOperation ID: %s\n", opID)
	if wait {
		if _, err := waitForOperation(cmd, opID); err != nil {
			return err
		}
	}
	return nil
}

func runInstancesMigrate(cmd *cobra.Command, args []string) error {
	targetNode, _ := cmd.Flags().GetString("target-node")
	wait, _ := cmd.Flags().GetBool("wait")

	if !confirmDestructive(cmd, "migrate", fmt.Sprintf("instance %q to node %q", args[0], targetNode)) {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}

	res, err := clientOrNil().MigrateInstance(args[0], targetNode)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Migration started!\nOperation ID: %s\nInstance ID: %s\nLogical Instance ID: %s\n",
		res.OperationID, res.InstanceID, res.LogicalInstanceID)
	if wait {
		if _, err := waitForOperation(cmd, res.OperationID); err != nil {
			return err
		}
	}
	return nil
}

func waitForOperation(cmd *cobra.Command, operationID string) (map[string]interface{}, error) {
	op, err := clientOrNil().WaitForOperation(operationID, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("wait for operation: %w", err)
	}
	status := fmt.Sprintf("%v", op["status"])
	fmt.Fprintf(cmd.OutOrStdout(), "Operation %s finished with status: %s\n", operationID, status)
	if status != "succeeded" {
		output.PrintJSON(op)
		return op, fmt.Errorf("operation %s finished with status %s", operationID, status)
	}
	return op, nil
}
