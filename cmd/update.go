package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	force    bool
	uiOnly   bool
	coreOnly bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update KubeOrchestra repositories",
	Long: `Update UI and Core repositories with latest changes.

Updates repositories based on your setup:
- Official repos: pulls from origin
- Fork repos: fetches from upstream and merges
- Handles merge conflicts gracefully
- Supports selective updates with --ui-only or --core-only

Examples:
  # Update both repositories
  orchcli update
  
  # Update only UI repository
  orchcli update --ui-only
  
  # Force reset both repositories to remote
  orchcli update --force
  
  # Update only Core repository with force reset
  orchcli update --core-only --force`,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().BoolVarP(&force, "force", "f", false, "force reset to remote (discards local changes)")
	updateCmd.Flags().BoolVar(&uiOnly, "ui-only", false, "update only UI repository")
	updateCmd.Flags().BoolVar(&coreOnly, "core-only", false, "update only Core repository")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	projectConfig, err := getCurrentProjectConfig()
	if err != nil {
		return fmt.Errorf("no project initialized in current directory. Run 'orchcli init' first")
	}

	uiLocal := projectConfig.UIPath != "" && dirExists(projectConfig.UIPath)
	coreLocal := projectConfig.CorePath != "" && dirExists(projectConfig.CorePath)

	if !uiLocal && !coreLocal {
		return fmt.Errorf("no repositories found. Run 'orchcli init' to clone repositories first")
	}

	if uiOnly && coreOnly {
		return fmt.Errorf("cannot use both --ui-only and --core-only flags")
	}

	var updateTasks []Task
	var summaryTasks []Task

	if uiLocal && (!coreOnly || uiOnly) {
		updateTasks = append(updateTasks, Task{
			Action: func() error {
				return updateRepository(projectConfig.UIPath, "UI", force)
			},
			Progress: NewProgressBar("Updating UI repository"),
			Name:     "Update UI repository",
		})
		summaryTasks = append(summaryTasks, Task{
			Action: func() error {
				return showRepositorySummary(projectConfig.UIPath, "UI")
			},
			Name: "Show UI repository summary",
		})
	}

	if coreLocal && (!uiOnly || coreOnly) {
		updateTasks = append(updateTasks, Task{
			Action: func() error {
				return updateRepository(projectConfig.CorePath, "Core", force)
			},
			Progress: NewProgressBar("Updating Core repository"),
			Name:     "Update Core repository",
		})
		summaryTasks = append(summaryTasks, Task{
			Action: func() error {
				return showRepositorySummary(projectConfig.CorePath, "Core")
			},
			Name: "Show Core repository summary",
		})
	}

	if len(updateTasks) == 0 {
		return fmt.Errorf("no repositories to update")
	}

	fmt.Println("🔄 updating repositories...")

	results := RunConcurrent(updateTasks)
	if err := AggregateErrors(results); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println("✅ repositories updated successfully")

	if len(summaryTasks) > 0 {
		fmt.Println("\n📊 update summary:")
		RunConcurrent(summaryTasks)
	}

	return nil
}

func updateRepository(repoPath, repoName string, force bool) error {
	if !dirExists(repoPath) {
		return fmt.Errorf("%s repository not found at %s", repoName, repoPath)
	}

	if !isGitRepository(repoPath) {
		return fmt.Errorf("%s directory at %s is not a git repository", repoName, repoPath)
	}

	if !force {
		if err := checkRepositoryStatus(repoPath, repoName); err != nil {
			return err
		}
	}

	isFork, err := isForkRepository(repoPath)
	if err != nil {
		return fmt.Errorf("failed to check %s repository type: %w", repoName, err)
	}

	if force {
		return forceResetRepository(repoPath, repoName, isFork)
	}

	return pullRepository(repoPath, repoName, isFork)
}

func isGitRepository(repoPath string) bool {
	gitDir := filepath.Join(repoPath, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

func checkRepositoryStatus(repoPath, repoName string) error {
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = repoPath
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check %s repository status: %w", repoName, err)
	}

	if len(output) > 0 {
		return fmt.Errorf("%s repository has uncommitted changes. Please commit or stash them first, or use --force to discard changes", repoName)
	}

	return nil
}

func isForkRepository(repoPath string) (bool, error) {
	upstreamCmd := exec.Command("git", "remote", "get-url", "upstream")
	upstreamCmd.Dir = repoPath
	upstreamOutput, err := upstreamCmd.Output()
	if err != nil {
		return false, nil
	}

	originCmd := exec.Command("git", "remote", "get-url", "origin")
	originCmd.Dir = repoPath
	originOutput, err := originCmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get origin remote: %w", err)
	}

	originURL := strings.TrimSpace(string(originOutput))
	upstreamURL := strings.TrimSpace(string(upstreamOutput))

	return originURL != upstreamURL, nil
}

func pullRepository(repoPath, repoName string, isFork bool) error {
	if err := validateRemoteConnectivity(repoPath, repoName, isFork); err != nil {
		return err
	}

	if isFork {
		return pullForkRepository(repoPath, repoName)
	}
	return pullOfficialRepository(repoPath, repoName)
}

