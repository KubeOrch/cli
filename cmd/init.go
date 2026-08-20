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

const (
	defaultUIRepo   = "KubeOrch/ui"
	defaultCoreRepo = "KubeOrch/core"
)

var (
	forkUI           string
	forkCore         string
	existingUIPath   string
	existingCorePath string
	skipDeps         bool
	autoInstall      bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize KubeOrch development environment",
	Long: `Initialize the KubeOrch environment. 

Without flags: Sets up for production testing using Docker images (no repos cloned).
With --fork-ui or --fork-core: Clones repositories for development.
With --ui-path or --core-path: Uses an existing local checkout.

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
  orchcli init --fork-core=

  # Use existing checkouts without cloning them
  orchcli init --ui-path ./ui --core-path ./core`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&forkUI, "fork-ui", "", "Clone UI repository (use --fork-ui= for official, or --fork-ui=username/repo for fork)")
	initCmd.Flags().StringVar(&forkCore, "fork-core", "", "Clone Core repository (use --fork-core= for official, or --fork-core=username/repo for fork)")
	initCmd.Flags().StringVar(&existingUIPath, "ui-path", "", "Use an existing UI checkout instead of cloning")
	initCmd.Flags().StringVar(&existingCorePath, "core-path", "", "Use an existing Core checkout instead of cloning")
	initCmd.Flags().BoolVar(&skipDeps, "skip-deps", false, "Skip dependency installation")
	initCmd.Flags().BoolVar(&autoInstall, "auto-install", true, "Automatically install missing dependencies (npm, go)")

	initCmd.Flags().Lookup("fork-ui").NoOptDefVal = defaultUIRepo
	initCmd.Flags().Lookup("fork-core").NoOptDefVal = defaultCoreRepo

	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cloneUI := cmd.Flags().Changed("fork-ui")
	cloneCore := cmd.Flags().Changed("fork-core")
	useExistingUI := cmd.Flags().Changed("ui-path")
	useExistingCore := cmd.Flags().Changed("core-path")

	if cloneUI && useExistingUI {
		return fmt.Errorf("--fork-ui and --ui-path cannot be used together")
	}
	if cloneCore && useExistingCore {
		return fmt.Errorf("--fork-core and --core-path cannot be used together")
	}

	if !cloneUI && !cloneCore && !useExistingUI && !useExistingCore {
		return setupProduction()
	}

	return setupDevelopment(cloneUI, cloneCore, useExistingUI, useExistingCore)
}

func setupProduction() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	existingProject := loadProjectAtPath(cwd)
	if existingProject != nil && (existingProject.UIPath != "" || existingProject.CorePath != "") {
		fmt.Println("🔧 Refreshing existing OrchCLI development environment")
		fmt.Println("   Preserving configured UI and Core source paths.")
	} else {
		fmt.Println("🚀 Setting up OrchCLI for production testing")
		fmt.Println("   No repositories will be cloned.")
		fmt.Println("   Docker images will be used for both UI and Core.")
	}

	if err := validateDockerCompose(); err != nil {
		return err
	}

	dirs := []string{"docker", "scripts"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Write embedded docker-compose files
	if err := writeEmbeddedComposeFiles(filepath.Join(cwd, "docker")); err != nil {
		return fmt.Errorf("failed to write docker-compose files: %w", err)
	}

	uiPath, corePath := "", ""
	if existingProject != nil {
		uiPath = existingProject.UIPath
		corePath = existingProject.CorePath
	}
	if err := setProjectConfig(cwd, uiPath, corePath); err != nil {
		return fmt.Errorf("failed to save project configuration: %w", err)
	}
	if uiPath != "" || corePath != "" {
		fmt.Println("\n✅ Development environment refreshed without changing its mode!")
		fmt.Printf("📁 Project initialized at: %s\n", cwd)
		fmt.Println("\n   Run 'orchcli start' to start the configured development services")
		return nil
	}

	fmt.Println("\n✅ Production environment ready!")
	fmt.Printf("📁 Project initialized at: %s\n", cwd)
	fmt.Println("\n📝 Docker images that will be used:")
	fmt.Println("   - ghcr.io/kubeorch/core:v0.0.3 (digest pinned)")
	fmt.Println("   - ghcr.io/kubeorch/ui:v0.0.3 (digest pinned)")
	fmt.Println("\n   Run 'orchcli start' to start the pinned release images")
	return nil
}

