package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	removeVolumes bool
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop KubeOrch services",
	Long:  `Stop running KubeOrch services`,
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

	projectConfig, err := getCurrentProjectConfig()
	if err != nil {
		return fmt.Errorf("no project initialized in current directory. Run 'orchcli init' first")
	}

	uiLocal := projectConfig.UIPath != "" && dirExists(projectConfig.UIPath)
	coreLocal := projectConfig.CorePath != "" && dirExists(projectConfig.CorePath)

	fmt.Println("🛑 stopping kubeorchestra services...")

	composeFile := getComposeFile(uiLocal, coreLocal)
	composeFile = filepath.Join(projectConfig.Path, composeFile)

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
	// #nosec G204 -- dockerCompose command is from a fixed set, cmdArgs are controlled
	composeCmd := exec.Command(dockerCompose[0], dockerCompose[1:]...)
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr
	composeCmd.Dir = projectConfig.Path

	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	fmt.Println("✅ services stopped")
	return nil
}
