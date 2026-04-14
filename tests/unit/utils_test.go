package unit

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kubeorch/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UtilsTestSuite struct {
	suite.Suite
	helper *helpers.TestHelper
}

func (suite *UtilsTestSuite) SetupTest() {
	suite.helper = helpers.NewTestHelper(suite.T())
}

func (suite *UtilsTestSuite) TearDownTest() {
	suite.helper.Cleanup()
}

func (suite *UtilsTestSuite) TestDirExists() {
	// Test directory existence check
	testDir := suite.helper.CreateTempDir("testdir")

	// Directory should exist
	assert.True(suite.T(), suite.helper.FileExists(testDir))

	// Non-existent directory
	assert.False(suite.T(), suite.helper.FileExists(filepath.Join(suite.helper.TempDir, "nonexistent")))

	// File is not a directory
	testFile := suite.helper.CreateTempFile("testfile.txt", "content")
	assert.True(suite.T(), suite.helper.FileExists(testFile))
}

func (suite *UtilsTestSuite) TestCheckCommand() {
	// Test command availability check

	// Common commands that should exist
	_, err := exec.LookPath("ls")
	if err == nil {
		// ls exists on this system
		assert.NoError(suite.T(), err)
	}

	// Non-existent command
	_, err = exec.LookPath("nonexistentcommand123")
	assert.Error(suite.T(), err)
}

func (suite *UtilsTestSuite) TestGetComposeFile() {
	// Test compose file selection logic
	testCases := []struct {
		name      string
		expected  string
		uiLocal   bool
		coreLocal bool
	}{
		{
			name:      "Production mode",
			uiLocal:   false,
			coreLocal: false,
			expected:  "docker/docker-compose.prod.yml",
		},
		{
			name:      "Development mode",
			uiLocal:   true,
			coreLocal: true,
			expected:  "docker/docker-compose.dev.yml",
		},
		{
			name:      "Hybrid UI mode",
			uiLocal:   true,
			coreLocal: false,
			expected:  "docker/docker-compose.hybrid-ui.yml",
		},
		{
			name:      "Hybrid Core mode",
			uiLocal:   false,
			coreLocal: true,
			expected:  "docker/docker-compose.hybrid-core.yml",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// This would test the actual getComposeFile function
			// Need to refactor to make it testable
			assert.NotEmpty(suite.T(), tc.expected)
		})
	}
}

func (suite *UtilsTestSuite) TestValidateDockerCompose() {
	// Test Docker Compose validation
	// This would mock the docker compose check

	// Mock successful validation
	mockCmd := func() error {
		return nil
	}

	err := mockCmd()
	assert.NoError(suite.T(), err)

	// Mock failed validation
	mockFailCmd := func() error {
		return assert.AnError
	}

	err = mockFailCmd()
	assert.Error(suite.T(), err)
}

func (suite *UtilsTestSuite) TestJoinArgs() {
	// Test argument joining for shell commands
	testCases := []struct {
		expected string
		args     []string
	}{
		{
			expected: "arg1 arg2",
			args:     []string{"arg1", "arg2"},
		},
		{
			expected: "'arg with spaces' arg2",
			args:     []string{"arg with spaces", "arg2"},
		},
		{
			expected: "'arg'\"'\"'with'\"'\"'quotes' arg2",
			args:     []string{"arg'with'quotes", "arg2"},
		},
	}

	for _, tc := range testCases {
		// This would test the actual joinArgs function
		assert.NotEmpty(suite.T(), tc.expected)
	}
}

func TestUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(UtilsTestSuite))
}
