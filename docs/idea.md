# OrchCLI - KubeOrchestra Developer CLI

## What is OrchCLI?

A developer tool for working on KubeOrchestra platform. It helps:
- Clone and setup UI/Core repositories for development
- Run local development environment with hot reload
- Handle fork-based contributions for external developers
- Quick production testing with latest images

## Repository Structure

```
cli/
├── .gitignore          # Ignores cloned ui/ and core/ directories
├── cmd/
│   ├── root.go
│   ├── init.go         # Clone repos or setup forks
│   ├── dev.go          # Development commands
│   └── run.go          # Run production images locally
├── docker/
│   ├── docker-compose.dev.yml   # For development (mounts local code)
│   └── docker-compose.prod.yml  # For production testing
└── docs/
```

## Commands Overview

```bash
# Development workflow (clones repos)
orchcli init                    # Clone official repos
orchcli init --fork-ui=user/ui  # Clone from forks
orchcli dev start               # Start with local code
orchcli dev start --ui-only    # Only UI local, others as containers

# Production testing (uses published images)
orchcli run                     # Run latest published images
orchcli run --version=1.2.3    # Run specific version

# Utilities
orchcli update                  # Pull latest changes
orchcli status                  # Check what's running
```

## Git Workflow

### 1. Organization Members
When you run `orchcli init`, it clones the official repos:
```bash
orchcli init
# Clones:
# - https://github.com/KubeOrchestra/ui → ./ui
# - https://github.com/KubeOrchestra/core → ./core

cd ui
# Already connected to origin, can push directly
git add .
git commit -m "feat: add new component"
git push origin main
```

### 2. External Contributors (Forks)
```bash
# First, fork repos on GitHub, then:
orchcli init --fork-ui=johndoe/ui --fork-core=johndoe/core

# This sets up:
# - origin: your fork (for pushing)
# - upstream: official repo (for pulling updates)

cd ui
git checkout -b feature/new-component
git add .
git commit -m "feat: add new component"
git push origin feature/new-component
# Then create PR from fork to upstream
```

## Implementation Details

### Init Command (cmd/init.go)
```go
package cmd

import (
    "fmt"
    "os/exec"
    "github.com/spf13/cobra"
)

var (
    forkUI   string
    forkCore string
)

var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize KubeOrchestra development environment",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Determine if using forks or official repos
        uiRepo := "https://github.com/KubeOrchestra/ui"
        coreRepo := "https://github.com/KubeOrchestra/core"
        
        if forkUI != "" {
            uiRepo = fmt.Sprintf("https://github.com/%s", forkUI)
        }
        if forkCore != "" {
            coreRepo = fmt.Sprintf("https://github.com/%s", forkCore)
        }
        
        // Clone repositories
        fmt.Println("📦 Cloning UI repository...")
        if err := cloneRepo(uiRepo, "./ui"); err != nil {
            return err
        }
        
        fmt.Println("📦 Cloning Core repository...")
        if err := cloneRepo(coreRepo, "./core"); err != nil {
            return err
        }
        
        // Setup upstream for forks
        if forkUI != "" {
            fmt.Println("🔗 Setting up upstream for UI fork...")
            setupUpstream("./ui", "https://github.com/KubeOrchestra/ui")
        }
        if forkCore != "" {
            fmt.Println("🔗 Setting up upstream for Core fork...")
            setupUpstream("./core", "https://github.com/KubeOrchestra/core")
        }
        
        // Install dependencies
        fmt.Println("📥 Installing dependencies...")
        installDependencies()
        
        fmt.Println("✅ Development environment ready!")
        fmt.Println("Run 'orchcli dev start' to begin development")
        return nil
    },
}

func cloneRepo(url, path string) error {
    cmd := exec.Command("git", "clone", url, path)
    return cmd.Run()
}

func setupUpstream(path, upstream string) error {
    cmd := exec.Command("git", "remote", "add", "upstream", upstream)
    cmd.Dir = path
    return cmd.Run()
}

func installDependencies() {
    // Install UI dependencies
    cmd := exec.Command("npm", "install")
    cmd.Dir = "./ui"
    cmd.Run()
    
    // Download Go modules
    cmd = exec.Command("go", "mod", "download")
    cmd.Dir = "./core"
    cmd.Run()
}

func init() {
    initCmd.Flags().StringVar(&forkUI, "fork-ui", "", "UI fork repository (e.g., username/ui)")
    initCmd.Flags().StringVar(&forkCore, "fork-core", "", "Core fork repository (e.g., username/core)")
    rootCmd.AddCommand(initCmd)
}
```

