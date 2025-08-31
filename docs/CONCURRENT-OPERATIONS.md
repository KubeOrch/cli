# OrchCLI Concurrent Operations

## Overview

OrchCLI implements concurrent task execution to improve performance when running multiple independent operations. This feature significantly reduces waiting time for operations like cloning repositories, pulling Docker images, and running health checks.

## Architecture

### Core Components

1. **Task Structure**
```go
type Task struct {
    Action   func() error    // Function to execute
    Progress *ProgressBar    // Visual progress indicator
    Name     string         // Task identifier
}
```

2. **Progress Indicators**
- Animated spinners during execution
- Success (✓) or failure (✗) markers on completion
- Concurrent-safe progress updates

3. **Result Handling**
```go
type TaskResult struct {
    Error error    // Task error if any
    Name  string   // Task identifier for error reporting
}
```

## Features

### Visual Progress

Each concurrent task displays:
- Task name and status
- Animated spinner while running
- Clear success/failure indicators
- Error messages for failed tasks

Example output:
```
⠹ Cloning UI repository...
⠸ Cloning Core repository...
✓ UI repository cloned
✓ Core repository cloned
```

### Error Aggregation

When multiple tasks fail:
- All errors are collected
- Formatted as a multi-line error message
- Each error clearly attributed to its task

### Thread-Safe Operations

- Mutex-protected progress updates
- Safe concurrent terminal output
- Proper cleanup of progress indicators

## Usage in OrchCLI

### Repository Cloning

During `orchcli init`, repositories are cloned concurrently:
```go
tasks := []Task{
    {
        Name: "Cloning UI repository",
        Action: func() error {
            return cloneRepo(uiURL, uiPath)
        },
    },
    {
        Name: "Cloning Core repository",
        Action: func() error {
            return cloneRepo(coreURL, corePath)
        },
    },
}
RunConcurrentTasks(tasks)
```

### Docker Operations

Starting services runs health checks concurrently:
```go
tasks := []Task{
    {Name: "Checking PostgreSQL", Action: checkPostgres},
    {Name: "Checking Core API", Action: checkCore},
    {Name: "Checking UI", Action: checkUI},
}
```

## Implementation Details

### Task Execution Flow

1. **Initialization**
   - Create progress bars for each task
   - Start spinner animations
   - Initialize result channels

2. **Execution**
   - Launch goroutines for each task
   - Tasks run independently
   - Progress bars update in real-time

3. **Completion**
   - Stop progress animations
   - Display final status (✓/✗)
   - Collect and return results

### Synchronization

```go
func RunConcurrentTasks(tasks []Task) error {
    var wg sync.WaitGroup
    results := make(chan TaskResult, len(tasks))
    
    for _, task := range tasks {
        wg.Add(1)
        go func(t Task) {
            defer wg.Done()
            err := t.Action()
            results <- TaskResult{Error: err, Name: t.Name}
        }(task)
    }
    
    wg.Wait()
    // Process results...
}
```

## Performance Benefits

### Before (Sequential)
```
Cloning UI repository... (5s)
Cloning Core repository... (5s)
Total time: 10s
```

### After (Concurrent)
```
Cloning UI repository... |
Cloning Core repository... | (parallel)
Total time: 5s
```

## Best Practices

1. **Use for Independent Operations**
   - Repository cloning
   - Docker image pulls
   - Health checks
   - File downloads

2. **Avoid for Dependent Operations**
   - Database migrations
   - Sequential setup steps
   - Configuration writes

3. **Error Handling**
   - Always check returned errors
   - Provide clear task names for debugging
   - Consider partial success scenarios

## Testing

The concurrent operations system includes:
- Unit tests for task execution
- Progress bar display tests
- Error aggregation tests
- Race condition testing

Run tests:
```bash
go test ./tests/unit/concurrent_test.go
```

## Limitations

- Terminal output may flicker on some systems
- Not suitable for operations requiring user input
- Progress bars require terminal that supports ANSI escape codes

## Future Enhancements

- Configurable concurrency limits
- Task priority and queuing
- Detailed progress percentages
- Non-terminal output modes
- Task cancellation support