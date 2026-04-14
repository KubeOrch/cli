package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <service> [command]",
	Short: "Execute a command in a running service container",
	Long: `Execute a command in a running service container.

Services: mongodb, core, ui

Examples:
  # Open MongoDB shell
  orchcli exec mongodb mongosh kubeorchestra

  # Run migrations in Core
  orchcli exec core go run . migrate

  # Open bash shell in UI container
  orchcli exec ui bash

  # Check Node version in UI container
  orchcli exec ui node --version`,
	Args: cobra.MinimumNArgs(1),
	RunE: runExec,
}

func init() {
	rootCmd.AddCommand(execCmd)
}

func runExec(cmd *cobra.Command, args []string) error {
	if err := validateDockerCompose(); err != nil {
		return err
	}

	service := args[0]
	validServices := map[string]string{
		"mongodb": "kubeorchestra-mongodb",
		"core":    "kubeorchestra-core",
		"ui":      "kubeorchestra-ui",
	}

	containerName, valid := validServices[service]
	if !valid {
		return fmt.Errorf("invalid service: %s. Valid services: mongodb, core, ui", service)
	}

	// Find the actual running container — name may have a -dev or -hybrid suffix
	// depending on which orchcli init mode was used.
	// #nosec G204 -- containerName is validated from a fixed allowlist above
	checkCmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
	output, err := checkCmd.Output()
	// Take the first line only — docker ps may return multiple matches.
	trimmed := strings.TrimSpace(string(output))
	if idx := strings.IndexByte(trimmed, '\n'); idx != -1 {
		trimmed = trimmed[:idx]
	}
	actualName := trimmed
	if err != nil || actualName == "" {
		return fmt.Errorf("service %s is not running. Start it with: orchcli start", service)
	}

	// Build docker exec command using the actual running container name
	dockerArgs := []string{"exec", "-it", actualName}

	if len(args) > 1 {
		// Specific command provided
		dockerArgs = append(dockerArgs, args[1:]...)
	} else {
		// Default shells for each service
		switch service {
		case "mongodb":
			dockerArgs = append(dockerArgs, "mongosh", "kubeorchestra")
		case "core":
			dockerArgs = append(dockerArgs, "sh")
		case "ui":
			dockerArgs = append(dockerArgs, "sh")
		}
	}

	// #nosec G204 -- dockerArgs are constructed from validated inputs
	execCommand := exec.Command("docker", dockerArgs...)
	execCommand.Stdout = os.Stdout
	execCommand.Stderr = os.Stderr
	execCommand.Stdin = os.Stdin

	return execCommand.Run()
}
