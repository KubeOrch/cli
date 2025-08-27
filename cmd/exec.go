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
	
Services: postgres, core, ui

Examples:
  # Open PostgreSQL shell
  orchcli exec postgres psql -U kubeorchestra -d kubeorchestra
  
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
		"postgres": "kubeorchestra-postgres",
		"core":     "kubeorchestra-core",
		"ui":       "kubeorchestra-ui",
	}

	containerName, valid := validServices[service]
	if !valid {
		return fmt.Errorf("invalid service: %s. Valid services: postgres, core, ui", service)
	}

	// Check if container is running
	// #nosec G204 -- containerName is validated from a fixed allowlist above
	checkCmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
	output, err := checkCmd.Output()
	if err != nil || strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("service %s is not running. Start it with: orchcli start", service)
	}

	// Build docker exec command
	dockerArgs := []string{"exec", "-it", containerName}

	if len(args) > 1 {
		// Specific command provided
		dockerArgs = append(dockerArgs, args[1:]...)
	} else {
		// Default shells for each service
		switch service {
		case "postgres":
			dockerArgs = append(dockerArgs, "psql", "-U", "kubeorchestra", "-d", "kubeorchestra")
		case "core":
			dockerArgs = append(dockerArgs, "sh")
		case "ui":
			dockerArgs = append(dockerArgs, "sh")
		}
	}

	// #nosec G204 -- dockerArgs are constructed from validated inputs
	execCmd := exec.Command("docker", dockerArgs...)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin

	return execCmd.Run()
}
