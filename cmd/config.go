package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofrs/flock"
)

const (
	projectMarkerDir      = ".kubeorch"
	projectMarkerFilename = "project.json"
	projectMarkerVersion  = 1
)

var errProjectMarkerNotFound = errors.New("project marker not found")

type projectMarker struct {
	UIPath   string `json:"ui_path,omitempty"`
	CorePath string `json:"core_path,omitempty"`
	Mode     string `json:"mode"`
	Version  int    `json:"version"`
}

type ProjectConfig struct {
	Path     string `json:"path"`
	UIPath   string `json:"ui_path,omitempty"`
	CorePath string `json:"core_path,omitempty"`
	Mode     string `json:"mode"`
}

type OrchConfig struct {
	Projects       map[string]*ProjectConfig `json:"projects"`
	CurrentProject string                    `json:"current_project,omitempty"`
}

func GetConfigDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get config directory: %w", err)
		}
		return filepath.Join(homeDir, ".orchcli"), nil
	}
	return filepath.Dir(execPath), nil
}

func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "orchcli-config.json"), nil
}

func LoadConfig() (*OrchConfig, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &OrchConfig{
				Projects: make(map[string]*ProjectConfig),
			}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config OrchConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if config.Projects == nil {
		config.Projects = make(map[string]*ProjectConfig)
	}

	return &config, nil
}

func SaveConfig(config *OrchConfig) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	const dirMode = 0750
	configDir := filepath.Dir(configPath)
	if mkErr := os.MkdirAll(configDir, dirMode); mkErr != nil {
		return fmt.Errorf("failed to create config directory: %w", mkErr)
	}

	// Marshal config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Use file locking for concurrent access
	lockPath := configPath + ".lock"
	fileLock := flock.New(lockPath)

	// Try to acquire lock
	err = fileLock.Lock()
	if err != nil {
		return fmt.Errorf("failed to acquire config lock: %w", err)
	}
	defer func() {
		_ = fileLock.Unlock()
	}()

	if err := writeFileAtomically(configPath, data, configFilePerm); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func getCurrentProjectConfig() (*ProjectConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	project, err := loadProjectMarker(cwd)
	if err == nil {
		return project, nil
	}
	if !errors.Is(err, errProjectMarkerNotFound) {
		return nil, err
	}

	// Older OrchCLI releases only wrote the global registry. Support those
	// projects when the current directory is actually inside a registered root.
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	var closest *ProjectConfig
	for _, registered := range config.Projects {
		if registered == nil || !pathContains(registered.Path, cwd) {
			continue
		}
		if closest == nil || len(registered.Path) > len(closest.Path) {
			closest = registered
		}
	}
	if closest != nil {
		if err := validateProjectSources(closest); err != nil {
			return nil, err
		}
		return closest, nil
	}

	return nil, fmt.Errorf(
		"%w: no %s found in %s or its parents; run 'orchcli init' from the project root",
		errProjectMarkerNotFound,
		projectMarkerFilename,
		cwd,
	)
}

func setProjectConfig(projectPath string, uiPath, corePath string) error {
	projectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve project path: %w", err)
	}
	projectPath = filepath.Clean(projectPath)

	uiPath, err = absoluteOptionalPath(uiPath)
	if err != nil {
		return fmt.Errorf("failed to resolve UI path: %w", err)
	}
	corePath, err = absoluteOptionalPath(corePath)
	if err != nil {
		return fmt.Errorf("failed to resolve Core path: %w", err)
	}

	mode := projectMode(uiPath, corePath)
	project := &ProjectConfig{
		Path:     projectPath,
		UIPath:   uiPath,
		CorePath: corePath,
		Mode:     mode,
	}
	return writeProjectMarker(project)
}

func projectMode(uiPath, corePath string) string {
	switch {
	case uiPath != "" && corePath != "":
		return "development"
	case uiPath != "":
		return "ui-dev"
	case corePath != "":
		return "core-dev"
	default:
		return "production"
	}
}

func absoluteOptionalPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

func writeProjectMarker(project *ProjectConfig) error {
	markerDir := filepath.Join(project.Path, projectMarkerDir)
	if err := os.MkdirAll(markerDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create project marker directory: %w", err)
	}

	marker := projectMarker{
		Version:  projectMarkerVersion,
		UIPath:   portableProjectPath(project.Path, project.UIPath),
		CorePath: portableProjectPath(project.Path, project.CorePath),
		Mode:     project.Mode,
	}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project marker: %w", err)
	}
	data = append(data, '\n')

	markerPath := filepath.Join(markerDir, projectMarkerFilename)
	if err := writeFileAtomically(markerPath, data, configFilePerm); err != nil {
		return fmt.Errorf("failed to write project marker %s: %w", markerPath, err)
	}
	return nil
}

