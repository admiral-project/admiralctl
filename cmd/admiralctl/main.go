// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/admiral-project/admiral/admiralctl/internal/client"
	"github.com/admiral-project/admiral/admiralctl/internal/config"
	"github.com/admiral-project/admiral/admiralctl/internal/output"
	"github.com/admiral-project/admiral/admirald/pkg/admiral"
	"github.com/admiral-project/admiral/admirald/pkg/admiral/tlsconfig"
	"gopkg.in/yaml.v2"
)

var Version = "dev"

func main() {
	if len(os.Args) < 2 {
		printGeneralUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("admiralctl %s\n", Version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	cli, err := client.New(cfg.ServerURL, cfg.Token, cfg.CACertFile, client.WithOperator(cfg.Operator))
	if err != nil {
		fmt.Printf("Error creating TLS client: %v\n", err)
		os.Exit(1)
	}

	subcommand := os.Args[1]
	switch subcommand {
	case "init":
		handleInit(cfg)
	case "status":
		handleStatus(cli)
	case "nodes":
		requireToken(cfg)
		handleNodes(cli)
	case "apps":
		requireToken(cfg)
		handleApps(cli)
	case "instances":
		requireToken(cfg)
		handleInstances(cli)
	case "operations":
		requireToken(cfg)
		handleOperations(cli)
	case "operation":
		requireToken(cfg)
		handleOperation(cli)
	case "backups":
		requireToken(cfg)
		handleBackups(cli)
	case "routes":
		requireToken(cfg)
		handleRoutes(cli)
	case "user":
		requireToken(cfg)
		handleUser(cli)
	case "storage":
		requireToken(cfg)
		handleStorage(cli)
	case "help":
		printGeneralUsage()
	default:
		fmt.Printf("Unknown command %q. Run 'admiralctl help' for instructions.\n", subcommand)
		os.Exit(1)
	}
}

func printGeneralUsage() {
	fmt.Println("Admiral official CLI - admiralctl")
	fmt.Println("\nUsage:")
	fmt.Println("  admiralctl <command> [subcommand] [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  init         Initialize or update local CLI settings")
	fmt.Println("  status       Get status details from control plane")
	fmt.Println("  nodes        List or register worker nodes")
	fmt.Println("  apps         Manage and apply application definition templates")
	fmt.Println("  instances    Manage provisioning, pausing, or deleting customer applications")
	fmt.Println("  operations   Query states of background PaaS operations")
	fmt.Println("  operation    Query one operation status directly")
	fmt.Println("  backups      List, show, restore, storage config, delete, and prune backups")
	fmt.Println("  routes       List, show, sync, enable, or disable public routes")
	fmt.Println("  storage      Show storage state for instances and nodes")
	fmt.Println("  user         Manage admin users (create, list, set-password)")
	fmt.Println("  version      Print the CLI version")
	fmt.Println("  help         Show usage help details")
	fmt.Println("\nUse 'admiralctl <command> -h' or 'admiralctl <command> <subcommand> -h' for specific parameters.")
}

// --- Init Command ---

func resolveToken(tokenFlag, cfgToken string) string {
	if tokenFlag != "" {
		fmt.Fprintln(os.Stderr, "Warning: --token exposes the secret in the process list. Prefer ADMIRAL_ADMIN_TOKEN env var.")
		return tokenFlag
	}
	if cfgToken != "" {
		return cfgToken
	}
	fmt.Print("Enter admin token: ")
	t, err := readPassword()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nFailed to read token: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
	return strings.TrimSpace(t)
}

func handleInit(cfg *config.Config) {
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	serverURL := initCmd.String("server", cfg.ServerURL, "Control plane server endpoint URL")
	token := initCmd.String("token", "", "Authentication token (visible in process list; prefer ADMIRAL_ADMIN_TOKEN)")
	caCert := initCmd.String("ca-cert", cfg.CACertFile, "CA certificate file for admirald HTTPS validation")
	genSigningKey := initCmd.Bool("generate-signing-key", false, "Generate Ed25519 signing key pair for task verification")

	_ = initCmd.Parse(os.Args[2:])

	if *genSigningKey {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating signing key: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("private_key:" + hex.EncodeToString(priv.Seed()))
		fmt.Println("public_key:" + hex.EncodeToString(pub))
		return
	}

	resolved := resolveToken(*token, cfg.Token)
	if resolved == "" {
		fmt.Println("Error: authentication token is required. Use --token or export ADMIRAL_ADMIN_TOKEN.")
		os.Exit(1)
	}
	if err := tlsconfig.ValidateURLScheme(*serverURL, "https"); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	cfg.ServerURL = *serverURL
	cfg.Token = resolved
	cfg.CACertFile = *caCert

	err := config.Save(cfg)
	if err != nil {
		fmt.Printf("Failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration initialized successfully!\nSaved to: %s\nTarget server: %s\n", config.GetConfigPath(), cfg.ServerURL)
}

func printPolicyRejected(err *client.ProvisionRejectedError) {
	resp := err.Response
	fmt.Printf("Action blocked by policy.\nCode: %s\nMessage: %s\n", resp.Code, resp.Message)
	if resp.OperationID != "" {
		fmt.Printf("Operation ID: %s\n", resp.OperationID)
	}
	if resp.TaskID != "" {
		fmt.Printf("Task ID: %s\n", resp.TaskID)
	}
	if resp.RequestedNodeID != "" {
		fmt.Printf("Requested node: %s\n", resp.RequestedNodeID)
	}
	for _, evaluation := range resp.NodeEvaluations {
		state := "blocked"
		if evaluation.Eligible {
			state = "eligible"
		}
		fmt.Printf("Node %s: %s", evaluation.NodeID, state)
		if len(evaluation.RejectionReasons) > 0 {
			fmt.Printf(" (%s)", strings.Join(evaluation.RejectionReasons, ", "))
		}
		fmt.Println()
	}
}

func confirmDestructive(action, target string) bool {
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "--force" || os.Args[i] == "-f" {
			os.Args = append(os.Args[:i], os.Args[i+1:]...)
			return true
		}
	}
	fmt.Printf("Are you sure you want to %s %s? (y/N): ", action, target)
	var confirm string
	_, _ = fmt.Scanf("%s", &confirm)
	return strings.ToLower(strings.TrimSpace(confirm)) == "y"
}

func requireToken(cfg *config.Config) {
	if strings.TrimSpace(cfg.Token) != "" {
		return
	}

	fmt.Println("Error: no authentication token configured. Run 'admiralctl init --token <token>' or set ADMIRAL_ADMIN_TOKEN.")
	os.Exit(1)
}

// --- Status Command ---

func handleStatus(cli *client.Client) {
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	outputFlag := statusCmd.String("output", "table", "Output format: table or json")
	_ = statusCmd.Parse(os.Args[2:])

	status, err := cli.GetStatus()
	if err != nil {
		fmt.Printf("Failed to contact control plane: %v\n", err)
		os.Exit(1)
	}

	if *outputFlag == "json" {
		output.PrintJSON(status)
		return
	}

	fmt.Println("Admiral PaaS Status:")
	fmt.Printf("  API connection:   online\n")
	fmt.Printf("  Control Plane:    %v\n", status["status"])
}

// --- Nodes Command ---

func handleNodes(cli *client.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: admiralctl nodes <list|register|show|enable|disable> [flags/args]")
		os.Exit(1)
	}

	action := os.Args[2]
	switch action {
	case "help", "-h", "--help":
		fmt.Println("Usage: admiralctl nodes <list|register|show|enable|disable> [flags/args]")
		return
	case "list":
		listCmd := flag.NewFlagSet("nodes list", flag.ExitOnError)
		outputFlag := listCmd.String("output", "table", "Output format: table or json")
		_ = listCmd.Parse(os.Args[3:])

		nodes, err := cli.GetNodes()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if *outputFlag == "json" {
			output.PrintJSON(nodes)
			return
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

	case "register":
		regCmd := flag.NewFlagSet("nodes register", flag.ExitOnError)
		id := regCmd.String("id", "", "Unique Node ID (required)")
		host := regCmd.String("hostname", "", "Node hostname (required)")
		ip := regCmd.String("ip", "", "Node IP address (required)")
		wgIP := regCmd.String("wireguard-ip", "", "WireGuard VPN IP address")
		role := regCmd.String("role", "worker", "Node role: admin, worker, or portal")
		publicIP := regCmd.String("public-ip", "", "Public IP address for remote connectivity")
		osType := regCmd.String("os", "linux", "Operating System")
		podmanV := regCmd.String("podman", "4.9.0", "Podman Version")
		token := regCmd.String("token", "", "Pre-generated node token for single-node mode")

		_ = regCmd.Parse(os.Args[3:])

		if *id == "" || *host == "" || *ip == "" {
			fmt.Println("Error: --id, --hostname, and --ip are mandatory fields.")
			regCmd.Usage()
			os.Exit(1)
		}

		req := admiral.RegisterNodeRequest{
			NodeID:      *id,
			Hostname:    *host,
			IP:          *ip,
			WireguardIP: *wgIP,
			NodeRole:    *role,
			PublicIP:    *publicIP,
			OS:          *osType,
			PodmanV:     *podmanV,
			Token:       *token,
		}

		err := cli.RegisterNode(req)
		if err != nil {
			fmt.Printf("Failed registering node: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Node %q registered successfully!\n", *id)

	case "show":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl nodes show <node_id>")
			os.Exit(1)
		}
		nodeID := os.Args[3]
		node, err := cli.GetNode(nodeID)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		output.PrintJSON(node)

	case "enable":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl nodes enable <node_id>")
			os.Exit(1)
		}
		if err := cli.EnableNode(os.Args[3]); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Node %q enabled.\n", os.Args[3])

	case "disable":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl nodes disable <node_id>")
			os.Exit(1)
		}
		if !confirmDestructive("disable", fmt.Sprintf("node %q", os.Args[3])) {
			fmt.Println("Cancelled.")
			return
		}
		if err := cli.DisableNode(os.Args[3]); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Node %q disabled.\n", os.Args[3])

	default:
		fmt.Printf("Unknown action %q for nodes.\n", action)
		os.Exit(1)
	}
}

