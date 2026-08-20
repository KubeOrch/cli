package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check status of KubeOrch services",
	Long:  `Check the status and health of running KubeOrch services`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	projectConfig, err := getCurrentProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to resolve KubeOrch project: %w", err)
	}

	if validationErr := validateDockerCompose(); validationErr != nil {
		return validationErr
	}
	commandContext := cmd.Context()

	uiLocal := projectConfig.UIPath != ""
	coreLocal := projectConfig.CorePath != ""

	composeFile := getComposeFile(uiLocal, coreLocal)
	composeFile = filepath.Join(projectConfig.Path, composeFile)

	if _, statErr := os.Stat(composeFile); os.IsNotExist(statErr) {
		fmt.Println("⚠️  no services are running")
		return nil
	}

	fmt.Println("🔍 checking kubeorchestra services...")
	fmt.Println()

	dockerCompose := getDockerComposeCommand()
	const additionalArgs = 3 // -f, composeFile, ps
	psArgs := make([]string, 0, len(dockerCompose)+additionalArgs)
	psArgs = append(psArgs, dockerCompose...)
	psArgs = append(psArgs, "-f", composeFile, "ps")
	psCmd := exec.CommandContext(commandContext, psArgs[0], psArgs[1:]...)
	psCmd.Dir = projectConfig.Path
	psOutput, err := psCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check service status: %w", err)
	}

	fmt.Println("📊 service status:")
	fmt.Println(string(psOutput))

	fmt.Println("💾 database status:")
	dbCheckCmd := exec.CommandContext(
		commandContext,
		"docker",
		"exec",
		"kubeorchestra-mongodb",
		"mongosh",
		"--eval",
		"db.adminCommand('ping')",
	)
	dbOutput, dbErr := dbCheckCmd.Output()
	if dbErr != nil {
		for _, name := range []string{"kubeorchestra-mongodb-dev", "kubeorchestra-mongodb-hybrid"} {
			altCmd := exec.CommandContext(
				commandContext,
				"docker",
				"exec",
				name,
				"mongosh",
				"--eval",
				"db.adminCommand('ping')",
			)
			if output, err := altCmd.Output(); err == nil {
				dbOutput = output
				dbErr = nil
				break
			}
		}
	}

	if dbErr != nil {
		fmt.Println("   ❌ mongodb is not healthy or not running")
	} else {
		output := strings.TrimSpace(string(dbOutput))
		if strings.Contains(output, "{ ok: 1 }") {
			fmt.Println("   ✅ mongodb is healthy and accepting connections")
		} else {
			fmt.Println("   ⚠️  mongodb status:", output)
		}
	}

	fmt.Println()
	fmt.Println("🩺 application status:")
	printApplicationStatus(commandContext)

	fmt.Println()
	fmt.Println("🌐 service endpoints:")
	fmt.Println("   ui:      http://localhost:3001")
	fmt.Println("   api:     http://localhost:3000/v1/api")
	fmt.Println("   mongodb: localhost:27017")

	fmt.Println()
	fmt.Println("💡 tips:")
	fmt.Println("   view logs:    orchcli logs")
	fmt.Println("   stop services: orchcli stop")
	fmt.Println("   restart:      orchcli restart")

	return nil
}

func printApplicationStatus(ctx context.Context) {
	checks := []struct {
		name string
		url  string
	}{
		{name: "core", url: "http://localhost:3000/v1/"},
		{name: "ui", url: "http://localhost:3001/"},
	}
	type healthResult struct {
		err        error
		name       string
		statusCode int
	}
	results := make(chan healthResult, len(checks))
	client := &http.Client{Timeout: 2 * time.Second}
	for _, check := range checks {
		go func(name, url string) {
			request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if requestErr != nil {
				results <- healthResult{name: name, err: requestErr}
				return
			}
			response, requestErr := client.Do(request)
			if requestErr != nil {
				results <- healthResult{name: name, err: requestErr}
				return
			}
			defer response.Body.Close()
			results <- healthResult{name: name, statusCode: response.StatusCode}
		}(check.name, check.url)
	}
	for range checks {
		result := <-results
		switch {
		case result.err != nil:
			fmt.Printf("   ❌ %s is not reachable: %v\n", result.name, result.err)
		case result.statusCode >= http.StatusOK && result.statusCode < http.StatusBadRequest:
			fmt.Printf("   ✅ %s is healthy (HTTP %d)\n", result.name, result.statusCode)
		default:
			fmt.Printf("   ⚠️  %s returned HTTP %d\n", result.name, result.statusCode)
		}
	}
}
