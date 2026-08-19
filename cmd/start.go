package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	detach bool
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start KubeOrch services",
	Long: `Start KubeOrch services based on your initialization:
- If no repos cloned: runs from Docker images
- If UI cloned: runs UI locally with hot reload, Core from image
- If Core cloned: runs Core locally with hot reload, UI from image
- If both cloned: runs both locally with hot reload`,
	RunE: runStart,
}

func init() {
	startCmd.Flags().BoolVarP(&detach, "detach", "d", false, "run services in background")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	projectConfig, err := getCurrentProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to resolve KubeOrch project: %w", err)
	}

	if err := validateDockerCompose(); err != nil {
		return err
	}

	uiLocal := projectConfig.UIPath != ""
	coreLocal := projectConfig.CorePath != ""

	fmt.Println("🚀 starting kubeorchestra services...")

	var composeFile string

	switch {
	case !uiLocal && !coreLocal:
		fmt.Println("   mode: production (using docker images)")
		composeFile = filepath.Join(projectConfig.Path, "docker", "docker-compose.prod.yml")
	case uiLocal && coreLocal:
		fmt.Println("   mode: development (both local)")
		composeFile = filepath.Join(projectConfig.Path, "docker", "docker-compose.dev.yml")
	case uiLocal:
		fmt.Println("   mode: ui development (ui local, core from image)")
		composeFile = filepath.Join(projectConfig.Path, "docker", "docker-compose.hybrid-ui.yml")
	default:
		fmt.Println("   mode: core development (core local, ui from image)")
		composeFile = filepath.Join(projectConfig.Path, "docker", "docker-compose.hybrid-core.yml")
	}

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("compose file %s not found. please ensure docker-compose files exist in docker/ directory", composeFile)
	}

	cmdArgs := []string{"-f", composeFile, "up"}

	if detach {
		cmdArgs = append(cmdArgs, "-d")
	}

	dockerCompose := getDockerComposeCommand()
	allArgs := make([]string, 0, len(dockerCompose)+len(cmdArgs))
	allArgs = append(allArgs, dockerCompose...)
	allArgs = append(allArgs, cmdArgs...)
	composeCmd := exec.Command(allArgs[0], allArgs[1:]...)
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr
	composeCmd.Stdin = os.Stdin
	composeCmd.Dir = projectConfig.Path

	fmt.Printf("   running: %s %s\n", strings.Join(dockerCompose, " "), joinArgs(cmdArgs))

	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	if detach {
		fmt.Println("✅ docker services started in background")
		fmt.Println()

		fmt.Println("⏳ waiting for mongodb to be ready...")
		if err := waitForMongoDB(); err != nil {
			fmt.Printf("⚠️  warning: %v\n", err)
			fmt.Println("   services may take a moment to be fully ready")
		} else {
			fmt.Println("✅ mongodb is ready")
		}

		fmt.Println()

		// provide instructions based on what was initialized
		switch {
		case uiLocal && coreLocal:
			fmt.Println("📝 next steps for development:")
			fmt.Printf("   1. start core: cd %s && go run .\n", projectConfig.CorePath)
			fmt.Printf("   2. start ui: cd %s && npm run dev\n", projectConfig.UIPath)
			fmt.Println()
			fmt.Println("   core API will run on http://localhost:3000/v1/api")
			fmt.Println("   ui will run on http://localhost:3001")
			fmt.Println("   mongodb is at localhost:27017")
		case uiLocal:
			fmt.Println("📝 next steps for ui development:")
			fmt.Printf("   start ui: cd %s && npm run dev\n", projectConfig.UIPath)
			fmt.Println()
			fmt.Println("   ui will run on http://localhost:3001")
			fmt.Println("   core api is at http://localhost:3000/v1/api (docker)")
			fmt.Println("   mongodb is at localhost:27017 (docker)")
		case coreLocal:
			fmt.Println("📝 next steps for core development:")
			fmt.Printf("   start core: cd %s && go run .\n", projectConfig.CorePath)
			fmt.Println()
			fmt.Println("   core api: http://localhost:3000/v1/api (host)")
			fmt.Println("   ui: http://localhost:3001 (docker)")
			fmt.Println("   mongodb: localhost:27017 (docker)")
		default:
			fmt.Println("📊 all services running in docker:")
			fmt.Println("   ui: http://localhost:3001")
			fmt.Println("   api: http://localhost:3000/v1/api")
			fmt.Println("   mongodb: localhost:27017")
		}

		fmt.Println()
		fmt.Println("🛑 stop docker services: orchcli stop")
		fmt.Println("📝 view logs: orchcli logs")
		fmt.Println("📊 check status: orchcli status")
	}

	return nil
}

func waitForMongoDB() error {
	maxRetries := 30
	containerNames := []string{
		"kubeorchestra-mongodb",
		"kubeorchestra-mongodb-dev",
		"kubeorchestra-mongodb-hybrid",
	}

	for i := 0; i < maxRetries; i++ {
		for _, name := range containerNames {
			// #nosec G204 -- name is from a hardcoded list of known container names
			cmd := exec.Command("docker", "exec", name, "mongosh", "--eval", "db.adminCommand('ping')")
			if err := cmd.Run(); err == nil {
				return nil
			}
		}

		time.Sleep(time.Second)
	}

	return fmt.Errorf("mongodb did not become ready in 30 seconds")
}