// --- Apps Command ---

func handleApps(cli *client.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: admiralctl apps <list|show|apply|validate|activate|deactivate> [flags/args]")
		os.Exit(1)
	}

	action := os.Args[2]
	switch action {
	case "help", "-h", "--help":
		fmt.Println("Usage: admiralctl apps <list|show|apply|validate|activate|deactivate> [flags/args]")
		return
	case "show":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl apps show <app_name>")
			os.Exit(1)
		}
		app, err := cli.GetApp(os.Args[3])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		output.PrintJSON(app)

	case "list":
		listCmd := flag.NewFlagSet("apps list", flag.ExitOnError)
		outputFlag := listCmd.String("output", "table", "Output format: table or json")
		_ = listCmd.Parse(os.Args[3:])

		apps, err := cli.GetApps()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if *outputFlag == "json" {
			output.PrintJSON(apps)
			return
		}

		headers := []string{"APP NAME", "DISPLAY NAME", "STATUS", "CREATED AT"}
		var rows [][]string
		for _, a := range apps {
			rows = append(rows, []string{
				fmt.Sprintf("%v", a["name"]),
				fmt.Sprintf("%v", a["display_name"]),
				fmt.Sprintf("%v", a["status"]),
				fmt.Sprintf("%v", a["created_at"]),
			})
		}
		output.PrintTable(headers, rows)

	case "apply":
		applyCmd := flag.NewFlagSet("apps apply", flag.ExitOnError)
		file := applyCmd.String("f", "", "Path to app definition YAML file (required)")
		_ = applyCmd.Parse(os.Args[3:])

		if *file == "" {
			fmt.Println("Error: --f is a required flag specifying the template YAML file.")
			os.Exit(1)
		}

		path, err := filepath.Abs(*file)
		if err != nil {
			fmt.Printf("Failed resolving file path %s: %v\n", *file, err)
			os.Exit(1)
		}
		path, err = sanitizeInputFilePath(path)
		if err != nil {
			fmt.Printf("Failed validating file path %s: %v\n", *file, err)
			os.Exit(1)
		}
		data, err := readInputFile(path)
		if err != nil {
			fmt.Printf("Failed reading file %s: %v\n", *file, err)
			os.Exit(1)
		}

		var payload admiral.AppDefinitionPayload
		if err := yaml.Unmarshal(data, &payload); err != nil {
			fmt.Printf("Error: YAML syntax validation failed: %v\n", err)
			os.Exit(1)
		}
		if err := admiral.ValidateAppDefinition(payload); err != nil {
			fmt.Printf("Error: application definition validation failed: %v\n", err)
			os.Exit(1)
		}

		name, err := cli.ApplyApp(string(data))
		if err != nil {
			fmt.Printf("Apply app failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Application definition %q applied successfully!\n", name)

	case "validate":
		validateCmd := flag.NewFlagSet("apps validate", flag.ExitOnError)
		file := validateCmd.String("f", "", "Path to app definition YAML file (required)")
		_ = validateCmd.Parse(os.Args[3:])

		if *file == "" {
			fmt.Println("Error: --f is required.")
			os.Exit(1)
		}

		path, err := filepath.Abs(*file)
		if err != nil {
			fmt.Printf("Failed resolving file path %s: %v\n", *file, err)
			os.Exit(1)
		}
		path, err = sanitizeInputFilePath(path)
		if err != nil {
			fmt.Printf("Failed validating file path %s: %v\n", *file, err)
			os.Exit(1)
		}
		data, err := readInputFile(path)
		if err != nil {
			fmt.Printf("Failed reading file: %v\n", err)
			os.Exit(1)
		}

		var payload admiral.AppDefinitionPayload
		if err := yaml.Unmarshal(data, &payload); err != nil {
			fmt.Printf("YAML Validation: FAILED\nReason: %v\n", err)
			os.Exit(1)
		}

		if err := admiral.ValidateAppDefinition(payload); err != nil {
			fmt.Printf("YAML Validation: FAILED\nReason: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("YAML Validation: PASSED\nApp: %s\nServices: %d\nTiers: %d\n", payload.Name, len(payload.Services), len(payload.Tiers))

	case "activate", "deactivate":
		statusCmd := flag.NewFlagSet("apps "+action, flag.ExitOnError)
		name := statusCmd.String("name", "", "Application definition name (required)")
		_ = statusCmd.Parse(os.Args[3:])

		if *name == "" {
			fmt.Println("Error: --name is required.")
			os.Exit(1)
		}

		targetStatus := "active"
		if action == "deactivate" {
			targetStatus = "inactive"
		}
		if action == "deactivate" && !confirmDestructive("deactivate", fmt.Sprintf("app definition %q", *name)) {
			fmt.Println("Cancelled.")
			return
		}
		if err := cli.UpdateAppStatus(*name, targetStatus); err != nil {
			fmt.Printf("Update app status failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Application definition %q is now %s.\n", *name, targetStatus)

	default:
		fmt.Printf("Unknown action %q for apps.\n", action)
		os.Exit(1)
	}
}

// --- Instances Command ---

func handleInstances(cli *client.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: admiralctl instances <list|show|inspect|provision|start|stop|restart|pause|resume|reactivate|backup|deprovision|destroy|resize|migrate> [flags/args]")
		os.Exit(1)
	}

	action := os.Args[2]
	switch action {
	case "help", "-h", "--help":
		fmt.Println("Usage: admiralctl instances <list|show|inspect|provision|start|stop|restart|pause|resume|reactivate|backup|deprovision|destroy|resize|migrate> [flags/args]")
		return
	case "show":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl instances show <instance_id>")
			os.Exit(1)
		}
		instance, err := cli.GetCustomerApp(os.Args[3])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		output.PrintJSON(instance)

	case "inspect":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl instances inspect <instance_id>")
			os.Exit(1)
		}
		if len(os.Args) >= 5 && os.Args[4] == "--result" {
			result, err := cli.GetInspectResult(os.Args[3])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			output.PrintJSON(result)
		} else {
			opID, err := cli.TriggerInspect(os.Args[3])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Inspect task queued: operation_id=%s\n", opID)
		}

	case "list":
		listCmd := flag.NewFlagSet("instances list", flag.ExitOnError)
		outputFlag := listCmd.String("output", "table", "Output format: table or json")
		_ = listCmd.Parse(os.Args[3:])

		apps, err := cli.GetCustomerApps()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if *outputFlag == "json" {
			output.PrintJSON(apps)
			return
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

	case "provision":
		provCmd := flag.NewFlagSet("instances provision", flag.ExitOnError)
		app := provCmd.String("app", "", "Name of application definition (required)")
		tier := provCmd.String("tier", "", "Name of the service tier (required)")
		cust := provCmd.String("customer", "", "Unique Customer ID (required)")
		nodeID := provCmd.String("node", "", "Explicit node ID to target")
		logicalInstanceID := provCmd.String("logical-instance-id", "", "Preserve logical instance identity for migration")
		outputFlag := provCmd.String("output", "table", "Output format: table or json")
		waitFlag := provCmd.Bool("wait", false, "Wait until the operation reaches a terminal state")
		quietFlag := provCmd.Bool("quiet", false, "Suppress credential output")

		_ = provCmd.Parse(os.Args[3:])

		if *app == "" || *tier == "" || *cust == "" {
			fmt.Println("Error: --app, --tier, and --customer are required parameters.")
			os.Exit(1)
		}

		req := admiral.ProvisionRequest{
			AppDefinitionName: *app,
			TierName:          *tier,
			CustomerID:        *cust,
			NodeID:            *nodeID,
			LogicalInstanceID: *logicalInstanceID,
		}

		res, err := cli.ProvisionApp(req)
		if err != nil {
			var rejected *client.ProvisionRejectedError
			if errors.As(err, &rejected) {
				if *outputFlag == "json" {
					output.PrintJSON(rejected.Response)
				} else {
					printPolicyRejected(rejected)
				}
				os.Exit(1)
			}
			fmt.Printf("Provision failed: %v\n", err)
			os.Exit(1)
		}

		if *outputFlag == "json" {
			output.PrintJSON(res)
			return
		}

		fmt.Printf("Provisioning queued successfully!\nOperation ID: %s\nRun 'admiralctl operations show %s' to monitor status.\n", res.OperationID, res.OperationID)
		if !*quietFlag && len(res.Credentials) > 0 {
			fmt.Println("Initial credentials:")
			for _, cred := range res.Credentials {
				fmt.Printf("  %s.%s: %s\n", cred.Service, cred.Name, cred.Value)
			}
			fmt.Println("Warning: save these credentials securely. They are displayed only once.")
		}
		if *waitFlag {
			waitForOperationOrExit(cli, res.OperationID)
		}

	case "pause", "resume", "reactivate", "start", "stop", "backup", "deprovision", "destroy":
		actionCmd := flag.NewFlagSet("instances "+action, flag.ExitOnError)
		service := actionCmd.String("service", "", "Service name for backup actions")
		waitFlag := actionCmd.Bool("wait", false, "Wait until the operation reaches a terminal state")
		_ = actionCmd.Parse(os.Args[3:])
		if actionCmd.NArg() < 1 {
			fmt.Printf("Usage: admiralctl instances %s <instance_id>\n", action)
			os.Exit(1)
		}
		instID := actionCmd.Arg(0)
		if action != "resume" && action != "reactivate" && action != "start" && action != "backup" {
			if !confirmDestructive(action, fmt.Sprintf("instance %q", instID)) {
				fmt.Println("Cancelled.")
				return
			}
		}
		apiAction := action
		if action == "destroy" {
			apiAction = "deprovision"
		}

		var (
			opID string
			err  error
		)
		if action == "backup" {
			if *service == "" {
				fmt.Println("Error: --service is required for backup.")
				os.Exit(1)
			}
			opID, err = cli.TriggerActionWithService(instID, apiAction, *service)
		} else {
			opID, err = cli.TriggerAction(instID, apiAction)
		}
		if err != nil {
			fmt.Printf("Action %s failed: %v\n", action, err)
			os.Exit(1)
		}

		fmt.Printf("Action %s queued successfully!\nOperation ID: %s\n", action, opID)
		if *waitFlag {
			waitForOperationOrExit(cli, opID)
		}

	case "restart":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl instances restart <instance_id>")
			os.Exit(1)
		}
		instID := os.Args[3]
		if !confirmDestructive("restart", fmt.Sprintf("instance %q", instID)) {
			fmt.Println("Cancelled.")
			return
		}
		stopOpID, err := cli.TriggerAction(instID, "stop")
		if err != nil {
			fmt.Printf("Restart stop phase failed: %v\n", err)
			os.Exit(1)
		}
		startOpID, err := cli.TriggerAction(instID, "start")
		if err != nil {
			fmt.Printf("Restart start phase failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Instance %s restarted (stop: %s / start: %s)\n", instID, stopOpID, startOpID)

	case "migrate":
		migrateCmd := flag.NewFlagSet("instances migrate", flag.ExitOnError)
		targetNode := migrateCmd.String("target-node", "", "Target node ID (required)")
		waitFlag := migrateCmd.Bool("wait", false, "Wait until migration completes")
		_ = migrateCmd.Parse(os.Args[3:])
		if migrateCmd.NArg() < 1 || *targetNode == "" {
			fmt.Println("Usage: admiralctl instances migrate --target-node <node_id> <instance_id>")
			os.Exit(1)
		}
		instID := migrateCmd.Arg(0)
		if !confirmDestructive("migrate", fmt.Sprintf("instance %q to node %q", instID, *targetNode)) {
			fmt.Println("Cancelled.")
			return
		}
		res, err := cli.MigrateInstance(instID, *targetNode)
		if err != nil {
			fmt.Printf("Migration failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Migration started!\nOperation ID: %s\nInstance ID: %s\nLogical Instance ID: %s\n",
			res.OperationID, res.InstanceID, res.LogicalInstanceID)
		if *waitFlag {
			waitForOperationOrExit(cli, res.OperationID)
		}

	case "resize":
		resizeCmd := flag.NewFlagSet("instances resize", flag.ExitOnError)
		tier := resizeCmd.String("tier", "", "Target tier name (required)")
		waitFlag := resizeCmd.Bool("wait", false, "Wait until the operation reaches a terminal state")
		_ = resizeCmd.Parse(os.Args[3:])
		if resizeCmd.NArg() < 1 || *tier == "" {
			fmt.Println("Usage: admiralctl instances resize --tier <tier_name> <instance_id>")
			os.Exit(1)
		}
		instID := resizeCmd.Arg(0)
		if !confirmDestructive("resize", fmt.Sprintf("instance %q to tier %q", instID, *tier)) {
			fmt.Println("Cancelled.")
			return
		}
		opID, err := cli.TriggerActionWithTier(instID, "resize", *tier)
		if err != nil {
			var rejected *client.ProvisionRejectedError
			if errors.As(err, &rejected) {
				printPolicyRejected(rejected)
				os.Exit(1)
			}
			fmt.Printf("Resize failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Resize queued successfully!\nOperation ID: %s\n", opID)
		if *waitFlag {
			waitForOperationOrExit(cli, opID)
		}

	default:
		fmt.Printf("Unknown action %q for instances.\n", action)
		os.Exit(1)
	}
}

// --- Operations Command ---

func handleOperations(cli *client.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: admiralctl operations <list|show|retry> [flags/args]")
		os.Exit(1)
	}

	action := os.Args[2]
	switch action {
	case "help", "-h", "--help":
		fmt.Println("Usage: admiralctl operations <list|show|retry> [flags/args]")
		return
	case "list":
		listCmd := flag.NewFlagSet("operations list", flag.ExitOnError)
		outputFlag := listCmd.String("output", "table", "Output format: table or json")
		_ = listCmd.Parse(os.Args[3:])

		ops, err := cli.GetOperations()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if *outputFlag == "json" {
			output.PrintJSON(ops)
			return
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

	case "show":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl operations show <operation_id>")
			os.Exit(1)
		}
		opID := os.Args[3]

		op, err := cli.GetOperation(opID)
		if err != nil {
			fmt.Printf("Error retrieving operation: %v\n", err)
			os.Exit(1)
		}

		output.PrintJSON(op)

	case "retry":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl operations retry <operation_id>")
			os.Exit(1)
		}
		res, err := cli.RetryOperation(os.Args[3])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Operation %s retried.\n", res["operation_id"])

	default:
		fmt.Printf("Unknown action %q for operations.\n", action)
		os.Exit(1)
	}
}

