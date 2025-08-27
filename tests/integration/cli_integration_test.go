//go:build integration
// +build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kubeorchestra/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CLIIntegrationTestSuite struct {
	suite.Suite
	helper     *helpers.TestHelper
	binaryPath string
}

func (suite *CLIIntegrationTestSuite) SetupSuite() {
	suite.helper = helpers.NewTestHelper(suite.T())

	// Build the CLI binary for testing
	suite.binaryPath = filepath.Join(suite.helper.TempDir, "orchcli")
	cmd := exec.Command("go", "build", "-o", suite.binaryPath, "../../main.go")
	err := cmd.Run()
	assert.NoError(suite.T(), err, "Failed to build CLI binary")
}

func (suite *CLIIntegrationTestSuite) TearDownSuite() {
	suite.helper.Cleanup()
}

func (suite *CLIIntegrationTestSuite) TestCLIVersion() {
	cmd := exec.Command(suite.binaryPath, "--version")
	output, err := cmd.CombinedOutput()

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(output), "OrchCLI")
	assert.Contains(suite.T(), string(output), "License: Apache-2.0")
}

func (suite *CLIIntegrationTestSuite) TestCLIHelp() {
	cmd := exec.Command(suite.binaryPath, "--help")
	output, err := cmd.CombinedOutput()

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(output), "OrchCLI is a developer tool")
	assert.Contains(suite.T(), string(output), "Available Commands:")
	assert.Contains(suite.T(), string(output), "init")
	assert.Contains(suite.T(), string(output), "start")
	assert.Contains(suite.T(), string(output), "stop")
	assert.Contains(suite.T(), string(output), "logs")
	assert.Contains(suite.T(), string(output), "status")
	assert.Contains(suite.T(), string(output), "restart")
	assert.Contains(suite.T(), string(output), "debug")
	assert.Contains(suite.T(), string(output), "exec")
}

func (suite *CLIIntegrationTestSuite) TestInitProductionMode() {
	// Change to temp directory
	os.Chdir(suite.helper.TempDir)

	cmd := exec.Command(suite.binaryPath, "init")
	output, err := cmd.CombinedOutput()

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(output), "Setting up OrchCLI for production testing")
	assert.Contains(suite.T(), string(output), "Production environment ready")

	// Check that docker directory was created
	assert.True(suite.T(), suite.helper.FileExists(filepath.Join(suite.helper.TempDir, "docker")))
	assert.True(suite.T(), suite.helper.FileExists(filepath.Join(suite.helper.TempDir, "scripts")))
}

func (suite *CLIIntegrationTestSuite) TestInitWithInvalidFork() {
	os.Chdir(suite.helper.TempDir)

	cmd := exec.Command(suite.binaryPath, "init", "--fork-ui=invalid@repo#name")
	output, err := cmd.CombinedOutput()

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), string(output), "invalid repository format")
}

func (suite *CLIIntegrationTestSuite) TestStartWithoutInit() {
	os.Chdir(suite.helper.TempDir)

	cmd := exec.Command(suite.binaryPath, "start")
	output, err := cmd.CombinedOutput()

	// Should fail because docker-compose files don't exist or are empty
	assert.Error(suite.T(), err)
	// Check for either error message
	errorFound := strings.Contains(string(output), "not found") || 
	              strings.Contains(string(output), "empty compose file") ||
	              strings.Contains(string(output), "failed to start")
	assert.True(suite.T(), errorFound, "Expected error message not found in output")
}

func (suite *CLIIntegrationTestSuite) TestStatusCommand() {
	os.Chdir(suite.helper.TempDir)
	
	// Create docker directory and compose file so status can run
	suite.helper.CreateTempDir("docker")
	suite.helper.CreateTempFile("docker/docker-compose.prod.yml", "version: '3.8'\nservices:\n  postgres:\n    image: postgres:14")

	cmd := exec.Command(suite.binaryPath, "status")
	output, _ := cmd.CombinedOutput()

	// Status command itself should not error even without running services
	// It might fail to connect but the command should execute
	assert.NotNil(suite.T(), output)
	assert.Contains(suite.T(), string(output), "checking kubeorchestra services")
}

func (suite *CLIIntegrationTestSuite) TestDebugCommand() {
	os.Chdir(suite.helper.TempDir)

	cmd := exec.Command(suite.binaryPath, "debug")
	output, err := cmd.CombinedOutput()

	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), string(output), "debugging service connectivity")
}

func (suite *CLIIntegrationTestSuite) TestExecWithoutRunningServices() {
	os.Chdir(suite.helper.TempDir)

	cmd := exec.Command(suite.binaryPath, "exec", "postgres", "psql")
	output, err := cmd.CombinedOutput()

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), string(output), "is not running")
}

func (suite *CLIIntegrationTestSuite) TestLogsWithoutRunningServices() {
	os.Chdir(suite.helper.TempDir)

	// Create docker directory and compose file to avoid "not found" error
	suite.helper.CreateTempDir("docker")
	suite.helper.CreateTempFile("docker/docker-compose.prod.yml", "version: '3.8'")

	cmd := exec.Command(suite.binaryPath, "logs")
	_, err := cmd.CombinedOutput()

	// Should fail gracefully when no services are running
	// The actual error will depend on docker-compose behavior
	assert.Error(suite.T(), err)
}

func (suite *CLIIntegrationTestSuite) TestCommandTimeout() {
	// Test that commands respect timeouts
	os.Chdir(suite.helper.TempDir)

	// Create a test that would timeout
	done := make(chan bool)
	go func() {
		cmd := exec.Command(suite.binaryPath, "--version")
		cmd.Run()
		done <- true
	}()

	select {
	case <-done:
		// Command completed
		assert.True(suite.T(), true)
	case <-time.After(5 * time.Second):
		// Command timed out
		assert.Fail(suite.T(), "Command timed out")
	}
}

func TestCLIIntegrationTestSuite(t *testing.T) {
	// Skip integration tests in CI unless explicitly enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests. Set RUN_INTEGRATION_TESTS=true to run")
	}

	suite.Run(t, new(CLIIntegrationTestSuite))
}
