package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	removeVolumes bool
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop KubeOrchestra services",
	Long:  `Stop running KubeOrchestra services`,
	RunE:  runStop,
}

func init() {
	stopCmd.Flags().BoolVarP(&removeVolumes, "volumes", "v", false, "remove volumes when stopping")
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	if err := validateDockerCompose(); err != nil {
		return err
	}

	uiLocal := dirExists("./ui")
	coreLocal := dirExists("./core")

	fmt.Println("🛑 stopping kubeorchestra services...")

	composeFile := getComposeFile(uiLocal, coreLocal)

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		fmt.Println("⚠️  no services are running")
		return nil
	}

	cmdArgs := []string{"-f", composeFile, "down"}

	if removeVolumes {
		cmdArgs = append(cmdArgs, "-v")
		fmt.Println("   removing volumes...")
	}

	dockerCompose := getDockerComposeCommand()
	dockerCompose = append(dockerCompose, cmdArgs...)
	composeCmd := exec.Command(dockerCompose[0], dockerCompose[1:]...) // #nosec G204 -- dockerCompose command is from a fixed set, cmdArgs are controlled
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr

	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	fmt.Println("✅ services stopped")
	return nil
}
