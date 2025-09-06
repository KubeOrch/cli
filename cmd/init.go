package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var (
	forkUI      string
	forkCore    string
	skipDeps    bool
	autoInstall bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize KubeOrch development environment",
	Long: `Initialize the KubeOrch environment. 

Without flags: Sets up for production testing using Docker images (no repos cloned).
With --fork-ui or --fork-core: Clones repositories for development.

Examples:
  # Production setup (uses Docker images only)
  orchcli init
  
  # Clone official repos for development (internal team members)
  orchcli init --fork-ui= --fork-core=
  
  # Clone from your forks (external contributors)
  orchcli init --fork-ui=myuser/ui --fork-core=myuser/core
  
  # Clone only UI for frontend development
  orchcli init --fork-ui=
  
  # Clone only Core for backend development
  orchcli init --fork-core=`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&forkUI, "fork-ui", "", "Clone UI repository (use --fork-ui= for official, or --fork-ui=username/repo for fork)")
	initCmd.Flags().StringVar(&forkCore, "fork-core", "", "Clone Core repository (use --fork-core= for official, or --fork-core=username/repo for fork)")
	initCmd.Flags().BoolVar(&skipDeps, "skip-deps", false, "Skip dependency installation")
	initCmd.Flags().BoolVar(&autoInstall, "auto-install", true, "Automatically install missing dependencies (npm, go)")

	initCmd.Flags().Lookup("fork-ui").NoOptDefVal = "KubeOrch/ui"
	initCmd.Flags().Lookup("fork-core").NoOptDefVal = "KubeOrch/core"

	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	uiSet := cmd.Flags().Changed("fork-ui")
	coreSet := cmd.Flags().Changed("fork-core")

	if !uiSet && !coreSet {
		return setupProduction()
	}

	return setupDevelopment(uiSet, coreSet)
}

