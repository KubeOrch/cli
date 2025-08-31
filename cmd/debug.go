// Package cmd contains all CLI commands for OrchCLI
package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug connectivity between services",
	Long:  `Debug network connectivity and service communication`,
	RunE:  runDebug,
}

func init() {
	rootCmd.AddCommand(debugCmd)
}

func runDebug(cmd *cobra.Command, args []string) error {
	if err := validateDockerCompose(); err != nil {
		return err
	}

	fmt.Println("🔍 debugging service connectivity...")
	fmt.Println()

	fmt.Println("📡 docker networks:")
	networkCmd := exec.Command("docker", "network", "ls")
	networkOutput, _ := networkCmd.Output()
	lines := strings.Split(string(networkOutput), "\n")
	for _, line := range lines {
		if strings.Contains(line, "kubeorchestra") || strings.Contains(line, "NETWORK ID") {
			fmt.Println("   " + line)
		}
	}
	fmt.Println()

	fmt.Println("📦 running containers:")
	psCmd := exec.Command("docker", "ps", "--format", "table {{.Names}}\t{{.Status}}\t{{.Ports}}")
	psOutput, _ := psCmd.Output()
	psLines := strings.Split(string(psOutput), "\n")
	for _, line := range psLines {
		if strings.Contains(line, "kubeorchestra") || strings.Contains(line, "NAMES") {
			fmt.Println("   " + line)
		}
	}
	fmt.Println()

	fmt.Println("💾 testing database connectivity:")

	coreContainers := []string{
		"kubeorchestra-core",
		"kubeorchestra-core-dev",
		"kubeorchestra-core-hybrid",
	}

	coreFound := false
	for _, container := range coreContainers {
		checkCmd := exec.Command("docker", "ps", "-q", "-f", fmt.Sprintf("name=%s", container))
		if output, err := checkCmd.Output(); err == nil && len(output) > 0 {
			coreFound = true

			fmt.Printf("   testing from %s to mongodb...\n", container)

			pingCmd := exec.Command("docker", "exec", container, "sh", "-c", "nc -zv mongodb 27017 2>&1 || echo 'nc not available'")
			pingOutput, pingErr := pingCmd.Output()
			if pingErr != nil {
				fmt.Printf("   ⚠️  could not test connectivity: %v\n", pingErr)
			} else {
				output := strings.TrimSpace(string(pingOutput))
				if strings.Contains(output, "succeeded") || strings.Contains(output, "open") {
					fmt.Println("   ✅ network connectivity to mongodb:27017 is working")
				} else if strings.Contains(output, "not available") {
					telnetArgs := "timeout 2 telnet mongodb 27017 2>&1 || echo 'connection test failed'"
					telnetCmd := exec.Command("docker", "exec", container, "sh", "-c", telnetArgs)
					telnetOutput, _ := telnetCmd.Output()
					if strings.Contains(string(telnetOutput), "Connected") {
						fmt.Println("   ✅ network connectivity to mongodb:27017 is working")
					} else {
						fmt.Println("   ❌ cannot connect to mongodb:27017")
					}
				} else {
					fmt.Printf("   connectivity test result: %s\n", output)
				}
			}

			break
		}
	}

	if !coreFound {
		fmt.Println("   ⚠️  core container not found or not running")
	}

	fmt.Println()
	fmt.Println("🔌 testing direct mongodb access:")

	mongodbContainers := []string{
		"kubeorchestra-mongodb",
		"kubeorchestra-mongodb-dev",
		"kubeorchestra-mongodb-hybrid",
	}

	for _, container := range mongodbContainers {
		testCmd := exec.Command("docker", "exec", container, "mongosh", "--eval", "db.adminCommand('ping')")
		if _, err := testCmd.Output(); err == nil {
			fmt.Printf("   ✅ %s is ready\n", container)

			connTestCmd := exec.Command("docker", "exec", container, "mongosh", "kubeorchestra", "--eval", "db.getName()")
			if _, err := connTestCmd.Output(); err == nil {
				fmt.Println("   ✅ database connection test successful")
			}
			break
		}
	}

	fmt.Println()
	fmt.Println("📋 network configuration:")
	fmt.Println("   all services are on the same docker network")
	fmt.Println("   mongodb is accessible via hostname: mongodb")
	fmt.Println("   mongodb port: 27017")
	fmt.Println()
	fmt.Println("💡 connection strings:")
	fmt.Println("   from containers: mongodb://mongodb:27017/kubeorchestra")
	fmt.Println("   from host:       mongodb://localhost:27017/kubeorchestra")

	return nil
}
