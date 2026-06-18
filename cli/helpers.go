// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/admiral-project/admiral/admirald/pkg/admiral"
	"github.com/spf13/cobra"
)

// confirmDestructive asks the user for confirmation before a destructive action.
// If the command has a --force flag set, it returns true without prompting.
func confirmDestructive(cmd *cobra.Command, action, target string) bool {
	if force, _ := cmd.Flags().GetBool("force"); force {
		return true
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Are you sure you want to %s %s? (y/N): ", action, target)
	var confirm string
	_, _ = fmt.Fscanln(cmd.InOrStdin(), &confirm)
	return strings.ToLower(strings.TrimSpace(confirm)) == "y"
}

// requireToken ensures a token is present in the loaded configuration.
func requireToken() {
	if cfg == nil || strings.TrimSpace(cfg.Token) == "" {
		cobra.CheckErr(fmt.Errorf("no authentication token configured. Run 'admiralctl init --token <token>' or set ADMIRAL_ADMIN_TOKEN"))
	}
}

// resolveToken returns the token to use, preferring an explicit flag, then the
// configured token, then prompting the user interactively.
func resolveToken(cmd *cobra.Command, tokenFlag, cfgToken string) string {
	if tokenFlag != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: --token exposes the secret in the process list. Prefer ADMIRAL_ADMIN_TOKEN env var.")
		return tokenFlag
	}
	if cfgToken != "" {
		return cfgToken
	}
	fmt.Fprint(cmd.OutOrStdout(), "Enter admin token: ")
	t, err := readPassword()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "\nFailed to read token: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(cmd.OutOrStdout())
	return strings.TrimSpace(t)
}

// printPolicyRejected renders a rejected provisioning response in human-readable form.
func printPolicyRejected(resp admiral.ProvisioningRejectedResponse) {
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

// readPassword reads a password from stdin without echoing it if the input is a terminal.
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

// sanitizeInputFilePath returns a clean absolute path for a user-provided file.
func sanitizeInputFilePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty file path")
	}
	return filepath.Clean(path), nil
}

// readInputFile reads the contents of a file that the user explicitly requested.
func readInputFile(path string) ([]byte, error) {
	return os.ReadFile(path) // #nosec G304 -- path comes from explicit user input for file-based commands
}
