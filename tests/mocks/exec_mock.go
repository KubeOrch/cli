package mocks

import (
	"os/exec"

	"github.com/stretchr/testify/mock"
)

// CommandExecutor interface for mocking exec.Command
type CommandExecutor interface {
	Command(name string, arg ...string) *exec.Cmd
	LookPath(file string) (string, error)
}

// MockCommandExecutor is a mock implementation
type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) Command(name string, arg ...string) *exec.Cmd {
	args := m.Called(name, arg)
	return args.Get(0).(*exec.Cmd)
}

func (m *MockCommandExecutor) LookPath(file string) (string, error) {
	args := m.Called(file)
	return args.String(0), args.Error(1)
}

// MockCmd represents a mock command for testing
type MockCmd struct {
	mock.Mock
	*exec.Cmd
}

func (m *MockCmd) Run() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCmd) Output() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCmd) CombinedOutput() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}
