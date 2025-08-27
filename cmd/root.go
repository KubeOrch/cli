package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "orchcli",
	Short: "OrchCLI - KubeOrchestra Developer CLI",
	Long: `OrchCLI is a developer tool for working with the KubeOrchestra platform.

It helps developers:
- Clone and setup UI/Core repositories for development
- Run local development environment with hot reload
- Handle fork-based contributions for external developers
- Quick production testing with latest images`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildDate),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate(`OrchCLI {{.Version}}
{{printf "License: Apache-2.0"}}
{{printf "Repository: https://github.com/kubeorchestra/cli"}}
`)

	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