func writeFileAtomically(targetPath string, data []byte, perm os.FileMode) error {
	return writeFileAtomicallyWithHook(targetPath, data, perm, nil)
}

func writeFileAtomicallyWithHook(
	targetPath string,
	data []byte,
	perm os.FileMode,
	beforeRename func(string) error,
) error {
	tempFile, err := os.CreateTemp(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if err := tempFile.Chmod(perm); err != nil {
		return fmt.Errorf("failed to set temporary file permissions: %w", err)
	}
	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	if beforeRename != nil {
		if err := beforeRename(tempPath); err != nil {
			return fmt.Errorf("failed before replacing target file: %w", err)
		}
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		return fmt.Errorf("failed to replace target file: %w", err)
	}
	return nil
}

func loadProjectMarker(startPath string) (*ProjectConfig, error) {
	current, err := filepath.Abs(startPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve current directory: %w", err)
	}
	current = filepath.Clean(current)

	for {
		markerPath := filepath.Join(current, projectMarkerDir, projectMarkerFilename)
		data, readErr := os.ReadFile(markerPath)
		if readErr == nil {
			return parseProjectMarker(current, markerPath, data)
		}
		if !os.IsNotExist(readErr) {
			return nil, fmt.Errorf("failed to read project marker %s: %w", markerPath, readErr)
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return nil, errProjectMarkerNotFound
}

func parseProjectMarker(projectPath, markerPath string, data []byte) (*ProjectConfig, error) {
	var marker projectMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return nil, fmt.Errorf("invalid project marker %s: %w; run 'orchcli init' from %s to repair it", markerPath, err, projectPath)
	}
	if marker.Version != projectMarkerVersion {
		return nil, fmt.Errorf(
			"unsupported project marker version %d in %s (expected %d); update OrchCLI or run 'orchcli init' from %s",
			marker.Version,
			markerPath,
			projectMarkerVersion,
			projectPath,
		)
	}

	uiPath := resolveProjectPath(projectPath, marker.UIPath)
	corePath := resolveProjectPath(projectPath, marker.CorePath)
	expectedMode := projectMode(uiPath, corePath)
	if marker.Mode != expectedMode {
		return nil, fmt.Errorf(
			"invalid project marker %s: mode %q does not match configured source paths (expected %q); "+
				"run 'orchcli init' from %s to repair it",
			markerPath,
			marker.Mode,
			expectedMode,
			projectPath,
		)
	}

	project := &ProjectConfig{
		Path:     projectPath,
		UIPath:   uiPath,
		CorePath: corePath,
		Mode:     marker.Mode,
	}
	if err := validateProjectSources(project); err != nil {
		return nil, fmt.Errorf("invalid project marker %s: %w", markerPath, err)
	}
	return project, nil
}

func validateProjectSources(project *ProjectConfig) error {
	if project.UIPath != "" && !dirExists(project.UIPath) {
		return fmt.Errorf(
			"configured UI checkout is missing: %s; run 'orchcli init --ui-path <path>' from %s to repair it",
			project.UIPath,
			project.Path,
		)
	}
	if project.CorePath != "" && !dirExists(project.CorePath) {
		return fmt.Errorf(
			"configured Core checkout is missing: %s; run 'orchcli init --core-path <path>' from %s to repair it",
			project.CorePath,
			project.Path,
		)
	}
	return nil
}

func portableProjectPath(projectPath, sourcePath string) string {
	if sourcePath == "" {
		return ""
	}
	rel, err := filepath.Rel(projectPath, sourcePath)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(rel)
	}
	return filepath.Clean(sourcePath)
}

func resolveProjectPath(projectPath, sourcePath string) string {
	if sourcePath == "" {
		return ""
	}
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(projectPath, filepath.FromSlash(sourcePath))
	}
	return filepath.Clean(sourcePath)
}

func pathContains(root, candidate string) bool {
	rootAbs, rootErr := filepath.Abs(root)
	candidateAbs, candidateErr := filepath.Abs(candidate)
	if rootErr != nil || candidateErr != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, candidateAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// removeProjectConfig removes a project from the configuration
// Keeping for future use when we add a 'remove' or 'clean' command
//
//nolint:unused // kept for future 'orchcli remove' command implementation
func removeProjectConfig(projectPath string) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Use file locking for concurrent access
	lockPath := configPath + ".lock"
	fileLock := flock.New(lockPath)

	// Try to acquire lock
	err = fileLock.Lock()
	if err != nil {
		return fmt.Errorf("failed to acquire config lock: %w", err)
	}
	defer func() {
		_ = fileLock.Unlock()
	}()

	// Load current config
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Remove project
	delete(config.Projects, projectPath)

	if config.CurrentProject == projectPath {
		config.CurrentProject = ""
	}

	// Marshal and save
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := writeFileAtomically(configPath, data, configFilePerm); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
