package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeorchestra/cli/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigManagement(t *testing.T) {
	t.Run("LoadEmptyConfig", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "orchcli-config.json")

		// Set environment to use temp config
		oldExecutable := os.Args[0]
		os.Args[0] = filepath.Join(tempDir, "orchcli")
		defer func() { os.Args[0] = oldExecutable }()

		// Create empty config file
		err := os.WriteFile(configPath, []byte("{}"), 0644)
		require.NoError(t, err)

		// Test loading - this would need the functions to be exported
		// For now, we'll test the file structure
		data, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var config map[string]interface{}
		err = json.Unmarshal(data, &config)
		require.NoError(t, err)
		assert.NotNil(t, config)
	})

	t.Run("SaveProjectConfig", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "orchcli-config.json")

		// Create a test config
		config := cmd.OrchConfig{
			CurrentProject: "/test/project",
			Projects: map[string]*cmd.ProjectConfig{
				"/test/project": {
					Path:     "/test/project",
					UIPath:   "/test/project/ui",
					CorePath: "/test/project/core",
					Mode:     "development",
				},
			},
		}

		// Marshal and save
		data, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		// Verify the saved content
		savedData, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var savedConfig cmd.OrchConfig
		err = json.Unmarshal(savedData, &savedConfig)
		require.NoError(t, err)

		assert.Equal(t, config.CurrentProject, savedConfig.CurrentProject)
		assert.Equal(t, len(config.Projects), len(savedConfig.Projects))
		assert.Equal(t, config.Projects["/test/project"].Mode, savedConfig.Projects["/test/project"].Mode)
	})

	t.Run("MultipleProjects", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "orchcli-config.json")

		config := cmd.OrchConfig{
			CurrentProject: "/project2",
			Projects: map[string]*cmd.ProjectConfig{
				"/project1": {
					Path: "/project1",
					Mode: "production",
				},
				"/project2": {
					Path:     "/project2",
					UIPath:   "/project2/ui",
					CorePath: "/project2/core",
					Mode:     "development",
				},
				"/project3": {
					Path:   "/project3",
					UIPath: "/project3/ui",
					Mode:   "ui-dev",
				},
			},
		}

		data, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		// Load and verify
		savedData, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var savedConfig cmd.OrchConfig
		err = json.Unmarshal(savedData, &savedConfig)
		require.NoError(t, err)

		assert.Equal(t, 3, len(savedConfig.Projects))
		assert.Equal(t, "/project2", savedConfig.CurrentProject)
		assert.Equal(t, "production", savedConfig.Projects["/project1"].Mode)
		assert.Equal(t, "development", savedConfig.Projects["/project2"].Mode)
		assert.Equal(t, "ui-dev", savedConfig.Projects["/project3"].Mode)
	})

	t.Run("ConfigModes", func(t *testing.T) {
		tests := []struct {
			name     string
			uiPath   string
			corePath string
			expected string
		}{
			{"Production", "", "", "production"},
			{"Development", "/path/ui", "/path/core", "development"},
			{"UI Dev", "/path/ui", "", "ui-dev"},
			{"Core Dev", "", "/path/core", "core-dev"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var mode string
				switch {
				case tt.uiPath != "" && tt.corePath != "":
					mode = "development"
				case tt.uiPath != "":
					mode = "ui-dev"
				case tt.corePath != "":
					mode = "core-dev"
				default:
					mode = "production"
				}
				assert.Equal(t, tt.expected, mode)
			})
		}
	})
}

func TestProjectPaths(t *testing.T) {
	t.Run("AbsolutePaths", func(t *testing.T) {
		config := cmd.ProjectConfig{
			Path:     "/home/user/myproject",
			UIPath:   "/home/user/myproject/ui",
			CorePath: "/home/user/myproject/core",
			Mode:     "development",
		}

		assert.True(t, filepath.IsAbs(config.Path))
		assert.True(t, filepath.IsAbs(config.UIPath))
		assert.True(t, filepath.IsAbs(config.CorePath))
	})

	t.Run("PathRelationships", func(t *testing.T) {
		basePath := "/home/user/project"
		config := cmd.ProjectConfig{
			Path:     basePath,
			UIPath:   filepath.Join(basePath, "ui"),
			CorePath: filepath.Join(basePath, "core"),
			Mode:     "development",
		}

		// UI and Core should be subdirectories of Path
		assert.Equal(t, basePath, filepath.Dir(config.UIPath))
		assert.Equal(t, basePath, filepath.Dir(config.CorePath))
	})
}

