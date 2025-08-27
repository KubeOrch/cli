package cmd

import (
	"bytes"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// ExecuteCommandC executes a command and returns the output
func ExecuteCommandC(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = root.Execute()

	w.Close()
	os.Stdout = oldStdout

	out, _ := io.ReadAll(r)
	output = buf.String() + string(out)

	return output, err
}

// GetRootCommand returns the root command for testing
func GetRootCommand() *cobra.Command {
	return rootCmd
}

// ResetCommands resets command flags for testing
func ResetCommands() {
	rootCmd.ResetFlags()
	initCmd.ResetFlags()
	startCmd.ResetFlags()
	stopCmd.ResetFlags()
	logsCmd.ResetFlags()
	statusCmd.ResetFlags()
	restartCmd.ResetFlags()
	debugCmd.ResetFlags()
	execCmd.ResetFlags()

	// Re-initialize flags
	initCmd.Flags().StringVar(&forkUI, "fork-ui", "", "Clone UI repository")
	initCmd.Flags().StringVar(&forkCore, "fork-core", "", "Clone Core repository")
	initCmd.Flags().BoolVar(&skipDeps, "skip-deps", false, "Skip dependency installation")
	initCmd.Flags().BoolVar(&autoInstall, "auto-install", true, "Automatically install missing dependencies")

	initCmd.Flags().Lookup("fork-ui").NoOptDefVal = "KubeOrchestra/ui"
	initCmd.Flags().Lookup("fork-core").NoOptDefVal = "KubeOrchestra/core"

	startCmd.Flags().BoolVarP(&detach, "detach", "d", false, "run services in background")
	stopCmd.Flags().BoolVarP(&removeVolumes, "volumes", "v", false, "remove volumes when stopping")

	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")
	logsCmd.Flags().StringVar(&tailLines, "tail", "100", "number of lines to show from the end of logs")
	logsCmd.Flags().BoolVarP(&timestamps, "timestamps", "t", false, "show timestamps")
	logsCmd.Flags().StringVar(&service, "service", "", "specific service to show logs for")
}
