package cmd

import (
	"fmt"
	"os"
	"os/exec"
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
	Short: "Initialize KubeOrchestra development environment",
	Long: `Initialize the KubeOrchestra environment. 

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

	initCmd.Flags().Lookup("fork-ui").NoOptDefVal = "KubeOrchestra/ui"
	initCmd.Flags().Lookup("fork-core").NoOptDefVal = "KubeOrchestra/core"

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

	dirs := []string{"docker", "scripts"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	fmt.Println("\n✅ Production environment ready!")
	fmt.Println("\n📝 Image tags that will be used:")
	fmt.Println("   - ghcr.io/kubeorchestra/ui:latest")
	fmt.Println("   - ghcr.io/kubeorchestra/core:latest")
	fmt.Println("\n   You can specify versions with: orchcli start --version=v1.2.3")
	fmt.Println("   Run 'orchcli start' to start services with latest images")
	return nil
}

func setupDevelopment(cloneUI, cloneCore bool) error {
	fmt.Println("🔧 Setting up OrchCLI for development")

	if cloneUI && forkUI == "" {
		forkUI = "KubeOrchestra/ui"
	}
	if cloneCore && forkCore == "" {
		forkCore = "KubeOrchestra/core"
	}

	if err := checkPrerequisites(); err != nil {
		return err
	}

	if err := validateAndCheckDirs(cloneUI, cloneCore); err != nil {
		return err
	}

	if cloneUI {
		repoURL, isFork := determineRepoURL(forkUI, "KubeOrchestra/ui")
		fmt.Printf("📦 Cloning UI repository from %s...\n", repoURL)

		if err := cloneRepo(repoURL, "./ui"); err != nil {
			return fmt.Errorf("failed to clone UI repository: %w", err)
		}

		if isFork {
			fmt.Println("🔗 Setting up upstream for UI fork...")
			if err := setupUpstream("./ui", "https://github.com/KubeOrchestra/ui"); err != nil {
				return fmt.Errorf("failed to setup upstream for UI: %w", err)
			}
		}

		if !skipDeps {
			fmt.Println("📥 installing ui dependencies...")
			if err := installUIDependencies(); err != nil {
				fmt.Printf("⚠️  warning: failed to install ui dependencies: %v\n", err)
				fmt.Println("   you can install them manually with: cd ui && npm install")
			}
		}
	}

	if cloneCore {
		repoURL, isFork := determineRepoURL(forkCore, "KubeOrchestra/core")
		fmt.Printf("📦 Cloning Core repository from %s...\n", repoURL)

		if err := cloneRepo(repoURL, "./core"); err != nil {
			return fmt.Errorf("failed to clone Core repository: %w", err)
		}

		if isFork {
			fmt.Println("🔗 Setting up upstream for Core fork...")
			if err := setupUpstream("./core", "https://github.com/KubeOrchestra/core"); err != nil {
				return fmt.Errorf("failed to setup upstream for Core: %w", err)
			}
		}

		if !skipDeps {
			fmt.Println("📥 downloading core dependencies...")
			if err := installCoreDependencies(); err != nil {
				fmt.Printf("⚠️  warning: failed to download core dependencies: %v\n", err)
				fmt.Println("   you can download them manually with: cd core && go mod download")
			}
		}
	}

	fmt.Println("\n✅ Development environment ready!")
	fmt.Println("\n📝 Next steps:")

	if cloneUI && cloneCore {
		fmt.Println("   1. Run 'orchcli start' to start both UI and Core locally")
	} else if cloneUI {
		fmt.Println("   1. Run 'orchcli start' to start UI locally with Core from Docker")
	} else if cloneCore {
		fmt.Println("   1. Run 'orchcli start' to start Core locally with UI from Docker")
	}

	fmt.Println("   2. Make your changes in the cloned repositories")
	fmt.Println("   3. Changes will hot-reload automatically")

	usingForks := (forkUI != "" && forkUI != "KubeOrchestra/ui") ||
		(forkCore != "" && forkCore != "KubeOrchestra/core")

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
	if repo == "" || repo == "KubeOrchestra/ui" || repo == "KubeOrchestra/core" {
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
	if checkUI {
		if err := validateRepoFormat(forkUI); err != nil {
			return fmt.Errorf("invalid UI repository: %w", err)
		}
		if dirExists("./ui") {
			return fmt.Errorf("UI directory already exists. Please remove it first or use 'orchcli update'")
		}
	}

	if checkCore {
		if err := validateRepoFormat(forkCore); err != nil {
			return fmt.Errorf("invalid Core repository: %w", err)
		}
		if dirExists("./core") {
			return fmt.Errorf("Core directory already exists. Please remove it first or use 'orchcli update'")
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

func installUIDependencies() error {
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
	cmd.Dir = "./ui"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func installCoreDependencies() error {
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
	cmd.Dir = "./core"
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
