package unit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kubeorchestra/cli/cmd"
	"github.com/kubeorchestra/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CommandTestSuite struct {
	suite.Suite
	origDir string
	tempDir string
	origPATH string
}

func (s *CommandTestSuite) SetupTest() {
	origDir, _ := os.Getwd()
	s.origDir = origDir
	s.origPATH = os.Getenv("PATH")

	s.tempDir = s.T().TempDir()
	os.Chdir(s.tempDir)

	cmd.ResetCommands()
}

func (s *CommandTestSuite) TearDownTest() {
	os.Chdir(s.origDir)
	os.Setenv("PATH", s.origPATH)
}

func (s *CommandTestSuite) TestVersionCommand() {
	rootCmd := cmd.GetRootCommand()
	output, err := cmd.ExecuteCommandC(rootCmd, "--version")

	assert.NoError(s.T(), err)
	assert.Contains(s.T(), output, "OrchCLI")
	assert.Contains(s.T(), output, "License: Apache-2.0")
}

func (s *CommandTestSuite) TestHelpCommand() {
	rootCmd := cmd.GetRootCommand()
	output, err := cmd.ExecuteCommandC(rootCmd, "--help")

	assert.NoError(s.T(), err)
	assert.Contains(s.T(), output, "OrchCLI is a developer tool")
	assert.Contains(s.T(), output, "Available Commands:")
	assert.Contains(s.T(), output, "init")
	assert.Contains(s.T(), output, "start")
	assert.Contains(s.T(), output, "stop")
}

func (s *CommandTestSuite) TestInitProductionMode() {
	if runtime.GOOS == "windows" {
		// Create mock docker.bat and docker-compose.bat
		helpers.CreateMockCommand(s.T(), s.tempDir,
			"docker", `echo Docker version 24.0.0`)
		helpers.CreateMockCommand(s.T(), s.tempDir,
			"docker-compose", `echo Docker Compose version v2.20.0`)
	} else {
		mockScript := `#!/bin/sh
echo "Docker Compose version v2.20.0"
`
		os.WriteFile(
			filepath.Join(s.tempDir, "docker-compose"),
			[]byte(mockScript), 0755,
		)
		// Also mock docker itself
		dockerMock := `#!/bin/sh
echo "Docker version 24.0.0"
`
		os.WriteFile(
			filepath.Join(s.tempDir, "docker"),
			[]byte(dockerMock), 0755,
		)
	}

	helpers.MockPATH(s.tempDir)

	rootCmd := cmd.GetRootCommand()
	output, err := cmd.ExecuteCommandC(rootCmd, "init", "--skip-deps")

	assert.NoError(s.T(), err)
	assert.Contains(s.T(), output, "Setting up OrchCLI for production testing")
	assert.Contains(s.T(), output, "Production environment ready")
	assert.Contains(s.T(), output, "Project initialized at:")

	assert.DirExists(s.T(), filepath.Join(s.tempDir, "docker"))
	assert.DirExists(s.T(), filepath.Join(s.tempDir, "scripts"))

	config, err := cmd.LoadConfig()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), config.Projects[s.tempDir])
	assert.Equal(s.T(), "production", config.Projects[s.tempDir].Mode)
}

func (s *CommandTestSuite) TestInitWithInvalidFork() {
	rootCmd := cmd.GetRootCommand()
	_, err := cmd.ExecuteCommandC(
		rootCmd, "init", "--fork-ui=invalid@repo#name",
	)

	assert.Error(s.T(), err)
}

func (s *CommandTestSuite) TestStartWithMissingComposeFile() {
	rootCmd := cmd.GetRootCommand()
	_, err := cmd.ExecuteCommandC(rootCmd, "start")

	assert.Error(s.T(), err)
	errMsg := err.Error()
	assert.True(s.T(),
		strings.Contains(errMsg, "no project initialized") ||
			strings.Contains(errMsg, "compose file") ||
			strings.Contains(errMsg, "docker"),
		"unexpected error: %s", errMsg)
}

func (s *CommandTestSuite) TestDebugCommand() {
	if runtime.GOOS == "windows" {
		helpers.CreateMockCommand(s.T(), s.tempDir, "docker",
			`echo NETWORK ID     NAME                DRIVER    SCOPE`)
	} else {
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
		os.WriteFile(
			filepath.Join(s.tempDir, "docker"),
			[]byte(mockScript), 0755,
		)
	}

	helpers.MockPATH(s.tempDir)

	rootCmd := cmd.GetRootCommand()
	output, err := cmd.ExecuteCommandC(rootCmd, "debug")

	assert.NoError(s.T(), err)
	assert.Contains(s.T(), output, "debugging service connectivity")
	assert.Contains(s.T(), output, "docker networks")
}

func (s *CommandTestSuite) TestStatusCommand() {
	config := &cmd.OrchConfig{
		CurrentProject: s.tempDir,
		Projects: map[string]*cmd.ProjectConfig{
			s.tempDir: {
				Path: s.tempDir,
				Mode: "production",
			},
		},
	}
	err := cmd.SaveConfig(config)
	assert.NoError(s.T(), err)

	os.MkdirAll(filepath.Join(s.tempDir, "docker"), 0755)
	composeContent := `version: '3.8'
services:
  postgres:
    image: postgres:14`
	os.WriteFile(
		filepath.Join(s.tempDir, "docker/docker-compose.prod.yml"),
		[]byte(composeContent), 0644,
	)

	if runtime.GOOS == "windows" {
		helpers.CreateMockCommand(s.T(), s.tempDir, "docker",
			`echo NAME                  STATUS    PORTS`)
	} else {
		mockScript := `#!/bin/sh
if [ "$1" = "compose" ]; then
	echo "Docker Compose version v2.20.0"
elif [ "$2" = "-f" ]; then
	echo "NAME                  STATUS    PORTS"
	echo "kubeorchestra-postgres  running   5432/tcp"
fi
`
		os.WriteFile(
			filepath.Join(s.tempDir, "docker"),
			[]byte(mockScript), 0755,
		)
	}

	helpers.MockPATH(s.tempDir)

	rootCmd := cmd.GetRootCommand()
	output, err := cmd.ExecuteCommandC(rootCmd, "status")

	assert.NoError(s.T(), err)
	assert.Contains(s.T(), output, "checking kubeorchestra services")
}

func (s *CommandTestSuite) TestExecWithInvalidService() {
	rootCmd := cmd.GetRootCommand()
	_, err := cmd.ExecuteCommandC(rootCmd, "exec", "invalid-service")

	assert.Error(s.T(), err)
}

func TestCommandTestSuite(t *testing.T) {
	suite.Run(t, new(CommandTestSuite))
}