func setupProduction() error {
	fmt.Println("🚀 Setting up OrchCLI for production testing")
	fmt.Println("   No repositories will be cloned.")
	fmt.Println("   Docker images will be used for both UI and Core.")

	if err := validateDockerCompose(); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	const dirMode = 0750
	dirs := []string{"docker", "scripts"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Save project configuration
	if err := setProjectConfig(cwd, "", ""); err != nil {
		fmt.Printf("⚠️  warning: failed to save project configuration: %v\n", err)
	}

	fmt.Println("\n✅ Production environment ready!")
	fmt.Printf("📁 Project initialized at: %s\n", cwd)
	fmt.Println("\n📝 Image tags that will be used:")
	fmt.Println("   - ghcr.io/kubeorch/ui:latest")
	fmt.Println("   - ghcr.io/kubeorch/core:latest")
	fmt.Println("\n   You can specify versions with: orchcli start --version=v1.2.3")
	fmt.Println("   Run 'orchcli start' to start services with latest images")
	return nil
}

func setupDevelopment(cloneUI, cloneCore bool) error {
	fmt.Println("🔧 Setting up OrchCLI for development")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if cloneUI && forkUI == "" {
		forkUI = "KubeOrch/ui"
	}
	if cloneCore && forkCore == "" {
		forkCore = "KubeOrch/core"
	}

	if err := checkPrerequisites(); err != nil {
		return err
	}

	if err := validateAndCheckDirs(cloneUI, cloneCore); err != nil {
		return err
	}

	// Prepare tasks for concurrent execution
	var cloneTasks []Task

	if cloneUI && cloneCore {
		fmt.Println("📦 Cloning repositories concurrently...")
	}

	// UI cloning task
	var uiRepoURL string
	var uiIsFork bool
	var uiPath string
	if cloneUI {
		uiRepoURL, uiIsFork = determineRepoURL(forkUI, "KubeOrch/ui")
		uiPath = filepath.Join(cwd, "ui")
		cloneTasks = append(cloneTasks, Task{
			Action: func() error {
				return cloneRepo(uiRepoURL, uiPath)
			},
			Progress: NewProgressBar(fmt.Sprintf("Cloning UI from %s", uiRepoURL)),
			Name:     "Clone UI repository",
		})
	}

	// Core cloning task
	var coreRepoURL string
	var coreIsFork bool
	var corePath string
	if cloneCore {
		coreRepoURL, coreIsFork = determineRepoURL(forkCore, "KubeOrch/core")
		corePath = filepath.Join(cwd, "core")
		cloneTasks = append(cloneTasks, Task{
			Action: func() error {
				return cloneRepo(coreRepoURL, corePath)
			},
			Progress: NewProgressBar(fmt.Sprintf("Cloning Core from %s", coreRepoURL)),
			Name:     "Clone Core repository",
		})
	}

	// Execute cloning tasks concurrently
	if len(cloneTasks) > 0 {
		results := RunConcurrent(cloneTasks)
		if err := AggregateErrors(results); err != nil {
			return err
		}
	}

	// Setup upstreams for forks (sequential as they're quick)
	if cloneUI && uiIsFork {
		fmt.Println("🔗 Setting up upstream for UI fork...")
		if err := setupUpstream(uiPath, "https://github.com/KubeOrch/ui"); err != nil {
			return fmt.Errorf("failed to setup upstream for UI: %w", err)
		}
	}

	if cloneCore && coreIsFork {
		fmt.Println("🔗 Setting up upstream for Core fork...")
		if err := setupUpstream(corePath, "https://github.com/KubeOrch/core"); err != nil {
			return fmt.Errorf("failed to setup upstream for Core: %w", err)
		}
	}

	// Install dependencies concurrently
	if !skipDeps {
		var depTasks []Task

		if cloneUI {
			depTasks = append(depTasks, Task{
				Action: func() error {
					return installUIDependencies(uiPath)
				},
				Progress: NewProgressBar("Installing UI dependencies (npm install)"),
				Name:     "Install UI dependencies",
			})
		}

		if cloneCore {
			depTasks = append(depTasks, Task{
				Action: func() error {
					return installCoreDependencies(corePath)
				},
				Progress: NewProgressBar("Downloading Core dependencies (go mod download)"),
				Name:     "Download Core dependencies",
			})
		}

		if len(depTasks) > 0 {
			fmt.Println("\n📥 Installing dependencies concurrently...")
			results := RunConcurrent(depTasks)

			// Show warnings for failed dependencies but don't fail
			for _, result := range results {
				if result.Error != nil {
					if result.Name == "Install UI dependencies" {
						fmt.Printf("⚠️  warning: failed to install ui dependencies: %v\n", result.Error)
						fmt.Printf("   you can install them manually with: cd %s && npm install\n", uiPath)
					} else if result.Name == "Download Core dependencies" {
						fmt.Printf("⚠️  warning: failed to download core dependencies: %v\n", result.Error)
						fmt.Printf("   you can download them manually with: cd %s && go mod download\n", corePath)
					}
				}
			}
		}
	}

	// Save project configuration
	if err := setProjectConfig(cwd, uiPath, corePath); err != nil {
		fmt.Printf("⚠️  warning: failed to save project configuration: %v\n", err)
	}

	fmt.Println("\n✅ Development environment ready!")
	fmt.Printf("📁 Project initialized at: %s\n", cwd)
	fmt.Println("\n📝 Next steps:")

	switch {
	case cloneUI && cloneCore:
		fmt.Println("   1. Run 'orchcli start' to start both UI and Core locally")
	case cloneUI:
		fmt.Println("   1. Run 'orchcli start' to start UI locally with Core from Docker")
	case cloneCore:
		fmt.Println("   1. Run 'orchcli start' to start Core locally with UI from Docker")
	}

	fmt.Println("   2. Make your changes in the cloned repositories")
	fmt.Println("   3. Changes will hot-reload automatically")

	usingForks := (forkUI != "" && forkUI != "KubeOrch/ui") ||
		(forkCore != "" && forkCore != "KubeOrch/core")

	if usingForks {
		fmt.Println("\n🍴 Fork workflow detected (External Contributor):")
		fmt.Println("   1. Create a feature branch: git checkout -b feature/my-feature")
		fmt.Println("   2. Push to your fork: git push origin feature/my-feature")
		fmt.Println("   3. Create a pull request on GitHub")
	} else if cloneUI || cloneCore {
		fmt.Println("\n👥 Official repo workflow (Team Member):")
		fmt.Println("   1. Create a feature branch or work on main")
		fmt.Println("   2. Push directly: git push origin <branch>")
	}

	return nil
}

func determineRepoURL(repoName, defaultRepo string) (string, bool) {
	repoName = strings.TrimSpace(repoName)

	if repoName == defaultRepo || repoName == "" {
		return fmt.Sprintf("https://github.com/%s", defaultRepo), false
	}

	return fmt.Sprintf("https://github.com/%s", repoName), true
}

func validateRepoFormat(repo string) error {
	if repo == "" || repo == "KubeOrch/ui" || repo == "KubeOrch/core" {
		return nil
	}

	pattern := `^[a-zA-Z0-9]([a-zA-Z0-9-]{0,38}[a-zA-Z0-9])?/[a-zA-Z0-9]([a-zA-Z0-9-_.]{0,99}[a-zA-Z0-9])?$`
	matched, err := regexp.MatchString(pattern, repo)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("invalid repository format: %s (expected: username/repo)", repo)
	}
	return nil
}