func handleOperation(cli *client.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: admiralctl operation status <operation_id>")
		os.Exit(1)
	}
	if os.Args[2] != "status" {
		fmt.Printf("Unknown action %q for operation.\n", os.Args[2])
		os.Exit(1)
	}
	if len(os.Args) < 4 {
		fmt.Println("Usage: admiralctl operation status <operation_id>")
		os.Exit(1)
	}
	op, err := cli.GetOperation(os.Args[3])
	if err != nil {
		fmt.Printf("Error retrieving operation: %v\n", err)
		os.Exit(1)
	}
	output.PrintJSON(op)
}

func waitForOperationOrExit(cli *client.Client, operationID string) {
	op, err := cli.WaitForOperation(operationID, 2*time.Second)
	if err != nil {
		fmt.Printf("Error waiting for operation: %v\n", err)
		os.Exit(1)
	}
	status := fmt.Sprintf("%v", op["status"])
	fmt.Printf("Operation %s finished with status: %s\n", operationID, status)
	if status != "succeeded" {
		output.PrintJSON(op)
		os.Exit(1)
	}
}

// --- Backups Command ---

func handleBackups(cli *client.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: admiralctl backups <list|show|restore|storage|delete|prune> [flags/args]")
		os.Exit(1)
	}

	action := os.Args[2]
	switch action {
	case "help", "-h", "--help":
		fmt.Println("Usage: admiralctl backups <list|show|restore|storage|delete|prune> [flags/args]")
		return
	case "list":
		listCmd := flag.NewFlagSet("backups list", flag.ExitOnError)
		outputFlag := listCmd.String("output", "table", "Output format: table or json")
		_ = listCmd.Parse(os.Args[3:])

		backups, err := cli.GetBackups()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if *outputFlag == "json" {
			output.PrintJSON(backups)
			return
		}
		headers := []string{"BACKUP ID", "INSTANCE ID", "TYPE", "STORAGE", "STATUS", "CREATED AT"}
		var rows [][]string
		for _, b := range backups {
			rows = append(rows, []string{
				fmt.Sprintf("%v", b["id"]),
				fmt.Sprintf("%v", b["instance_id"]),
				fmt.Sprintf("%v", b["backup_type"]),
				fmt.Sprintf("%v", b["storage_backend"]),
				fmt.Sprintf("%v", b["status"]),
				fmt.Sprintf("%v", b["created_at"]),
			})
		}
		output.PrintTable(headers, rows)

	case "show":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl backups show <backup_id>")
			os.Exit(1)
		}
		backupID := os.Args[3]
		backup, err := cli.GetBackup(backupID)
		if err != nil {
			fmt.Printf("Error retrieving backup: %v\n", err)
			os.Exit(1)
		}
		output.PrintJSON(backup)

	case "restore":
		restoreCmd := flag.NewFlagSet("backups restore", flag.ExitOnError)
		backupID := restoreCmd.String("backup-id", "", "Backup ID to restore (required)")
		instanceID := restoreCmd.String("instance-id", "", "Target instance ID (required)")
		service := restoreCmd.String("service", "", "Service name matching the backup source (required)")
		targetNode := restoreCmd.String("target-node", "", "Optional target node ID")
		sourceType := restoreCmd.String("source-type", "", "Optional source type override")
		sourceURI := restoreCmd.String("source-uri", "", "Optional source URI override")
		verifyChecksum := restoreCmd.Bool("verify-checksum", true, "Verify checksum during restore")
		_ = restoreCmd.Parse(os.Args[3:])

		if *backupID == "" || *instanceID == "" || *service == "" {
			fmt.Println("Error: --backup-id, --instance-id, and --service are required.")
			os.Exit(1)
		}

		req := admiral.RestoreBackupRequest{
			BackupID:       *backupID,
			TargetAppID:    *instanceID,
			Service:        *service,
			TargetNodeID:   *targetNode,
			RestoreMode:    "replace",
			VerifyChecksum: *verifyChecksum,
		}
		if *sourceType != "" || *sourceURI != "" {
			req.Source.Type = *sourceType
			req.Source.URI = *sourceURI
		}

		res, err := cli.RestoreBackup(req)
		if err != nil {
			fmt.Printf("Restore failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Restore queued successfully!\nOperation ID: %s\n", res.OperationID)

	case "storage":
		storageCmd := flag.NewFlagSet("backups storage", flag.ExitOnError)
		outputFlag := storageCmd.String("output", "table", "Output format: table or json")
		_ = storageCmd.Parse(os.Args[3:])

		if storageCmd.NArg() > 0 {
			sub := storageCmd.Arg(0)
			switch sub {
			case "get":
				cfg, err := cli.GetBackupStorageConfig()
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				if *outputFlag == "json" {
					output.PrintJSON(cfg)
					return
				}
				fmt.Printf("Backend:  %s\n", cfg.Backend)
				fmt.Printf("Enabled:  %v\n", cfg.Enabled)
				fmt.Printf("Endpoint: %s\n", cfg.Endpoint)
				fmt.Printf("Region:   %s\n", cfg.Region)
				fmt.Printf("Bucket:   %s\n", cfg.Bucket)
				fmt.Printf("Prefix:   %s\n", cfg.Prefix)

			case "set":
				setCmd := flag.NewFlagSet("backups storage set", flag.ExitOnError)
				backend := setCmd.String("backend", "s3", "Storage backend: s3 or local")
				endpoint := setCmd.String("endpoint", "", "S3-compatible endpoint URL")
				region := setCmd.String("region", "us-east-1", "S3 region")
				bucket := setCmd.String("bucket", "", "S3 bucket name")
				prefix := setCmd.String("prefix", "", "S3 key prefix")
				accessKeyEnv := setCmd.String("access-key-env", "ADMIRAL_AWS_ACCESS_KEY_ID", "Env var name for access key")
				secretKeyEnv := setCmd.String("secret-key-env", "ADMIRAL_AWS_SECRET_ACCESS_KEY", "Env var name for secret key")
				_ = setCmd.Parse(os.Args[4:])

				cfg := admiral.BackupStorageConfig{
					Backend:      *backend,
					Enabled:      true,
					Endpoint:     *endpoint,
					Region:       *region,
					Bucket:       *bucket,
					Prefix:       *prefix,
					AccessKeyEnv: *accessKeyEnv,
					SecretKeyEnv: *secretKeyEnv,
				}
				if err := cli.SetBackupStorageConfig(cfg); err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("Backup storage configuration updated.")

			case "test":
				if err := cli.TestBackupStorageConfig(); err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("Backup storage test passed.")

			default:
				fmt.Printf("Unknown storage subcommand %q. Use: get, set, test\n", sub)
				os.Exit(1)
			}
			return
		}
		fmt.Println("Usage: admiralctl backups storage <get|set|test> [flags]")
		fmt.Println("\nSubcommands:")
		fmt.Println("  get          Show current backup storage configuration")
		fmt.Println("  set          Update backup storage configuration")
		fmt.Println("  test         Test storage connectivity")

	case "delete":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl backups delete <backup_id>")
			os.Exit(1)
		}
		backupID := os.Args[3]
		fmt.Printf("Are you sure you want to delete backup %s? (y/N): ", backupID)
		var confirm string
		_, _ = fmt.Scanf("%s", &confirm)
		if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
			fmt.Println("Cancelled.")
			return
		}
		if err := cli.DeleteBackup(backupID); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Backup %s deleted.\n", backupID)

	case "prune":
		fmt.Print("Are you sure you want to prune old succeeded backups? (y/N): ")
		var confirm string
		_, _ = fmt.Scanf("%s", &confirm)
		if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
			fmt.Println("Cancelled.")
			return
		}
		if err := cli.PruneBackups(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Backups pruned successfully.")

	default:
		fmt.Printf("Unknown action %q for backups.\n", action)
		os.Exit(1)
	}
}