func validateRemoteConnectivity(repoPath, repoName string, isFork bool) error {
	var remote string
	if isFork {
		remote = "upstream"
	} else {
		remote = "origin"
	}

	cmd := exec.Command("git", "ls-remote", remote)
	cmd.Dir = repoPath
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot connect to %s remote for %s repository. Check network connection and remote URL", remote, repoName)
	}

	return nil
}

func pullOfficialRepository(repoPath, repoName string) error {
	currentBranch, err := getCurrentBranch(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get current branch for %s repository: %w", repoName, err)
	}

	cmd := exec.Command("git", "pull", "origin", currentBranch)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull %s repository: %w", repoName, err)
	}

	return nil
}

func pullForkRepository(repoPath, repoName string) error {
	fetchCmd := exec.Command("git", "fetch", "upstream")
	fetchCmd.Dir = repoPath
	fetchCmd.Stdout = os.Stdout
	fetchCmd.Stderr = os.Stderr

	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch upstream for %s repository: %w", repoName, err)
	}

	defaultBranch, err := getDefaultBranch(repoPath, "upstream")
	if err != nil {
		defaultBranch = "main"
	}

	mergeCmd := exec.Command("git", "merge", fmt.Sprintf("upstream/%s", defaultBranch))
	mergeCmd.Dir = repoPath
	mergeCmd.Stdout = os.Stdout
	mergeCmd.Stderr = os.Stderr

	if err := mergeCmd.Run(); err != nil {
		return handleMergeConflict(repoPath, repoName, err)
	}

	return nil
}

func handleMergeConflict(repoPath, repoName string, mergeErr error) error {
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = repoPath
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("merge conflict in %s repository. Please resolve manually: %w", repoName, mergeErr)
	}

	if len(output) > 0 {
		return fmt.Errorf("merge conflict in %s repository. Files with conflicts:\n%s\n\nTo resolve:\n1. Edit conflicted files\n2. git add <files>\n3. git commit\n\nOr abort merge: git merge --abort", repoName, string(output))
	}

	return fmt.Errorf("merge failed in %s repository: %w", repoName, mergeErr)
}

func forceResetRepository(repoPath, repoName string, isFork bool) error {
	var resetCmd *exec.Cmd
	if isFork {
		defaultBranch, err := getDefaultBranch(repoPath, "upstream")
		if err != nil {
			defaultBranch = "main"
		}
		resetCmd = exec.Command("git", "reset", "--hard", fmt.Sprintf("upstream/%s", defaultBranch))
	} else {
		currentBranch, err := getCurrentBranch(repoPath)
		if err != nil {
			return fmt.Errorf("failed to get current branch for %s repository: %w", repoName, err)
		}
		resetCmd = exec.Command("git", "reset", "--hard", fmt.Sprintf("origin/%s", currentBranch))
	}

	resetCmd.Dir = repoPath
	resetCmd.Stdout = os.Stdout
	resetCmd.Stderr = os.Stderr

	if err := resetCmd.Run(); err != nil {
		return fmt.Errorf("failed to reset %s repository: %w", repoName, err)
	}

	return nil
}

func getCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getDefaultBranch(repoPath, remote string) (string, error) {
	cmd := exec.Command("git", "remote", "show", remote)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "main", nil
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "HEAD branch:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return strings.TrimSpace(parts[2]), nil
			}
		}
	}

	return "main", nil
}

func showRepositorySummary(repoPath, repoName string) error {
	if !dirExists(repoPath) || !isGitRepository(repoPath) {
		return nil
	}

	fmt.Printf("\n  📁 %s repository:\n", repoName)

	branch, err := getCurrentBranch(repoPath)
	if err == nil {
		fmt.Printf("    branch: %s\n", branch)
	}

	status, err := getRepositoryStatus(repoPath)
	if err == nil && status != "" {
		fmt.Printf("    status: %s\n", status)
	}

	recentCommits, err := getRecentCommits(repoPath)
	if err == nil && len(recentCommits) > 0 {
		fmt.Printf("    recent commits:\n")
		for _, commit := range recentCommits {
			fmt.Printf("      %s\n", commit)
		}
	}

	return nil
}

func getRepositoryStatus(repoPath string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return "clean", nil
	}

	modified := 0
	untracked := 0
	for _, line := range lines {
		if line != "" {
			if strings.HasPrefix(line, "??") {
				untracked++
			} else {
				modified++
			}
		}
	}

	if modified > 0 && untracked > 0 {
		return fmt.Sprintf("%d modified, %d untracked files", modified, untracked), nil
	} else if modified > 0 {
		return fmt.Sprintf("%d modified files", modified), nil
	} else if untracked > 0 {
		return fmt.Sprintf("%d untracked files", untracked), nil
	}

	return "clean", nil
}

func getRecentCommits(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "log", "--oneline", "-5")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var commits []string
	for _, line := range lines {
		if line != "" {
			commits = append(commits, line)
		}
	}

	return commits, nil
}