func checkPrerequisites() error {
	if err := checkCommand("git", "--version"); err != nil {
		if autoInstall {
			fmt.Println("⚠️  git not found. installing git...")
			if err := installGit(); err != nil {
				return fmt.Errorf("failed to install git: %w. please install manually", err)
			}
			fmt.Println("✅ git installed successfully")
		} else {
			return fmt.Errorf("git is not installed. please install git first")
		}
	}

	if err := validateDockerCompose(); err != nil {
		return err
	}

	return nil
}

func validateAndCheckDirs(checkUI, checkCore bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if checkUI {
		if err := validateRepoFormat(forkUI); err != nil {
			return fmt.Errorf("invalid UI repository: %w", err)
		}
		uiPath := filepath.Join(cwd, "ui")
		if dirExists(uiPath) {
			return fmt.Errorf("UI directory already exists at %s. Please remove it first or use 'orchcli update'", uiPath)
		}
	}

	if checkCore {
		if err := validateRepoFormat(forkCore); err != nil {
			return fmt.Errorf("invalid Core repository: %w", err)
		}
		corePath := filepath.Join(cwd, "core")
		if dirExists(corePath) {
			return fmt.Errorf("core directory already exists at %s. Please remove it first or use 'orchcli update'", corePath)
		}
	}

	return nil
}