// --- Routes Command ---

func handleRoutes(cli *client.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: admiralctl routes <list|show|sync|enable|disable> [flags/args]")
		os.Exit(1)
	}

	action := os.Args[2]
	switch action {
	case "help", "-h", "--help":
		fmt.Println("Usage: admiralctl routes <list|show|sync|enable|disable> [flags/args]")
		return
	case "list":
		listCmd := flag.NewFlagSet("routes list", flag.ExitOnError)
		outputFlag := listCmd.String("output", "table", "Output format: table or json")
		_ = listCmd.Parse(os.Args[3:])

		routes, err := cli.GetRoutes()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if *outputFlag == "json" {
			output.PrintJSON(routes)
			return
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
		output.PrintTable(headers, rows)

	case "show":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl routes show <hostname>")
			os.Exit(1)
		}
		route, err := cli.GetRoute(os.Args[3])
		if err != nil {
			fmt.Printf("Error retrieving route: %v\n", err)
			os.Exit(1)
		}
		output.PrintJSON(route)

	case "sync":
		if err := cli.SyncRoutes(); err != nil {
			fmt.Printf("Sync failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Routes synchronized successfully.")

	case "enable", "disable":
		if len(os.Args) < 4 {
			fmt.Printf("Usage: admiralctl routes %s <hostname>\n", action)
			os.Exit(1)
		}
		hostname := os.Args[3]
		var err error
		if action == "enable" {
			err = cli.EnableRoute(hostname)
		} else {
			if !confirmDestructive("disable", fmt.Sprintf("route %q", hostname)) {
				fmt.Println("Cancelled.")
				return
			}
			err = cli.DisableRoute(hostname)
		}
		if err != nil {
			fmt.Printf("Route %s failed: %v\n", action, err)
			os.Exit(1)
		}
		fmt.Printf("Route %s successfully.\n", action)

	default:
		fmt.Printf("Unknown action %q for routes.\n", action)
		os.Exit(1)
	}
}

