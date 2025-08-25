package cmd

import (
	"fmt"
	"os"
	"os/exec"
)

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info != nil && info.IsDir()
}

// checkCommand checks if a command is available
func checkCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// validateDockerCompose checks if docker and docker-compose are available
func validateDockerCompose() error {
	// check docker
	if err := checkCommand("docker", "--version"); err != nil {
		return fmt.Errorf("docker is not installed. please install docker first")
	}
	
	// check if docker daemon is running
	if err := checkCommand("docker", "info"); err != nil {
		return fmt.Errorf("docker daemon is not running. please start docker")
	}
	
	// check docker-compose
	if err := checkCommand("docker-compose", "--version"); err != nil {
		// try docker compose (v2)
		if err := checkCommand("docker", "compose", "version"); err != nil {
			return fmt.Errorf("docker-compose is not installed. please install docker-compose")
		}
		// docker compose v2 is available, we'll use it via alias
		fmt.Println("   using docker compose v2")
	}
	
	return nil
}

// getComposeFile returns the appropriate docker-compose file based on what's initialized
func getComposeFile(uiLocal, coreLocal bool) string {
	if !uiLocal && !coreLocal {
		return "docker/docker-compose.prod.yml"
	} else if uiLocal && coreLocal {
		return "docker/docker-compose.dev.yml"
	} else if uiLocal {
		return "docker/docker-compose.hybrid-ui.yml"
	} else {
		return "docker/docker-compose.hybrid-core.yml"
	}
}

// joinArgs joins command arguments into a string for display
func joinArgs(args []string) string {
	result := ""
	for _, arg := range args {
		if result != "" {
			result += " "
		}
		result += arg
	}
	return result
}