func cloneRepo(url, path string) error {
	if dirExists(path) {
		return fmt.Errorf("directory %s already exists", path)
	}

	cmd := exec.Command("git", "clone", url, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

func setupUpstream(repoPath, upstreamURL string) error {
	cmd := exec.Command("git", "remote", "add", "upstream", upstreamURL)
	cmd.Dir = repoPath

	if err := cmd.Run(); err != nil {
		checkCmd := exec.Command("git", "remote", "get-url", "upstream")
		checkCmd.Dir = repoPath
		if checkErr := checkCmd.Run(); checkErr == nil {
			updateCmd := exec.Command("git", "remote", "set-url", "upstream", upstreamURL)
			updateCmd.Dir = repoPath
			return updateCmd.Run()
		}
		return err
	}

	fetchCmd := exec.Command("git", "fetch", "upstream")
	fetchCmd.Dir = repoPath
	fetchCmd.Stdout = os.Stdout
	fetchCmd.Stderr = os.Stderr

	return fetchCmd.Run()
}

func installUIDependencies(uiPath string) error {
	if err := checkCommand("npm", "--version"); err != nil {
		if autoInstall {
			fmt.Println("⚠️  npm not found. installing node.js and npm...")
			if err := installNodeJS(); err != nil {
				return fmt.Errorf("failed to install node.js: %w. please install manually from https://nodejs.org/", err)
			}
			fmt.Println("✅ node.js and npm installed successfully")
		} else {
			return fmt.Errorf("npm is not installed. please install node.js and npm from https://nodejs.org/")
		}
	}

	fmt.Println("   this may take a few minutes...")
	cmd := exec.Command("npm", "install")
	cmd.Dir = uiPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func installCoreDependencies(corePath string) error {
	if err := checkCommand("go", "version"); err != nil {
		if autoInstall {
			fmt.Println("⚠️  go not found. installing go...")
			if err := installGo(); err != nil {
				return fmt.Errorf("failed to install go: %w. please install manually from https://go.dev/", err)
			}
			fmt.Println("✅ go installed successfully")
		} else {
			return fmt.Errorf("go is not installed. please install go from https://go.dev/")
		}
	}

	fmt.Println("   downloading go modules...")
	cmd := exec.Command("go", "mod", "download")
	cmd.Dir = corePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func installNodeJS() error {
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		fmt.Println("   installing via apt...")

		updateCmd := exec.Command("apt-get", "update")
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr
		if err := updateCmd.Run(); err != nil {
			return fmt.Errorf("failed to update package list: %w", err)
		}

		curlCmd := exec.Command("apt-get", "install", "-y", "curl")
		curlCmd.Stdout = os.Stdout
		curlCmd.Stderr = os.Stderr
		if err := curlCmd.Run(); err != nil {
			return fmt.Errorf("failed to install curl: %w", err)
		}

		setupCmd := exec.Command("bash", "-c", "curl -fsSL https://deb.nodesource.com/setup_lts.x | bash -")
		setupCmd.Stdout = os.Stdout
		setupCmd.Stderr = os.Stderr
		if err := setupCmd.Run(); err != nil {
			return fmt.Errorf("failed to setup node.js repository: %w", err)
		}

		installCmd := exec.Command("apt-get", "install", "-y", "nodejs")
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to install nodejs: %w", err)
		}
		return nil
	}

	if _, err := exec.LookPath("brew"); err == nil {
		fmt.Println("   installing via homebrew...")
		cmd := exec.Command("brew", "install", "node")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install node via brew: %w", err)
		}
		return nil
	}

	return fmt.Errorf("automatic installation not supported for this os")
}

func installGo() error {
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		fmt.Println("   installing go via apt...")

		updateCmd := exec.Command("apt-get", "update")
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr
		if err := updateCmd.Run(); err != nil {
			return fmt.Errorf("failed to update package list: %w", err)
		}

		installCmd := exec.Command("apt-get", "install", "-y", "golang-go")
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to install golang-go: %w", err)
		}
		return nil
	}

	if _, err := exec.LookPath("brew"); err == nil {
		fmt.Println("   installing via homebrew...")
		cmd := exec.Command("brew", "install", "go")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install go via brew: %w", err)
		}
		return nil
	}

	return fmt.Errorf("automatic installation not supported for this os")
}

func installGit() error {
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		fmt.Println("   installing git via apt...")

		updateCmd := exec.Command("apt-get", "update")
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr
		if err := updateCmd.Run(); err != nil {
			return fmt.Errorf("failed to update package list: %w", err)
		}

		installCmd := exec.Command("apt-get", "install", "-y", "git")
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to install git: %w", err)
		}
		return nil
	}

	if _, err := exec.LookPath("brew"); err == nil {
		fmt.Println("   installing via homebrew...")
		cmd := exec.Command("brew", "install", "git")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install git via brew: %w", err)
		}
		return nil
	}

	return fmt.Errorf("automatic installation not supported for this os")
}
