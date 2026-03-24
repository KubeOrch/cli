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

// isDebian returns true if the current system is Debian/Ubuntu-based.
func isDebian() bool {
	_, err := os.Stat("/etc/debian_version")
	return err == nil
}

// hasHomebrew returns true if Homebrew is available on the system.
func hasHomebrew() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

// runCommand executes a command with stdout/stderr piped to the terminal.
func runCommand(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// runShell executes a shell script via bash.
func runShell(script string) error {
	c := exec.Command("bash", "-c", script)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// installViaApt updates package lists and installs the given packages.
func installViaApt(name string, packages []string) error {
	fmt.Printf("   installing %s via apt...\n", name)
	if err := runCommand("apt-get", "update"); err != nil {
		return fmt.Errorf("failed to update package list: %w", err)
	}
	args := append([]string{"install", "-y"}, packages...)
	if err := runCommand("apt-get", args...); err != nil {
		return fmt.Errorf("failed to install %s: %w", name, err)
	}
	return nil
}

// installViaBrew installs a package using Homebrew.
func installViaBrew(name, brewPkg string, cask bool) error {
	fmt.Printf("   installing %s via homebrew...\n", name)
	args := []string{"install"}
	if cask {
		args = append(args, "--cask")
	}
	args = append(args, brewPkg)
	if err := runCommand("brew", args...); err != nil {
		return fmt.Errorf("failed to install %s via brew: %w", name, err)
	}
	return nil
}

func installDocker() error {
	if isDebian() {
		if err := installViaApt("docker dependencies", []string{"ca-certificates", "curl", "gnupg", "lsb-release"}); err != nil {
			return err
		}

		gpgScript := "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | " +
			"gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg"
		if err := runShell(gpgScript); err != nil {
			return fmt.Errorf("failed to add docker gpg key: %w", err)
		}

		repoScript := `echo "deb [arch=$(dpkg --print-architecture) ` +
			`signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] ` +
			`https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | ` +
			`tee /etc/apt/sources.list.d/docker.list > /dev/null`
		if err := runShell(repoScript); err != nil {
			return fmt.Errorf("failed to setup docker repository: %w", err)
		}

		if err := runCommand("apt-get", "update"); err != nil {
			return fmt.Errorf("failed to update package list after adding docker repo: %w", err)
		}
		if err := runCommand("apt-get", "install", "-y", "docker-ce", "docker-ce-cli", "containerd.io"); err != nil {
			return fmt.Errorf("failed to install docker: %w", err)
		}
		return nil
	}

	if hasHomebrew() {
		return installViaBrew("docker", "docker", true)
	}

	return fmt.Errorf("automatic installation of docker not supported for this os")
}

func installDockerCompose() error {
	if isDebian() {
		fmt.Println("   installing docker-compose...")
		composeURL := `https://github.com/docker/compose/releases/latest/download/` +
			`docker-compose-$(uname -s)-$(uname -m)`
		dlScript := fmt.Sprintf(`curl -L "%s" -o /usr/local/bin/docker-compose`, composeURL)
		if err := runShell(dlScript); err != nil {
			return fmt.Errorf("failed to download docker-compose: %w", err)
		}
		if err := runCommand("chmod", "+x", "/usr/local/bin/docker-compose"); err != nil {
			return fmt.Errorf("failed to make docker-compose executable: %w", err)
		}
		return nil
	}

	if hasHomebrew() {
		return installViaBrew("docker-compose", "docker-compose", false)
	}

	return fmt.Errorf("automatic installation of docker-compose not supported for this os")
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
