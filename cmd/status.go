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
	// #nosec G204 -- the executable is selected from hardcoded Docker Compose command names.
	psCmd := exec.CommandContext(commandContext, psArgs[0], psArgs[1:]...)
	psCmd.Dir = projectConfig.Path
	psOutput, err := psCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check service status: %w", err)
	}

	fmt.Println("📊 service status:")
	fmt.Println(string(psOutput))

	fmt.Println("💾 database status:")
	dbArgs := append([]string{}, dockerCompose...)
	dbArgs = append(dbArgs,
		"-f", composeFile,
		"exec", "-T", "mongodb",
		"mongosh", "--eval", "db.adminCommand('ping')",
	)
	// #nosec G204 -- the executable is selected from hardcoded Docker Compose command names.
	dbCheckCmd := exec.CommandContext(commandContext, dbArgs[0], dbArgs[1:]...)
	dbCheckCmd.Dir = projectConfig.Path
	dbOutput, dbErr := dbCheckCmd.Output()

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
	corePort := configuredEnvironmentValue("KUBEORCH_CORE_PORT", "3000")
	uiPort := configuredEnvironmentValue("KUBEORCH_UI_PORT", "3001")
	mongoPort := configuredEnvironmentValue("KUBEORCH_MONGO_PORT", "27017")
	printApplicationStatus(commandContext, corePort, uiPort)

	fmt.Println()
	fmt.Println("🌐 service endpoints:")
	fmt.Printf("   ui:      http://localhost:%s\n", uiPort)
	fmt.Printf("   api:     http://localhost:%s/v1/api\n", corePort)
	fmt.Printf("   mongodb: localhost:%s\n", mongoPort)

	fmt.Println()
	fmt.Println("💡 tips:")
	fmt.Println("   view logs:    orchcli logs")
	fmt.Println("   stop services: orchcli stop")
	fmt.Println("   restart:      orchcli restart")

	return nil
}

func printApplicationStatus(ctx context.Context, corePort, uiPort string) {
	checks := []struct {
		name string
		url  string
	}{
		{name: "core", url: "http://localhost:" + corePort + "/v1"},
		{name: "ui", url: "http://localhost:" + uiPort + "/"},
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
