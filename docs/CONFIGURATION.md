# OrchCLI Configuration Management

## Overview

OrchCLI uses a JSON-based configuration system to manage multiple projects and their settings. The configuration supports concurrent access through file locking to prevent data corruption.

## Configuration Structure

The configuration is stored in `orchcli-config.json` with the following structure:

```json
{
  "projects": {
    "project-name": {
      "path": "/path/to/project",
      "ui_path": "/path/to/ui/repo",
      "core_path": "/path/to/core/repo",
      "mode": "development|production|hybrid-ui|hybrid-core"
    }
  },
  "current_project": "project-name"
}
```

## Configuration Location

The config file is stored in one of these locations (in order of preference):
1. Same directory as the OrchCLI executable
2. `~/.orchcli/` directory (fallback)

## Features

### File Locking

OrchCLI implements file locking using `github.com/gofrs/flock` to ensure safe concurrent access:
- Prevents race conditions when multiple OrchCLI instances run simultaneously
- Uses a `.lock` file alongside the config file
- Automatically releases locks on completion or error

### Project Management

Each project configuration includes:
- **path**: Root directory of the project
- **ui_path**: Path to the UI repository (optional)
- **core_path**: Path to the Core repository (optional)
- **mode**: Development mode based on cloned repositories

### Development Modes

The mode is automatically determined based on cloned repositories:

| Cloned Repos | Mode | Description |
|--------------|------|-------------|
| None | `production` | Uses Docker images for all services |
| UI only | `hybrid-ui` | UI runs locally, backend in Docker |
| Core only | `hybrid-core` | Core mounted in Docker, UI from image |
| Both | `development` | Full local development |

## Configuration API

### Loading Configuration
```go
config, err := LoadConfig()
```
- Returns empty config if file doesn't exist
- Automatically initializes empty projects map

### Saving Configuration
```go
err := SaveConfig(config)
```
- Creates config directory if needed (mode 0750)
- Uses file locking for concurrent safety
- Writes formatted JSON with 2-space indentation

### Getting/Setting Current Project
```go
// Get current project
project := GetCurrentProjectConfig()

// Set current project
err := SetCurrentProject(projectName)
```

### Managing Projects
```go
// Save project configuration
err := SaveProjectConfig(projectName, projectPath, uiPath, corePath)

// Remove project
err := RemoveProjectConfig(projectName)

// Get specific project
project := GetProjectConfig(projectName)
```

## Directory Permissions

OrchCLI uses secure directory permissions:
- Config directory: `0750` (rwxr-x---)
- Config file: `0644` (rw-r--r--)
- Lock file: Managed by flock library

## Error Handling

The configuration system provides detailed error messages for:
- Directory creation failures
- File read/write errors
- JSON parsing issues
- Lock acquisition failures
- Missing project configurations

## Concurrent Access Safety

The file locking mechanism ensures:
1. Only one process can write to config at a time
2. Reads wait for writes to complete
3. Automatic cleanup of lock files
4. Graceful handling of stale locks

## Best Practices

1. **Always use the provided API functions** - Don't directly modify the config file
2. **Check for errors** - All config operations can fail and should be handled
3. **Avoid long-running operations while holding config** - Load, modify, and save quickly
4. **Use project-specific configs** - Store project settings within their respective config entries

## Testing

The configuration system includes comprehensive tests:
- Unit tests for all config operations
- Concurrent access stress tests
- File permission validation
- Error condition handling

Run tests with:
```bash
go test ./tests/unit/config_test.go
go test ./tests/unit/config_concurrent_test.go
```

## Future Enhancements

- Migration support for config schema changes
- Backup and restore functionality
- Config validation and schema enforcement
- Environment-specific configurations
- Config encryption for sensitive data