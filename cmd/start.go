package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	if err := validateDockerCompose(); err != nil {
		return err
	}

	projectConfig, err := getCurrentProjectConfig()
	if err != nil {
		return fmt.Errorf("no project initialized in current directory. Run 'orchcli init' first")
	}

	uiLocal := projectConfig.UIPath != "" && dirExists(projectConfig.UIPath)
	coreLocal := projectConfig.CorePath != "" && dirExists(projectConfig.CorePath)

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

		fmt.Println("⏳ waiting for postgres to be ready...")
		if err := waitForPostgres(); err != nil {
			fmt.Printf("⚠️  warning: %v\n", err)
			fmt.Println("   services may take a moment to be fully ready")
		} else {
			fmt.Println("✅ postgres is ready")
		}

		fmt.Println()

		// provide instructions based on what was initialized
		switch {
		case uiLocal && coreLocal:
			fmt.Println("📝 next steps for development:")
			fmt.Printf("   1. start core: cd %s && air\n", projectConfig.CorePath)
			fmt.Printf("   2. start ui: cd %s && npm run dev\n", projectConfig.UIPath)
			fmt.Println()
			fmt.Println("   core will run on http://localhost:3000")
			fmt.Println("   ui will run on http://localhost:3001")
			fmt.Println("   postgresql is at localhost:5432")
		case uiLocal:
			fmt.Println("📝 next steps for ui development:")
			fmt.Printf("   start ui: cd %s && npm run dev\n", projectConfig.UIPath)
			fmt.Println()
			fmt.Println("   ui will run on http://localhost:3001")
			fmt.Println("   core api is at http://localhost:3000 (docker)")
			fmt.Println("   postgresql is at localhost:5432 (docker)")
		case coreLocal:
			fmt.Println("📝 backend development mode:")
			fmt.Println("   ✅ core is running in docker with your code mounted")
			fmt.Println("   ✅ hot reload enabled - just edit your files")
			fmt.Println()
			fmt.Println("   core api: http://localhost:3000 (docker with mounted code)")
			fmt.Println("   ui: http://localhost:3001 (docker)")
			fmt.Println("   postgresql: localhost:5432 (docker)")
			fmt.Println()
			fmt.Println("   note: no go installation required!")
		default:
			fmt.Println("📊 all services running in docker:")
			fmt.Println("   ui: http://localhost:3001")
			fmt.Println("   api: http://localhost:3000")
			fmt.Println("   postgresql: localhost:5432")
		}

		fmt.Println()
		fmt.Println("🛑 stop docker services: orchcli stop")
		fmt.Println("📝 view logs: orchcli logs")
		fmt.Println("📊 check status: orchcli status")
	}

	return nil
}

func waitForPostgres() error {
	maxRetries := 30
	containerNames := []string{
		"kubeorchestra-postgres",
		"kubeorchestra-postgres-dev",
		"kubeorchestra-postgres-hybrid",
	}

	for i := 0; i < maxRetries; i++ {
		for _, name := range containerNames {
			cmd := exec.Command("docker", "exec", name, "pg_isready", "-U", "kubeorchestra", "-d", "kubeorchestra")
			if err := cmd.Run(); err == nil {
				return nil
			}
		}

		_ = exec.Command("sleep", "1").Run()
	}

	return fmt.Errorf("postgres did not become ready in 30 seconds")
}
