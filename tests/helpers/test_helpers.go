// Package helpers provides test utility functions for the OrchCLI test suite.
package helpers

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHelper provides common test utilities
type TestHelper struct {
	T       *testing.T
	TempDir string
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	tempDir := t.TempDir()
	return &TestHelper{
		T:       t,
		TempDir: tempDir,
	}
}

// CreateTempFile creates a temporary file with content
func (h *TestHelper) CreateTempFile(name, content string) string {
	path := filepath.Join(h.TempDir, name)
	dir := filepath.Dir(path)

	err := os.MkdirAll(dir, 0755)
	require.NoError(h.T, err)

	err = os.WriteFile(path, []byte(content), 0644)
	require.NoError(h.T, err)

	return path
}

// CreateTempDir creates a temporary directory
func (h *TestHelper) CreateTempDir(name string) string {
	path := filepath.Join(h.TempDir, name)
	err := os.MkdirAll(path, 0755)
	require.NoError(h.T, err)
	return path
}

// FileExists checks if a file exists
func (h *TestHelper) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile reads a file and returns its content
func (h *TestHelper) ReadFile(path string) string {
	content, err := os.ReadFile(path)
	require.NoError(h.T, err)
	return string(content)
}

// CaptureOutput captures stdout and stderr
func CaptureOutput(f func()) (string, string, error) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	// Run the function
	f()

	// Restore
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	stdout, _ := io.ReadAll(rOut)
	stderr, _ := io.ReadAll(rErr)

	return string(stdout), string(stderr), nil
}

// Cleanup performs cleanup operations
func (h *TestHelper) Cleanup() {
	if h.TempDir != "" {
		os.RemoveAll(h.TempDir)
	}
}