func loadProjectAtPath(projectPath string) *ProjectConfig {
	project, err := loadProjectMarker(projectPath)
	if err != nil || filepath.Clean(project.Path) != filepath.Clean(projectPath) {
		return nil
	}
	return project
}

type developmentSetup struct {
	projectPath string
	uiPath      string
	corePath    string
	uiRepoURL   string
	coreRepoURL string
	cloneUI     bool
	cloneCore   bool
	uiIsFork    bool
	coreIsFork  bool
}

func setupDevelopment(cloneUI, cloneCore, useExistingUI, useExistingCore bool) error {
	fmt.Println("🔧 Setting up OrchCLI for development")

	setup, err := prepareDevelopmentSetup(cloneUI, cloneCore, useExistingUI, useExistingCore)
	if err != nil {
		return err
	}
	if err := setup.cloneRepositories(); err != nil {
		return err
	}
	if err := writeDevelopmentComposeFiles(setup.projectPath); err != nil {
		return err
	}
	if err := setup.configureUpstreams(); err != nil {
		return err
	}
	setup.installDependencies()
	setup.generateConfigFiles()

	if err := setProjectConfig(setup.projectPath, setup.uiPath, setup.corePath); err != nil {
		return fmt.Errorf("failed to save project configuration: %w", err)
	}
	setup.printSummary()
	return nil
}

func prepareDevelopmentSetup(cloneUI, cloneCore, useExistingUI, useExistingCore bool) (*developmentSetup, error) {
	projectPath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	if cloneUI && forkUI == "" {
		forkUI = defaultUIRepo
	}
	if cloneCore && forkCore == "" {
		forkCore = defaultCoreRepo
	}
	if prerequisiteErr := checkPrerequisites(cloneUI || cloneCore); prerequisiteErr != nil {
		return nil, prerequisiteErr
	}
	if validationErr := validateAndCheckDirs(cloneUI, cloneCore); validationErr != nil {
		return nil, validationErr
	}

	setup := &developmentSetup{projectPath: projectPath, cloneUI: cloneUI, cloneCore: cloneCore}
	if useExistingUI {
		setup.uiPath, err = resolveExistingCheckout(projectPath, existingUIPath, "UI", "package.json")
		if err != nil {
			return nil, err
		}
	}
	if useExistingCore {
		setup.corePath, err = resolveExistingCheckout(projectPath, existingCorePath, "Core", "go.mod")
		if err != nil {
			return nil, err
		}
	}
	if cloneUI {
		setup.uiRepoURL, setup.uiIsFork = determineRepoURL(forkUI, defaultUIRepo)
		setup.uiPath = filepath.Join(projectPath, "ui")
	}
	if cloneCore {
		setup.coreRepoURL, setup.coreIsFork = determineRepoURL(forkCore, defaultCoreRepo)
		setup.corePath = filepath.Join(projectPath, "core")
	}
	return setup, nil
}

func (setup *developmentSetup) hasUI() bool {
	return setup.uiPath != ""
}

func (setup *developmentSetup) hasCore() bool {
	return setup.corePath != ""
}

func (setup *developmentSetup) cloneRepositories() error {
	var tasks []Task
	if setup.cloneUI && setup.cloneCore {
		fmt.Println("📦 Cloning repositories concurrently...")
	}
	if setup.cloneUI {
		tasks = append(tasks, Task{
			Action:   func() error { return cloneRepo(setup.uiRepoURL, setup.uiPath) },
			Progress: NewProgressBar(fmt.Sprintf("Cloning UI from %s", setup.uiRepoURL)),
			Name:     "Clone UI repository",
		})
	}
	if setup.cloneCore {
		tasks = append(tasks, Task{
			Action:   func() error { return cloneRepo(setup.coreRepoURL, setup.corePath) },
			Progress: NewProgressBar(fmt.Sprintf("Cloning Core from %s", setup.coreRepoURL)),
			Name:     "Clone Core repository",
		})
	}
	if len(tasks) == 0 {
		return nil
	}
	return AggregateErrors(RunConcurrent(tasks))
}

