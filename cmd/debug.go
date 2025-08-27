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
			
			fmt.Printf("   testing from %s to postgres...\n", container)
			
			pingCmd := exec.Command("docker", "exec", container, "sh", "-c", "nc -zv postgres 5432 2>&1 || echo 'nc not available'")
			pingOutput, pingErr := pingCmd.Output()
			if pingErr != nil {
				fmt.Printf("   ⚠️  could not test connectivity: %v\n", pingErr)
			} else {
				output := strings.TrimSpace(string(pingOutput))
				if strings.Contains(output, "succeeded") || strings.Contains(output, "open") {
					fmt.Println("   ✅ network connectivity to postgres:5432 is working")
				} else if strings.Contains(output, "not available") {
					telnetCmd := exec.Command("docker", "exec", container, "sh", "-c", "timeout 2 telnet postgres 5432 2>&1 || echo 'connection test failed'")
					telnetOutput, _ := telnetCmd.Output()
					if strings.Contains(string(telnetOutput), "Connected") {
						fmt.Println("   ✅ network connectivity to postgres:5432 is working")
					} else {
						fmt.Println("   ❌ cannot connect to postgres:5432")
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
	fmt.Println("🔌 testing direct postgres access:")
	
	postgresContainers := []string{
		"kubeorchestra-postgres",
		"kubeorchestra-postgres-dev",
		"kubeorchestra-postgres-hybrid",
	}
	
	for _, container := range postgresContainers {
		testCmd := exec.Command("docker", "exec", container, "pg_isready", "-U", "kubeorchestra", "-d", "kubeorchestra")
		if output, err := testCmd.Output(); err == nil {
			fmt.Printf("   ✅ %s is ready: %s", container, strings.TrimSpace(string(output)))
			
			connTestCmd := exec.Command("docker", "exec", container, "psql", "-U", "kubeorchestra", "-d", "kubeorchestra", "-c", "SELECT 1")
			if _, err := connTestCmd.Output(); err == nil {
				fmt.Println("   ✅ database connection test successful")
			}
			break
		}
	}
	
	fmt.Println()
	fmt.Println("📋 network configuration:")
	fmt.Println("   all services are on the same docker network")
	fmt.Println("   postgres is accessible via hostname: postgres")
	fmt.Println("   postgres port: 5432")
	fmt.Println()
	fmt.Println("💡 connection strings:")
	fmt.Println("   from containers: postgres://kubeorchestra:kubeorchestra@postgres:5432/kubeorchestra")
	fmt.Println("   from host:       postgres://kubeorchestra:kubeorchestra@localhost:5432/kubeorchestra")
	
	return nil
}