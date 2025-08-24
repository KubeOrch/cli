# OrchCLI Implementation Tasks

This document lists all the issues that need to be implemented to create the orchcli tool for KubeOrchestra development environment orchestration.

## Phase 1: Core CLI Setup

### Issue #1: Initialize Go Project Structure
**Priority**: High  
**Type**: Setup  
**Description**: Set up the basic Go project structure with Cobra CLI framework
**Tasks**:
- Initialize Go module: `go mod init github.com/kubeorchestra/cli`
- Add Cobra dependency: `go get github.com/spf13/cobra@latest`
- Create main.go with basic CLI entry point
- Set up cmd/ directory structure
- Create root command with version and help flags
- Add version management using build flags

---

### Issue #2: Create Project Directory Structure
**Priority**: High  
**Type**: Setup  
**Description**: Create all necessary directories and configuration files
**Tasks**:
- Create docker/ directory for orchestration files
- Create npm/ directory for npm package files
- Create scripts/ directory for helper scripts
- Create docs/ directory for documentation
- Set up .gitignore to exclude cloned repos (ui/, core/)
- Create README.md with project overview

---

## Phase 2: Init Command Implementation

### Issue #3: Implement Basic Init Command
**Priority**: High  
**Type**: Feature  
**Description**: Create the init command for cloning repositories
**Tasks**:
- Create cmd/init.go with init command structure
- Implement cloneRepo function using git commands
- Add validation for existing directories
- Implement error handling and user feedback
- Add progress indicators during cloning

---

### Issue #4: Add Fork Support to Init Command
**Priority**: High  
**Type**: Feature  
**Description**: Support forking workflow for external contributors
**Tasks**:
- Add --fork-ui and --fork-core flags
- Implement setupUpstream function for fork configuration
- Validate fork repository format (username/repo)
- Configure proper git remotes (origin and upstream)
- Add documentation for fork workflow

---

### Issue #5: Implement Dependency Installation
**Priority**: Medium  
**Type**: Feature  
**Description**: Auto-install dependencies after cloning repos
**Tasks**:
- Detect if npm is installed
- Run npm install in ui/ directory
- Detect if go is installed
- Run go mod download in core/ directory
- Add --skip-deps flag for manual installation
- Handle installation errors gracefully

---

## Phase 3: Dev Command Implementation

### Issue #6: Create Dev Command Structure
**Priority**: High  
**Type**: Feature  
**Description**: Implement dev command with subcommands
**Tasks**:
- Create cmd/dev.go
- Add start, stop, logs, restart subcommands
- Implement docker-compose wrapper functions
- Add --ui-only and --core-only flags
- Validate docker and docker-compose availability

---

### Issue #7: Create Development Docker Compose Files
**Priority**: High  
**Type**: Configuration  
**Description**: Create docker-compose files for development
**Tasks**:
- Create docker/docker-compose.dev.yml
- Configure PostgreSQL service with proper environment
- Configure Core service with volume mounts and hot reload
- Configure UI service with volume mounts and hot reload
- Set up proper networking between services
- Add health checks for all services

---

### Issue #8: Create Development Dockerfiles
**Priority**: High  
**Type**: Configuration  
**Description**: Create Dockerfiles for development containers
**Tasks**:
- Create docker/Dockerfile.core.dev with air for hot reload
- Create docker/Dockerfile.ui.dev with npm dev server
- Optimize for development (not production)
- Configure proper working directories
- Set up non-root user for containers

---

### Issue #9: Implement Service Management
**Priority**: Medium  
**Type**: Feature  
**Description**: Add service management capabilities
**Tasks**:
- Implement dev start with service selection
- Implement dev stop with graceful shutdown
- Implement dev restart for individual services
- Implement dev logs with follow and tail options
- Add dev status to show running services
- Handle partial service startup (ui-only, core-only)

---

## Phase 4: Run Command Implementation

### Issue #10: Implement Run Command for Production Testing
**Priority**: Medium  
**Type**: Feature  
**Description**: Create run command for testing production images
**Tasks**:
- Create cmd/run.go
- Add --version flag for specific versions
- Create docker/docker-compose.prod.yml
- Pull images from ghcr.io registry
- Implement version validation
- Add --pull flag to force image updates

---

### Issue #11: Add Environment Configuration
**Priority**: Medium  
**Type**: Feature  
**Description**: Support environment variable configuration
**Tasks**:
- Add .env.example file
- Implement env file loading in docker-compose
- Add --env-file flag for custom env files
- Document all required environment variables
- Add env validation before starting services

---

## Phase 5: Utility Commands