func writeDevelopmentComposeFiles(projectPath string) error {
	dockerDir := filepath.Join(projectPath, "docker")
	if err := os.MkdirAll(dockerDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create docker directory: %w", err)
	}
	if err := writeEmbeddedComposeFiles(dockerDir); err != nil {
		return fmt.Errorf("failed to write docker-compose files: %w", err)
	}
	return nil
}

func (setup *developmentSetup) configureUpstreams() error {
	if setup.cloneUI && setup.uiIsFork {
		fmt.Println("🔗 Setting up upstream for UI fork...")
		if err := setupUpstream(setup.uiPath, "https://github.com/"+defaultUIRepo); err != nil {
			return fmt.Errorf("failed to setup upstream for UI: %w", err)
		}
	}
	if setup.cloneCore && setup.coreIsFork {
		fmt.Println("🔗 Setting up upstream for Core fork...")
		if err := setupUpstream(setup.corePath, "https://github.com/"+defaultCoreRepo); err != nil {
			return fmt.Errorf("failed to setup upstream for Core: %w", err)
		}
	}
	return nil
}

func (setup *developmentSetup) installDependencies() {
	if skipDeps {
		return
	}

	var tasks []Task
	if setup.hasUI() {
		tasks = append(tasks, Task{
			Action:   func() error { return installUIDependencies(setup.uiPath) },
			Progress: NewProgressBar("Installing UI dependencies (npm install)"),
			Name:     "Install UI dependencies",
		})
	}
	if setup.hasCore() {
		tasks = append(tasks, Task{
			Action:   func() error { return installCoreDependencies(setup.corePath) },
			Progress: NewProgressBar("Downloading Core dependencies (go mod download)"),
			Name:     "Download Core dependencies",
		})
	}
	if len(tasks) == 0 {
		return
	}

	fmt.Println("\n📥 Installing dependencies concurrently...")
	setup.printDependencyWarnings(RunConcurrent(tasks))
}

func (setup *developmentSetup) printDependencyWarnings(results []TaskResult) {
	for _, result := range results {
		if result.Error == nil {
			continue
		}
		switch result.Name {
		case "Install UI dependencies":
			fmt.Printf("⚠️  warning: failed to install ui dependencies: %v\n", result.Error)
			fmt.Printf("   you can install them manually with: cd %s && npm install\n", setup.uiPath)
		case "Download Core dependencies":
			fmt.Printf("⚠️  warning: failed to download core dependencies: %v\n", result.Error)
			fmt.Printf("   you can download them manually with: cd %s && go mod download\n", setup.corePath)
		}
	}
}

func (setup *developmentSetup) generateConfigFiles() {
	if setup.hasCore() {
		generateCoreConfig(setup.corePath)
	}
	if setup.hasUI() {
		generateUIConfig(setup.uiPath)
	}
}

func generateCoreConfig(corePath string) {
	configPath := filepath.Join(corePath, "config.yaml")
	_, statErr := os.Stat(configPath)
	configExists := statErr == nil
	if err := writeConfigYAML(configPath); err != nil {
		fmt.Printf("⚠️  warning: failed to generate config.yaml: %v\n", err)
	} else if configExists {
		fmt.Println("✅ Using existing core/config.yaml")
	} else {
		fmt.Println("✅ Generated core/config.yaml with default values")
	}
}

func generateUIConfig(uiPath string) {
	envPath := filepath.Join(uiPath, ".env.local")
	_, statErr := os.Stat(envPath)
	envExists := statErr == nil
	if err := writeEnvLocal(envPath); err != nil {
		fmt.Printf("⚠️  warning: failed to generate .env.local: %v\n", err)
	} else if envExists {
		fmt.Println("✅ Using existing ui/.env.local")
	} else {
		fmt.Println("✅ Generated ui/.env.local with default API URL")
	}
}