### Dev Command (cmd/dev.go)
```go
var devStartCmd = &cobra.Command{
    Use:   "start",
    Short: "Start development environment with local code",
    RunE: func(cmd *cobra.Command, args []string) error {
        uiOnly, _ := cmd.Flags().GetBool("ui-only")
        coreOnly, _ := cmd.Flags().GetBool("core-only")
        
        if uiOnly {
            // Run UI locally, Core as container
            return runDockerCompose("docker/docker-compose.prod.yml", "core", "postgres")
        } else if coreOnly {
            // Run Core locally, UI as container
            return runDockerCompose("docker/docker-compose.prod.yml", "ui", "postgres")
        }
        
        // Run everything with local code
        return runDockerCompose("docker/docker-compose.dev.yml")
    },
}
```

### Run Command (cmd/run.go)
```go
var runCmd = &cobra.Command{
    Use:   "run",
    Short: "Run production images locally for testing",
    Long:  "Runs the latest published Docker images without cloning repositories",
    RunE: func(cmd *cobra.Command, args []string) error {
        version, _ := cmd.Flags().GetString("version")
        
        // Use docker-compose.prod.yml which pulls images
        return runDockerCompose("docker/docker-compose.prod.yml")
    },
}
```

### Docker Compose Files

**docker/docker-compose.dev.yml** (for development):
```yaml
services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: kubeorchestra
    ports:
      - "5432:5432"

  core:
    build:
      context: ../core
      dockerfile: Dockerfile.dev
    volumes:
      - ../core:/app       # Mount local code
      - /app/vendor
    ports:
      - "3000:3000"
    environment:
      DB_HOST: postgres
    depends_on:
      - postgres
    command: air  # Hot reload

  ui:
    build:
      context: ../ui
      dockerfile: Dockerfile.dev
    volumes:
      - ../ui:/app         # Mount local code
      - /app/node_modules
    ports:
      - "3001:3000"
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:3000
    depends_on:
      - core
    command: npm run dev  # Hot reload
```

**docker/docker-compose.prod.yml** (for production testing):
```yaml
services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: kubeorchestra
    ports:
      - "5432:5432"

  core:
    image: ghcr.io/kubeorchestra/core:${VERSION:-latest}
    ports:
      - "3000:3000"
    environment:
      DB_HOST: postgres
    depends_on:
      - postgres

  ui:
    image: ghcr.io/kubeorchestra/ui:${VERSION:-latest}
    ports:
      - "3001:3000"
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:3000
    depends_on:
      - core
```

### .gitignore
```gitignore
# Cloned repositories (managed separately)
/ui/
/core/

# Binary builds
/bin/
/dist/
orchcli
*.exe

# Go
vendor/
*.mod.sum

# Node
node_modules/
npm/binaries/

# Environment
.env
.env.local

# OS
.DS_Store
```

## Workflows Summary

### For Team Members
```bash
# One-time setup
orchcli init

# Daily development
cd ui  # or core
git pull origin main
orchcli dev start
# Make changes...
git add .
git commit -m "feat: something"
git push origin main
```

### For External Contributors
```bash
# Fork repos on GitHub first, then:
orchcli init --fork-ui=myuser/ui --fork-core=myuser/core

# Development
orchcli dev start
cd ui
git checkout -b feature/cool-feature
# Make changes...
git push origin feature/cool-feature
# Create PR on GitHub
```

### For Testing Production
```bash
# No need to clone anything
orchcli run               # Latest versions
orchcli run --version=1.2.3  # Specific version
```

## Benefits

1. **Clean Separation**: Cloned repos are gitignored, no nested git issues
2. **Direct Git Access**: Developers work directly in ui/ and core/ folders
3. **Fork Support**: External contributors can easily work with forks
4. **Production Testing**: Can run latest images without cloning code
5. **Flexible Workflows**: Support for UI-only, Core-only, or full development

## Installation

```bash
# Via npm
npm install -g @kubeorchestra/orchcli

# Via go
go install github.com/kubeorchestra/cli@latest

# Binary will be called orchcli
```