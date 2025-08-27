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

	uiLocal := dirExists("./ui")
	coreLocal := dirExists("./core")

	fmt.Println("🔄 restarting kubeorchestra services...")

	composeFile := getComposeFile(uiLocal, coreLocal)

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("no services are running. start services first with: orchcli start")
	}

	cmdArgs := []string{"-f", composeFile, "restart"}

	if len(args) > 0 {
		cmdArgs = append(cmdArgs, args[0])
		fmt.Printf("   restarting %s...\n", args[0])
	} else {
		fmt.Println("   restarting all services...")
	}

	dockerCompose := getDockerComposeCommand()
	dockerCompose = append(dockerCompose, cmdArgs...)
	// #nosec G204 -- dockerCompose command is from a fixed set, cmdArgs are controlled
	composeCmd := exec.Command(dockerCompose[0], dockerCompose[1:]...)
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr

	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to restart services: %w", err)
	}

	fmt.Println("✅ services restarted")
	return nil
}
