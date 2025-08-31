package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ProjectConfig struct {
	Path     string `json:"path"`
	UIPath   string `json:"ui_path,omitempty"`
	CorePath string `json:"core_path,omitempty"`
	Mode     string `json:"mode"` // "production", "development", "ui-dev", "core-dev"
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

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func getCurrentProjectConfig() (*ProjectConfig, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	if project, exists := config.Projects[cwd]; exists {
		return project, nil
	}

	if config.CurrentProject != "" {
		if project, exists := config.Projects[config.CurrentProject]; exists {
			return project, nil
		}
	}

	return nil, fmt.Errorf("no project configured for current directory. Run 'orchcli init' first")
}

func setProjectConfig(projectPath string, uiPath, corePath string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	var mode string
	switch {
	case uiPath != "" && corePath != "":
		mode = "development"
	case uiPath != "":
		mode = "ui-dev"
	case corePath != "":
		mode = "core-dev"
	default:
		mode = "production"
	}

	config.Projects[projectPath] = &ProjectConfig{
		Path:     projectPath,
		UIPath:   uiPath,
		CorePath: corePath,
		Mode:     mode,
	}
	config.CurrentProject = projectPath

	return SaveConfig(config)
}


// removeProjectConfig removes a project from the configuration (unused but kept for future use)
func removeProjectConfig(projectPath string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	delete(config.Projects, projectPath)

	if config.CurrentProject == projectPath {
		config.CurrentProject = ""
	}

	return SaveConfig(config)
}

