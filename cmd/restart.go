package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [service]",
	Short: "Restart KubeOrchestra services",
	Long:  `Restart KubeOrchestra services. Optionally specify a service name (ui, core, postgres)`,
	RunE:  runRestart,
}

func init() {
	rootCmd.AddCommand(restartCmd)
}

func runRestart(cmd *cobra.Command, args []string) error {
	if err := validateDockerCompose(); err != nil {
		return err
	}

	// detect what was initialized
	uiLocal := dirExists("./ui")
	coreLocal := dirExists("./core")

	fmt.Println("🔄 restarting kubeorchestra services...")
	
	// determine which compose file to use
	composeFile := getComposeFile(uiLocal, coreLocal)
	
	// check if compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("no services are running. start services first with: orchcli start")
	}
	
	// build docker-compose command
	cmdArgs := []string{"-f", composeFile, "restart"}
	
	// add service name if provided
	if len(args) > 0 {
		cmdArgs = append(cmdArgs, args[0])
		fmt.Printf("   restarting %s...\n", args[0])
	} else {
		fmt.Println("   restarting all services...")
	}
	
	// execute docker-compose
	composeCmd := exec.Command("docker-compose", cmdArgs...)
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr
	
	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to restart services: %w", err)
	}
	
	fmt.Println("✅ services restarted")
	return nil
}