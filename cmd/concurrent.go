package cmd

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Task represents a concurrent task
type Task struct {
	Action   func() error
	Progress *ProgressBar
	Name     string
}

// TaskResult holds the result of a task execution
type TaskResult struct {
	Error error
	Name  string
}

// ProgressBar represents a simple progress indicator
type ProgressBar struct {
	stopChan chan bool
	Name     string
	mu       sync.Mutex
	Done     bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(name string) *ProgressBar {
	return &ProgressBar{
		Name:     name,
		stopChan: make(chan bool),
	}
}

const spinnerDelay = 100 * time.Millisecond

// Start begins the progress indicator
func (p *ProgressBar) Start() {
	go func() {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-p.stopChan:
				return
			default:
				p.mu.Lock()
				if !p.Done {
					fmt.Printf("\r%s %s...", spinner[i%len(spinner)], p.Name)
				}
				p.mu.Unlock()
				time.Sleep(spinnerDelay)
				i++
			}
		}
	}()
}

// Complete marks the progress as complete
func (p *ProgressBar) Complete(success bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Done = true
	close(p.stopChan)

	if success {
		fmt.Printf("\r✅ %s completed\n", p.Name)
	} else {
		fmt.Printf("\r❌ %s failed\n", p.Name)
	}
}

// RunConcurrent executes tasks concurrently with progress indication
func RunConcurrent(tasks []Task) []TaskResult {
	var wg sync.WaitGroup
	results := make([]TaskResult, len(tasks))
	resultChan := make(chan TaskResult, len(tasks))

	// Start all progress bars
	for _, task := range tasks {
		if task.Progress != nil {
			task.Progress.Start()
		}
	}

	// Execute tasks concurrently
	for _, task := range tasks {
		wg.Add(1)
		go func(t Task) {
			defer wg.Done()

			err := t.Action()

			if t.Progress != nil {
				t.Progress.Complete(err == nil)
			}

			resultChan <- TaskResult{
				Name:  t.Name,
				Error: err,
			}
		}(task)
	}

	// Wait for all tasks to complete
	wg.Wait()
	close(resultChan)

	// Collect results
	i := 0
	for result := range resultChan {
		results[i] = result
		i++
	}

	return results
}

// AggregateErrors combines multiple errors into a single error message
func AggregateErrors(results []TaskResult) error {
	var errors []string
	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.Name, result.Error))
		}
	}

	if len(errors) == 0 {
		return nil
	}

	if len(errors) == 1 {
		return fmt.Errorf("%s", errors[0])
	}

	return fmt.Errorf("multiple errors occurred:\n  - %s", strings.Join(errors, "\n  - "))
}
