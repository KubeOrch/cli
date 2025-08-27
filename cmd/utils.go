package cmd

import (
	"fmt"
	"os"
	"os/exec"
)

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info != nil && info.IsDir()
}

func checkCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func validateDockerCompose() error {
	if err := checkCommand("docker", "--version"); err != nil {
		fmt.Println("⚠️  docker not found. installing docker...")
		if err := installDocker(); err != nil {
			return fmt.Errorf("failed to install docker: %w. please install manually", err)
		}
		fmt.Println("✅ docker installed successfully")
	}

	if err := checkCommand("docker", "info"); err != nil {
		fmt.Println("⚠️  docker daemon is not running. starting docker...")
		if err := startDockerDaemon(); err != nil {
			return fmt.Errorf("failed to start docker daemon: %w. please start manually", err)
		}
		fmt.Println("✅ docker daemon started")
	}

	if err := checkCommand("docker", "compose", "version"); err != nil {
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

func getComposeFile(uiLocal, coreLocal bool) string {
	switch {
	case !uiLocal && !coreLocal:
		return "docker/docker-compose.prod.yml"
	case uiLocal && coreLocal:
		return "docker/docker-compose.dev.yml"
	case uiLocal:
		return "docker/docker-compose.hybrid-ui.yml"
	default:
		return "docker/docker-compose.hybrid-core.yml"
	}
}

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

func installDocker() error {
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		fmt.Println("   installing docker via apt...")

		updateCmd := exec.Command("apt-get", "update")
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr
		if err := updateCmd.Run(); err != nil {
			return fmt.Errorf("failed to update package list: %w", err)
		}

		depsCmd := exec.Command("apt-get", "install", "-y", "ca-certificates", "curl", "gnupg", "lsb-release")
		depsCmd.Stdout = os.Stdout
		depsCmd.Stderr = os.Stderr
		if err := depsCmd.Run(); err != nil {
			return fmt.Errorf("failed to install docker dependencies: %w", err)
		}

		gpgScript := "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | " +
			"gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg"
		gpgCmd := exec.Command("bash", "-c", gpgScript)
		gpgCmd.Stdout = os.Stdout
		gpgCmd.Stderr = os.Stderr
		if err := gpgCmd.Run(); err != nil {
			return fmt.Errorf("failed to add docker gpg key: %w", err)
		}

		repoScript := `echo "deb [arch=$(dpkg --print-architecture) ` +
			`signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] ` +
			`https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | ` +
			`tee /etc/apt/sources.list.d/docker.list > /dev/null`
		repoCmd := exec.Command("bash", "-c", repoScript)
		repoCmd.Stdout = os.Stdout
		repoCmd.Stderr = os.Stderr
		if err := repoCmd.Run(); err != nil {
			return fmt.Errorf("failed to setup docker repository: %w", err)
		}

		update2Cmd := exec.Command("apt-get", "update")
		update2Cmd.Stdout = os.Stdout
		update2Cmd.Stderr = os.Stderr
		if err := update2Cmd.Run(); err != nil {
			return fmt.Errorf("failed to update package list after adding docker repo: %w", err)
		}

		installCmd := exec.Command("apt-get", "install", "-y", "docker-ce", "docker-ce-cli", "containerd.io")
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to install docker: %w", err)
		}
		return nil
	}

	if _, err := exec.LookPath("brew"); err == nil {
		fmt.Println("   installing docker via homebrew...")
		cmd := exec.Command("brew", "install", "--cask", "docker")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install docker via brew: %w", err)
		}
		return nil
	}

	return fmt.Errorf("automatic installation not supported for this os")
}

func installDockerCompose() error {
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		fmt.Println("   installing docker-compose...")

		downloadCmd := exec.Command("bash", "-c",
			`curl -L "https://github.com/docker/compose/releases/latest/download/`+
				`docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose`)
		downloadCmd.Stdout = os.Stdout
		downloadCmd.Stderr = os.Stderr
		if err := downloadCmd.Run(); err != nil {
			return fmt.Errorf("failed to download docker-compose: %w", err)
		}

		chmodCmd := exec.Command("chmod", "+x", "/usr/local/bin/docker-compose")
		if err := chmodCmd.Run(); err != nil {
			return fmt.Errorf("failed to make docker-compose executable: %w", err)
		}
		return nil
	}

	if _, err := exec.LookPath("brew"); err == nil {
		fmt.Println("   installing docker-compose via homebrew...")
		cmd := exec.Command("brew", "install", "docker-compose")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install docker-compose via brew: %w", err)
		}
		return nil
	}

	return fmt.Errorf("automatic installation not supported for this os")
}

func getDockerComposeCommand() []string {
	if err := checkCommand("docker", "compose", "version"); err == nil {
		return []string{"docker", "compose"}
	}
	return []string{"docker-compose"}
}

func startDockerDaemon() error {
	if err := checkCommand("systemctl", "--version"); err == nil {
		fmt.Println("   starting docker with systemctl...")
		startCmd := exec.Command("systemctl", "start", "docker")
		startCmd.Stdout = os.Stdout
		startCmd.Stderr = os.Stderr
		if err := startCmd.Run(); err != nil {
			return fmt.Errorf("failed to start docker with systemctl: %w", err)
		}

		enableCmd := exec.Command("systemctl", "enable", "docker")
		if err := enableCmd.Run(); err != nil {
			fmt.Printf("   warning: failed to enable docker on boot: %v\n", err)
		}
		return nil
	}

	if err := checkCommand("service", "--version"); err == nil {
		fmt.Println("   starting docker with service...")
		return exec.Command("service", "docker", "start").Run()
	}

	if _, err := exec.LookPath("open"); err == nil {
		fmt.Println("   opening docker desktop...")
		if err := exec.Command("open", "-a", "Docker").Run(); err == nil {
			fmt.Println("   waiting for docker to start...")
			for i := 0; i < 30; i++ {
				if err := checkCommand("docker", "info"); err == nil {
					return nil
				}
				_ = exec.Command("sleep", "1").Run()
			}
			return fmt.Errorf("docker desktop did not start in time")
		}
	}

	return fmt.Errorf("unable to start docker daemon automatically")
}
