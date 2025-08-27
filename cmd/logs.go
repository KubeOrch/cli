package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	follow     bool
	tailLines  string
	timestamps bool
	service    string
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "View logs from KubeOrchestra services",
	Long:  `View logs from running KubeOrchestra services. Optionally specify a service name (ui, core, postgres)`,
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")
	logsCmd.Flags().StringVar(&tailLines, "tail", "100", "number of lines to show from the end of logs")
	logsCmd.Flags().BoolVarP(&timestamps, "timestamps", "t", false, "show timestamps")
	logsCmd.Flags().StringVar(&service, "service", "", "specific service to show logs for (ui, core, postgres)")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	if err := validateDockerCompose(); err != nil {
		return err
	}

	uiLocal := dirExists("./ui")
	coreLocal := dirExists("./core")

	composeFile := getComposeFile(uiLocal, coreLocal)

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("no services are running. start services first with: orchcli start")
	}

	cmdArgs := []string{"-f", composeFile, "logs"}

	if follow {
		cmdArgs = append(cmdArgs, "-f")
	}

	if tailLines != "" {
		cmdArgs = append(cmdArgs, "--tail", tailLines)
	}

	if timestamps {
		cmdArgs = append(cmdArgs, "-t")
	}

	if service != "" {
		cmdArgs = append(cmdArgs, service)
	} else if len(args) > 0 {
		cmdArgs = append(cmdArgs, args[0])
	}

	dockerCompose := getDockerComposeCommand()
	dockerCompose = append(dockerCompose, cmdArgs...)
	// #nosec G204 -- dockerCompose command is from a fixed set, cmdArgs are controlled
	composeCmd := exec.Command(dockerCompose[0], dockerCompose[1:]...)
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr
	composeCmd.Stdin = os.Stdin

	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}

	return nil
}
