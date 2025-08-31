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
	CurrentProject string                    `json:"current_project,omitempty"`
	Projects       map[string]*ProjectConfig `json:"projects"`
}

func getConfigDir() (string, error) {
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

func getConfigPath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "orchcli-config.json"), nil
}

func loadConfig() (*OrchConfig, error) {
	configPath, err := getConfigPath()
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

func saveConfig(config *OrchConfig) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func getCurrentProjectConfig() (*ProjectConfig, error) {
	config, err := loadConfig()
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
	config, err := loadConfig()
	if err != nil {
		return err
	}

	mode := "production"
	if uiPath != "" && corePath != "" {
		mode = "development"
	} else if uiPath != "" {
		mode = "ui-dev"
	} else if corePath != "" {
		mode = "core-dev"
	}

	config.Projects[projectPath] = &ProjectConfig{
		Path:     projectPath,
		UIPath:   uiPath,
		CorePath: corePath,
		Mode:     mode,
	}
	config.CurrentProject = projectPath

	return saveConfig(config)
}

func removeProjectConfig(projectPath string) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	delete(config.Projects, projectPath)
	
	if config.CurrentProject == projectPath {
		config.CurrentProject = ""
	}

	return saveConfig(config)
}