// --- User Command ---

func handleUser(cli *client.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: admiralctl user <create|list|set-password> [flags]")
		os.Exit(1)
	}

	action := os.Args[2]
	switch action {
	case "help", "-h", "--help":
		fmt.Println("Usage: admiralctl user <create|list|set-password> [flags]")
		return
	case "list":
		listCmd := flag.NewFlagSet("user list", flag.ExitOnError)
		outputFlag := listCmd.String("output", "table", "Output format: table or json")
		_ = listCmd.Parse(os.Args[3:])

		users, err := cli.ListUsers()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if *outputFlag == "json" {
			output.PrintJSON(users)
			return
		}

		headers := []string{"USERNAME", "ROLE", "CREATED AT"}
		var rows [][]string
		for _, u := range users {
			rows = append(rows, []string{
				fmt.Sprintf("%v", u["username"]),
				fmt.Sprintf("%v", u["role"]),
				fmt.Sprintf("%v", u["created_at"]),
			})
		}
		output.PrintTable(headers, rows)

	case "create":
		createCmd := flag.NewFlagSet("user create", flag.ExitOnError)
		userType := createCmd.String("type", "admin", "Role: superadmin, admin, platform, support, audit")
		_ = createCmd.Parse(os.Args[3:])
		if createCmd.NArg() < 1 {
			fmt.Println("Usage: admiralctl user create [--type <role>] <username>")
			os.Exit(1)
		}
		username := createCmd.Arg(0)

		fmt.Printf("Enter password for user %q: ", username)
		password, err := readPassword()
		if err != nil {
			fmt.Printf("\nFailed to read password: %v\n", err)
			os.Exit(1)
		}
		fmt.Println()

		res, err := cli.CreateUser(username, password, *userType)
		if err != nil {
			fmt.Printf("Error creating user: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("User %q created successfully with role %q\n", res["username"], res["role"])

	case "set-password":
		if len(os.Args) < 4 {
			fmt.Println("Usage: admiralctl user set-password <username>")
			os.Exit(1)
		}
		username := os.Args[3]

		fmt.Printf("Enter new password for user %q: ", username)
		password, err := readPassword()
		if err != nil {
			fmt.Printf("\nFailed to read password: %v\n", err)
			os.Exit(1)
		}
		fmt.Println()

		if err := cli.SetPassword(username, password); err != nil {
			fmt.Printf("Error setting password: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Password for user %q updated successfully.\n", username)

	default:
		fmt.Printf("Unknown action %q for user.\n", action)
		os.Exit(1)
	}
}

func readPassword() (string, error) {
	if isTerminal(os.Stdin.Fd()) {
		return readPasswordFromTerminal(int(os.Stdin.Fd()))
	}
	return readPasswordFromReader(os.Stdin)
}

func readPasswordFromReader(r io.Reader) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func isTerminal(fd uintptr) bool {
	_, err := getTermios(fd)
	return err == nil
}

func readPasswordFromTerminal(fd int) (string, error) {
	state, err := getTermios(uintptr(fd))
	if err != nil {
		return "", err
	}
	original := *state
	modified := original
	modified.Lflag &^= syscall.ECHO
	if err := setTermios(uintptr(fd), &modified); err != nil {
		return "", err
	}
	defer func() {
		_ = setTermios(uintptr(fd), &original)
	}()
	return readPasswordFromReader(os.Stdin)
}

func getTermios(fd uintptr) (*syscall.Termios, error) {
	termios := &syscall.Termios{}
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(termios)), 0, 0, 0) // #nosec G103 -- terminal ioctls require unsafe syscall access
	if errno != 0 {
		return nil, errno
	}
	return termios, nil
}