### Issue #12: Implement Update Command
**Priority**: Low  
**Type**: Feature  
**Description**: Add command to pull latest changes from repos
**Tasks**:
- Create cmd/update.go
- Implement git pull for both repos
- Handle merge conflicts gracefully
- Add --force flag for hard reset
- Show diff summary after update

---

### Issue #13: Implement Status Command
**Priority**: Low  
**Type**: Feature  
**Description**: Show current environment status
**Tasks**:
- Create cmd/status.go
- Check if repos are cloned
- Show git status for each repo
- Display running docker services
- Show port mappings
- Display service health status

---

### Issue #14: Implement Clean Command
**Priority**: Low  
**Type**: Feature  
**Description**: Clean up development environment
**Tasks**:
- Create cmd/clean.go
- Remove docker containers and volumes
- Add --repos flag to remove cloned repos
- Add --cache flag to clear build caches
- Implement confirmation prompts
- Add --force flag to skip confirmations

---

## Phase 6: NPM Package Setup

### Issue #15: Create NPM Package Structure
**Priority**: Medium  
**Type**: Setup  
**Description**: Set up npm package for distribution
**Tasks**:
- Create npm/package.json with proper metadata
- Create npm/bin/orchcli.js wrapper script
- Create npm/scripts/postinstall.js for binary download
- Add platform detection logic
- Create npm/README.md

---

### Issue #16: Implement Binary Download Logic
**Priority**: Medium  
**Type**: Feature  
**Description**: Download appropriate binary during npm install
**Tasks**:
- Implement platform to binary name mapping
- Add HTTPS download with redirect support
- Implement checksum verification
- Add retry logic for failed downloads
- Set proper file permissions on Unix
- Handle proxy configurations

---

## Phase 7: Build and Release

### Issue #17: Configure GoReleaser
**Priority**: High  
**Type**: Configuration  
**Description**: Set up GoReleaser for multi-platform builds
**Tasks**:
- Create .goreleaser.yml configuration
- Configure builds for multiple platforms
- Set up binary naming convention
- Configure changelog generation
- Add build hooks for version injection

---

### Issue #18: Set Up GitHub Actions for Releases
**Priority**: High  
**Type**: CI/CD  
**Description**: Automate release process with GitHub Actions
**Tasks**:
- Create .github/workflows/release.yml
- Configure GoReleaser action
- Add npm publish step
- Set up secrets for npm token
- Add release notes generation
- Configure draft releases for testing

---

### Issue #19: Add Testing Workflow
**Priority**: Medium  
**Type**: CI/CD  
**Description**: Add automated testing for PRs
**Tasks**:
- Create .github/workflows/test.yml
- Add Go tests execution
- Add linting with golangci-lint
- Test binary builds
- Validate docker-compose files
- Check npm package validity

---

## Phase 9: Documentation and Testing

### Issue #20: Create Comprehensive Documentation
**Priority**: Medium  
**Type**: Documentation  
**Description**: Write complete documentation
**Tasks**:
- Create detailed README.md
- Write CONTRIBUTING.md guide
- Add docs/ with detailed guides
- Create video tutorials
- Add troubleshooting guide
- Document all commands with examples

---

### Issue #21: Add Integration Tests
**Priority**: Medium  
**Type**: Testing  
**Description**: Create integration test suite
**Tasks**:
- Set up test environment
- Test init command with mocked git
- Test dev commands with docker
- Test npm package installation
- Add CI integration for tests
- Create test fixtures

---

### Issue #22: Add Shell Completion Support
**Priority**: Low  
**Type**: Feature  
**Description**: Add shell completion for better UX
**Tasks**:
- Generate bash completion script
- Generate zsh completion script
- Generate fish completion script
- Add installation instructions
- Test on different shells

---

### Issue #23: Add Concurrent Operations
**Priority**: Low  
**Type**: Optimization  
**Description**: Run operations in parallel where possible
**Tasks**:
- Clone repos concurrently in init
- Install dependencies in parallel
- Start services concurrently
- Add progress bars for long operations
- Implement proper error aggregation

---

## Implementation Order

### Milestone 1: MVP (Issues #1-9)
Get basic init and dev commands working

### Milestone 2: Production Testing (Issues #10-11)
Add ability to test production images

### Milestone 3: NPM Distribution (Issues #15-19)
Enable npm installation

### Milestone 4: Polish (Issues #12-14, #23-24)
Add utility commands and documentation

### Milestone 5: Advanced Features (Issues #20-22, #25-28)
Add advanced functionality

## Notes

- Each issue should be created as a GitHub issue with appropriate labels
- Issues can be worked on in parallel where dependencies allow
- Priority levels: High (required for MVP), Medium (important features), Low (nice to have)
- Testing should be done continuously, not just in the testing phase
- Documentation should be updated as features are implemented