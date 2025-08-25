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
		fmt.Println("⚠️  docker not found. installing docker...")
		if err := installDocker(); err != nil {
			return fmt.Errorf("failed to install docker: %w. please install manually", err)
		}
		fmt.Println("✅ docker installed successfully")
	}
	
	// check if docker daemon is running
	if err := checkCommand("docker", "info"); err != nil {
		fmt.Println("⚠️  docker daemon is not running. starting docker...")
		if err := startDockerDaemon(); err != nil {
			return fmt.Errorf("failed to start docker daemon: %w. please start manually", err)
		}
		fmt.Println("✅ docker daemon started")
	}
	
	// check docker compose (prefer v2)
	if err := checkCommand("docker", "compose", "version"); err != nil {
		// fallback to docker-compose v1
		if err := checkCommand("docker-compose", "--version"); err != nil {
			fmt.Println("⚠️  docker compose not found. installing docker compose...")
			if err := installDockerCompose(); err != nil {
				return fmt.Errorf("failed to install docker compose: %w. please install manually", err)
			}
			fmt.Println("✅ docker compose installed successfully")
		}
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

// installDocker installs docker based on the OS
func installDocker() error {
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		fmt.Println("   installing docker via apt...")
		
		// update package list
		exec.Command("apt-get", "update").Run()
		
		// install dependencies
		exec.Command("apt-get", "install", "-y", "ca-certificates", "curl", "gnupg", "lsb-release").Run()
		
		// add docker's official gpg key
		exec.Command("bash", "-c", "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg").Run()
		
		// set up stable repository
		exec.Command("bash", "-c", `echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null`).Run()
		
		// update package list again
		exec.Command("apt-get", "update").Run()
		
		// install docker engine
		installCmd := exec.Command("apt-get", "install", "-y", "docker-ce", "docker-ce-cli", "containerd.io")
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		return installCmd.Run()
	}
	
	if _, err := exec.LookPath("brew"); err == nil {
		fmt.Println("   installing docker via homebrew...")
		cmd := exec.Command("brew", "install", "--cask", "docker")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	
	return fmt.Errorf("automatic installation not supported for this os")
}

// installDockerCompose installs docker-compose
func installDockerCompose() error {
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		fmt.Println("   installing docker-compose...")
		
		// download docker-compose binary
		downloadCmd := exec.Command("bash", "-c", 
			`curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose`)
		downloadCmd.Stdout = os.Stdout
		downloadCmd.Stderr = os.Stderr
		if err := downloadCmd.Run(); err != nil {
			return err
		}
		
		// make it executable
		return exec.Command("chmod", "+x", "/usr/local/bin/docker-compose").Run()
	}
	
	if _, err := exec.LookPath("brew"); err == nil {
		fmt.Println("   installing docker-compose via homebrew...")
		cmd := exec.Command("brew", "install", "docker-compose")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	
	return fmt.Errorf("automatic installation not supported for this os")
}

// getDockerComposeCommand returns the appropriate docker-compose command
func getDockerComposeCommand() []string {
	// prefer docker compose v2
	if err := checkCommand("docker", "compose", "version"); err == nil {
		return []string{"docker", "compose"}
	}
	// fallback to docker-compose v1
	return []string{"docker-compose"}
}

// startDockerDaemon attempts to start the docker daemon
func startDockerDaemon() error {
	// try systemctl first (systemd systems)
	if err := checkCommand("systemctl", "--version"); err == nil {
		fmt.Println("   starting docker with systemctl...")
		if err := exec.Command("systemctl", "start", "docker").Run(); err == nil {
			// enable docker to start on boot
			exec.Command("systemctl", "enable", "docker").Run()
			return nil
		}
	}
	
	// try service command (init.d systems)
	if err := checkCommand("service", "--version"); err == nil {
		fmt.Println("   starting docker with service...")
		return exec.Command("service", "docker", "start").Run()
	}
	
	// for macos, docker desktop needs to be opened
	if _, err := exec.LookPath("open"); err == nil {
		fmt.Println("   opening docker desktop...")
		if err := exec.Command("open", "-a", "Docker").Run(); err == nil {
			fmt.Println("   waiting for docker to start...")
			// wait for docker to be ready (max 30 seconds)
			for i := 0; i < 30; i++ {
				if err := checkCommand("docker", "info"); err == nil {
					return nil
				}
				exec.Command("sleep", "1").Run()
			}
			return fmt.Errorf("docker desktop did not start in time")
		}
	}
	
	return fmt.Errorf("unable to start docker daemon automatically")
}