func (setup *developmentSetup) printSummary() {
	fmt.Println("\n✅ Development environment ready!")
	fmt.Printf("📁 Project initialized at: %s\n", setup.projectPath)
	fmt.Println("\n📝 Next steps:")

	switch {
	case setup.hasUI() && setup.hasCore():
		fmt.Println("   1. Run 'orchcli start' to start both UI and Core locally")
	case setup.hasUI():
		fmt.Println("   1. Run 'orchcli start' to start UI locally with Core from Docker")
	case setup.hasCore():
		fmt.Println("   1. Run 'orchcli start' to start Core locally with UI from Docker")
	}
	fmt.Println("   2. Make your changes in the source repositories")
	fmt.Println("   3. UI changes hot-reload; restart a host Core process after Core changes")

	if setup.uiIsFork || setup.coreIsFork {
		fmt.Println("\n🍴 Fork workflow detected (External Contributor):")
		fmt.Println("   1. Create a feature branch: git checkout -b feature/my-feature")
		fmt.Println("   2. Push to your fork: git push origin feature/my-feature")
		fmt.Println("   3. Create a pull request on GitHub")
	} else if setup.hasUI() || setup.hasCore() {
		fmt.Println("\n👥 Official repo workflow (Team Member):")
		fmt.Println("   1. Create a feature branch or work on main")
		fmt.Println("   2. Push directly: git push origin <branch>")
	}
}

func resolveExistingCheckout(projectPath, sourcePath, component, markerFile string) (string, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return "", fmt.Errorf("--%s-path requires a directory", strings.ToLower(component))
	}
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(projectPath, sourcePath)
	}
	absPath, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s checkout %q: %w", component, sourcePath, err)
	}
	absPath = filepath.Clean(absPath)
	if !dirExists(absPath) {
		return "", fmt.Errorf("%s checkout does not exist or is not a directory: %s", component, absPath)
	}
	info, statErr := os.Stat(filepath.Join(absPath, markerFile))
	if statErr != nil || info.IsDir() {
		return "", fmt.Errorf("%s checkout at %s is missing %s", component, absPath, markerFile)
	}
	return absPath, nil
}

func determineRepoURL(repoName, defaultRepo string) (string, bool) {
	repoName = strings.TrimSpace(repoName)

	if repoName == defaultRepo || repoName == "" {
		return fmt.Sprintf("https://github.com/%s", defaultRepo), false
	}

	return fmt.Sprintf("https://github.com/%s", repoName), true
}

func validateRepoFormat(repo string) error {
	if repo == "" || repo == defaultUIRepo || repo == defaultCoreRepo {
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

func checkPrerequisites(requireGit bool) error {
	if requireGit {
		if err := ensureGit(); err != nil {
			return err
		}
	}
	return validateDockerCompose()
}

func ensureGit() error {
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
	if isDebian() {
		if err := installViaApt("curl", []string{"curl"}); err != nil {
			return err
		}
		if err := runShell("curl -fsSL https://deb.nodesource.com/setup_lts.x | bash -"); err != nil {
			return fmt.Errorf("failed to setup node.js repository: %w", err)
		}
		return installViaApt("nodejs", []string{"nodejs"})
	}
	if hasHomebrew() {
		return installViaBrew("node", "node", false)
	}
	return fmt.Errorf("automatic installation of nodejs not supported for this os")
}

func installGo() error {
	if isDebian() {
		return installViaApt("go", []string{"golang-go"})
	}
	if hasHomebrew() {
		return installViaBrew("go", "go", false)
	}
	return fmt.Errorf("automatic installation of go not supported for this os")
}

func installGit() error {
	if isDebian() {
		return installViaApt("git", []string{"git"})
	}
	if hasHomebrew() {
		return installViaBrew("git", "git", false)
	}
	return fmt.Errorf("automatic installation of git not supported for this os")
}
