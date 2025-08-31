package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check status of KubeOrchestra services",
	Long:  `Check the status and health of running KubeOrchestra services`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	if err := validateDockerCompose(); err != nil {
		return err
	}

	projectConfig, err := getCurrentProjectConfig()
	if err != nil {
		return fmt.Errorf("no project initialized in current directory. Run 'orchcli init' first")
	}

	uiLocal := projectConfig.UIPath != "" && dirExists(projectConfig.UIPath)
	coreLocal := projectConfig.CorePath != "" && dirExists(projectConfig.CorePath)

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
	psCmd := exec.Command(psArgs[0], psArgs[1:]...)
	psCmd.Dir = projectConfig.Path
	psOutput, err := psCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check service status: %w", err)
	}

	fmt.Println("📊 service status:")
	fmt.Println(string(psOutput))

	fmt.Println("💾 database status:")
	dbCheckCmd := exec.Command("docker", "exec", "kubeorchestra-mongodb", "mongosh", "--eval", "db.adminCommand('ping')")
	dbOutput, dbErr := dbCheckCmd.Output()
	if dbErr != nil {
		for _, name := range []string{"kubeorchestra-mongodb-dev", "kubeorchestra-mongodb-hybrid"} {
			altCmd := exec.Command("docker", "exec", name, "mongosh", "--eval", "db.adminCommand('ping')")
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
	fmt.Println("🌐 service endpoints:")
	fmt.Println("   ui:      http://localhost:3001")
	fmt.Println("   api:     http://localhost:3000")
	fmt.Println("   mongodb: localhost:27017")

	fmt.Println()
	fmt.Println("💡 tips:")
	fmt.Println("   view logs:    orchcli logs")
	fmt.Println("   stop services: orchcli stop")
	fmt.Println("   restart:      orchcli restart")

	return nil
}
