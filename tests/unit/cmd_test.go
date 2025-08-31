package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeorchestra/cli/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CommandTestSuite struct {
	suite.Suite
	origDir string
	tempDir string
}

func (suite *CommandTestSuite) SetupTest() {
	// Save original directory
	origDir, _ := os.Getwd()
	suite.origDir = origDir

	// Create and change to temp directory
	suite.tempDir = suite.T().TempDir()
	os.Chdir(suite.tempDir)

	// Reset commands for each test
	cmd.ResetCommands()
}

func (suite *CommandTestSuite) TearDownTest() {
	// Return to original directory
	os.Chdir(suite.origDir)
}

func (suite *CommandTestSuite) TestVersionCommand() {
	rootCmd := cmd.GetRootCommand()
	output, err := cmd.ExecuteCommandC(rootCmd, "--version")

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), output, "OrchCLI")
	assert.Contains(suite.T(), output, "License: Apache-2.0")
}

func (suite *CommandTestSuite) TestHelpCommand() {
	rootCmd := cmd.GetRootCommand()
	output, err := cmd.ExecuteCommandC(rootCmd, "--help")

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), output, "OrchCLI is a developer tool")
	assert.Contains(suite.T(), output, "Available Commands:")
	assert.Contains(suite.T(), output, "init")
	assert.Contains(suite.T(), output, "start")
	assert.Contains(suite.T(), output, "stop")
}

func (suite *CommandTestSuite) TestInitProductionMode() {
	// Create mock docker-compose command
	mockScript := `#!/bin/sh
echo "Docker Compose version v2.20.0"
`
	err := os.WriteFile(filepath.Join(suite.tempDir, "docker-compose"), []byte(mockScript), 0755)
	assert.NoError(suite.T(), err)

	// Add temp dir to PATH
	os.Setenv("PATH", suite.tempDir+":"+os.Getenv("PATH"))

	rootCmd := cmd.GetRootCommand()
	output, err := cmd.ExecuteCommandC(rootCmd, "init")

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), output, "Setting up OrchCLI for production testing")
	assert.Contains(suite.T(), output, "Production environment ready")
	assert.Contains(suite.T(), output, "Project initialized at:")

	// Check directories were created
	assert.DirExists(suite.T(), filepath.Join(suite.tempDir, "docker"))
	assert.DirExists(suite.T(), filepath.Join(suite.tempDir, "scripts"))

	// Check config was saved
	config, err := cmd.LoadConfig()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), config.Projects[suite.tempDir])
	assert.Equal(suite.T(), "production", config.Projects[suite.tempDir].Mode)
}

func (suite *CommandTestSuite) TestInitWithInvalidFork() {
	rootCmd := cmd.GetRootCommand()
	_, err := cmd.ExecuteCommandC(rootCmd, "init", "--fork-ui=invalid@repo#name")

	assert.Error(suite.T(), err)
}

func (suite *CommandTestSuite) TestStartWithMissingComposeFile() {
	rootCmd := cmd.GetRootCommand()
	_, err := cmd.ExecuteCommandC(rootCmd, "start")

	assert.Error(suite.T(), err)
	// The error could be either no project initialized or missing compose file
	assert.True(suite.T(),
		err.Error() == "no project initialized in current directory. Run 'orchcli init' first" ||
			strings.Contains(err.Error(), "compose file"))
}

func (suite *CommandTestSuite) TestDebugCommand() {
	// Create mock docker command
	mockScript := `#!/bin/sh
case "$1" in
	network)
		echo "NETWORK ID     NAME                DRIVER    SCOPE"
		echo "abc123         kubeorchestra-net   bridge    local"
		;;
	ps)
		if [ "$2" = "--format" ]; then
			echo "NAMES                 STATUS       PORTS"
			echo "kubeorchestra-core    Up 5 min     3000/tcp"
		fi
		;;
	exec)
		echo "postgres:5432 - accepting connections"
		;;
esac
`
	err := os.WriteFile(filepath.Join(suite.tempDir, "docker"), []byte(mockScript), 0755)
	assert.NoError(suite.T(), err)
	os.Setenv("PATH", suite.tempDir+":"+os.Getenv("PATH"))

	rootCmd := cmd.GetRootCommand()
	output, err := cmd.ExecuteCommandC(rootCmd, "debug")

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), output, "debugging service connectivity")
	assert.Contains(suite.T(), output, "docker networks")
}

func (suite *CommandTestSuite) TestStatusCommand() {
	// Initialize project first
	config := &cmd.OrchConfig{
		CurrentProject: suite.tempDir,
		Projects: map[string]*cmd.ProjectConfig{
			suite.tempDir: {
				Path: suite.tempDir,
				Mode: "production",
			},
		},
	}
	err := cmd.SaveConfig(config)
	assert.NoError(suite.T(), err)

	// Create compose file
	os.MkdirAll(filepath.Join(suite.tempDir, "docker"), 0755)
	composeContent := `version: '3.8'
services:
  postgres:
    image: postgres:14`
	err = os.WriteFile(filepath.Join(suite.tempDir, "docker/docker-compose.prod.yml"), []byte(composeContent), 0644)
	assert.NoError(suite.T(), err)

	// Create mock docker binary (not directory)
	mockScript := `#!/bin/sh
if [ "$1" = "compose" ]; then
	echo "Docker Compose version v2.20.0"
elif [ "$2" = "-f" ]; then
	echo "NAME                  STATUS    PORTS"
	echo "kubeorchestra-postgres  running   5432/tcp"
fi
`
	err = os.WriteFile(filepath.Join(suite.tempDir, "docker-mock"), []byte(mockScript), 0755)
	assert.NoError(suite.T(), err)

	// Create symlink or copy to docker
	err = os.Symlink(filepath.Join(suite.tempDir, "docker-mock"), filepath.Join(suite.tempDir, "docker"))
	if err != nil {
		// If symlink fails, just skip this test
		suite.T().Skip("Cannot create docker mock")
	}

	os.Setenv("PATH", suite.tempDir+":"+os.Getenv("PATH"))

	rootCmd := cmd.GetRootCommand()
	output, err := cmd.ExecuteCommandC(rootCmd, "status")

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), output, "checking kubeorchestra services")
}

func (suite *CommandTestSuite) TestExecWithInvalidService() {
	rootCmd := cmd.GetRootCommand()
	_, err := cmd.ExecuteCommandC(rootCmd, "exec", "invalid-service")

	assert.Error(suite.T(), err)
}

func TestCommandTestSuite(t *testing.T) {
	suite.Run(t, new(CommandTestSuite))
}
