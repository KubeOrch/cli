# OrchCLI - KubeOrchestra Developer CLI

[![Apache 2.0 License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Cloud Native](https://img.shields.io/badge/Cloud%20Native-orange.svg)](https://landscape.cncf.io/)

OrchCLI is the official command-line tool for developing with the KubeOrchestra platform. It streamlines local development, testing, and contribution workflows.

## Features

- 🚀 **Quick Setup**: Clone and configure UI/Core repositories with a single command
- 🔥 **Hot Reload**: Development environment with automatic code reloading
- 🍴 **Fork Support**: Seamless workflow for external contributors
- 📦 **Production Testing**: Test latest production images locally
- 🐳 **Docker Integration**: Automated container orchestration

## Installation

### Via Go
```bash
go install github.com/kubeorchestra/cli@latest
```

### Via npm (coming soon)
```bash
npm install -g @kubeorchestra/orchcli
```

### From Source
```bash
git clone https://github.com/kubeorchestra/cli
cd cli
make install
```

## Quick Start

### Production Testing (No Repos Cloned)
```bash
# Initialize for production testing only
orchcli init

# Run with latest Docker images
orchcli run

# Run specific version
orchcli run --version=1.2.3
```

### Development Setup

#### For Team Members
```bash
# Clone official repositories for development
orchcli init --fork-ui --fork-core

# Or clone only what you need
orchcli init --fork-ui     # Frontend development only
orchcli init --fork-core   # Backend development only

# Start development environment
orchcli dev start

# Make changes and commit directly
cd ui
git add .
git commit -m "feat: add new feature"
git push origin main
```

#### For External Contributors
```bash
# Fork repos on GitHub first, then:
orchcli init --fork-ui=youruser/ui --fork-core=youruser/core

# Or work on specific parts:
orchcli init --fork-ui=youruser/ui    # Frontend only
orchcli init --fork-core=youruser/core # Backend only

# Start development
orchcli dev start

# Create PR from your fork
cd ui
git checkout -b feature/my-feature
git push origin feature/my-feature
```

## Commands

| Command | Description |
|---------|-------------|
| `orchcli init` | Initialize environment (production or development) |
| `orchcli dev start` | Start development environment |
| `orchcli dev stop` | Stop all services |
| `orchcli dev logs` | View service logs |
| `orchcli run` | Run production images |
| `orchcli update` | Pull latest changes |
| `orchcli status` | Check environment status |

### Init Command Flags

- **No flags** - Production setup (uses Docker images only, no repos cloned)
- `--fork-ui` - Clone official UI repo (KubeOrchestra/ui) for team members
- `--fork-ui=user/repo` - Clone UI from specified fork for external contributors
- `--fork-core` - Clone official Core repo (KubeOrchestra/core) for team members
- `--fork-core=user/repo` - Clone Core from specified fork for external contributors
- `--skip-deps` - Skip dependency installation

### Dev Command Flags

- `--ui-only` - Run only UI locally (Core as container)
- `--core-only` - Run only Core locally (UI as container)

### Run Command Flags

- `--version=x.y.z` - Specify version for production images

## Project Structure

```
cli/
├── cmd/                 # CLI commands
│   ├── root.go         # Root command
│   ├── init.go         # Repository initialization
│   ├── dev.go          # Development commands
│   └── run.go          # Production testing
├── docker/             # Docker orchestration files
│   ├── docker-compose.dev.yml
│   └── docker-compose.prod.yml
├── npm/                # npm package distribution
├── scripts/            # Helper scripts
└── docs/               # Documentation
```

## Development

### Building
```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build with specific version
make build VERSION=1.0.0
```

### Testing
```bash
# Run tests
make test

# Format code
make fmt

# Run linters
make vet
```

## Architecture

OrchCLI supports two distinct workflows:

### Production Testing Workflow
- **No repositories cloned** - Uses Docker images exclusively
- **Quick testing** - Test latest or specific versions
- **Minimal setup** - Only requires Docker

### Development Workflow
- **Selective cloning** - Clone only the repos you need (UI, Core, or both)
- **Fork support** - Automatic upstream configuration for contributors
- **Hot reload** - Changes reflect immediately during development
- **Flexible modes** - Work on frontend, backend, or full-stack

Key features:
1. **Repository Management**: Intelligent cloning with fork detection
2. **Docker Orchestration**: Seamless container management
3. **Dependency Handling**: Automatic npm/go module installation
4. **Git Integration**: Proper remote setup for forks

The CLI owns all orchestration files while UI/Core repositories remain focused on application code.

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### For External Contributors

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Requirements

- Docker & Docker Compose
- Git
- Go 1.21+ (for Core development)
- Node.js 18+ (for UI development)

## Docker Image Tagging Strategy

When publishing Docker images to GitHub Container Registry (ghcr.io):

### Image Names
- `ghcr.io/kubeorchestra/ui` - Frontend application
- `ghcr.io/kubeorchestra/core` - Backend application

### Tagging Convention
- `latest` - Latest stable release
- `v1.2.3` - Semantic version tags for releases
- `dev` - Latest development build from main branch
- `pr-123` - Pull request builds for testing
- `sha-abc1234` - Commit-specific builds

### Examples
```bash
# Run latest stable
orchcli run

# Run specific version
orchcli run --version=v1.2.3

# Run development version
orchcli run --version=dev
```

## License

Apache-2.0 - see [LICENSE](LICENSE) file for details.

## Support

- 📖 [Documentation](https://github.com/kubeorchestra/cli/docs)
- 🐛 [Issue Tracker](https://github.com/kubeorchestra/cli/issues)
- 💬 [Discussions](https://github.com/kubeorchestra/cli/discussions)

## Maintainers

Maintained by the KubeOrchestra team.

---

Made with ❤️ by the KubeOrchestra community
