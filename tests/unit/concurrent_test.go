package unit

import (
	"errors"
	"testing"
	"time"
	
	"github.com/kubeorchestra/cli/cmd"
)

func TestRunConcurrent(t *testing.T) {
	t.Run("executes tasks concurrently", func(t *testing.T) {
		start := time.Now()
		tasks := []cmd.Task{
			{
				Name: "Task 1",
				Action: func() error {
					time.Sleep(100 * time.Millisecond)
					return nil
				},
			},
			{
				Name: "Task 2",
				Action: func() error {
					time.Sleep(100 * time.Millisecond)
					return nil
				},
			},
		}
		
		results := cmd.RunConcurrent(tasks)
		duration := time.Since(start)
		
		if duration >= 200*time.Millisecond {
			t.Errorf("Tasks did not run concurrently. Duration: %v", duration)
		}
		
		for _, result := range results {
			if result.Error != nil {
				t.Errorf("Unexpected error: %v", result.Error)
			}
		}
	})
	
	t.Run("collects errors correctly", func(t *testing.T) {
		tasks := []cmd.Task{
			{
				Name: "Success Task",
				Action: func() error {
					return nil
				},
			},
			{
				Name: "Error Task",
				Action: func() error {
					return errors.New("task failed")
				},
			},
		}
		
		results := cmd.RunConcurrent(tasks)
		
		var errorCount int
		for _, result := range results {
			if result.Error != nil {
				errorCount++
			}
		}
		
		if errorCount != 1 {
			t.Errorf("Expected 1 error, got %d", errorCount)
		}
	})
}

func TestAggregateErrors(t *testing.T) {
	t.Run("returns nil for no errors", func(t *testing.T) {
		results := []cmd.TaskResult{
			{Name: "Task 1", Error: nil},
			{Name: "Task 2", Error: nil},
		}
		
		err := cmd.AggregateErrors(results)
		if err != nil {
			t.Errorf("Expected nil, got %v", err)
		}
	})
	
	t.Run("returns single error", func(t *testing.T) {
		results := []cmd.TaskResult{
			{Name: "Task 1", Error: nil},
			{Name: "Task 2", Error: errors.New("task failed")},
		}
		
		err := cmd.AggregateErrors(results)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		
		if err.Error() != "Task 2: task failed" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
	
	t.Run("aggregates multiple errors", func(t *testing.T) {
		results := []cmd.TaskResult{
			{Name: "Task 1", Error: errors.New("error 1")},
			{Name: "Task 2", Error: errors.New("error 2")},
		}
		
		err := cmd.AggregateErrors(results)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		
		expectedPrefix := "multiple errors occurred:"
		if len(err.Error()) < len(expectedPrefix) {
			t.Errorf("Error message too short: %v", err)
		}
	})
}