func setTermios(fd uintptr, state *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(state)), 0, 0, 0) // #nosec G103 -- terminal ioctls require unsafe syscall access
	if errno != 0 {
		return errno
	}
	return nil
}

func sanitizeInputFilePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty file path")
	}
	return filepath.Clean(path), nil
}

func readInputFile(path string) ([]byte, error) {
	return os.ReadFile(path) // #nosec G304 -- path comes from explicit user input for file-based commands
}

// --- Storage Command ---

func handleStorage(cli *client.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: admiralctl storage <instances|nodes> [flags]")
		os.Exit(1)
	}

	action := os.Args[2]
	switch action {
	case "help", "-h", "--help":
		fmt.Println("Usage: admiralctl storage <instances|nodes> [flags]")
		return
	case "instances":
		listCmd := flag.NewFlagSet("storage instances", flag.ExitOnError)
		outputFlag := listCmd.String("output", "table", "Output format: table or json")
		_ = listCmd.Parse(os.Args[3:])

		apps, err := cli.GetCustomerApps()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if *outputFlag == "json" {
			output.PrintJSON(apps)
			return
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

	case "nodes":
		listCmd := flag.NewFlagSet("storage nodes", flag.ExitOnError)
		outputFlag := listCmd.String("output", "table", "Output format: table or json")
		_ = listCmd.Parse(os.Args[3:])

		nodes, err := cli.GetNodes()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if *outputFlag == "json" {
			output.PrintJSON(nodes)
			return
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

	default:
		fmt.Printf("Unknown action %q for storage.\n", action)
		os.Exit(1)
	}
}
