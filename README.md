# OrchCLI - KubeOrchestra Developer CLI

[![Apache 2.0 License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Cloud Native](https://img.shields.io/badge/Cloud%20Native-orange.svg)](https://landscape.cncf.io/)

OrchCLI is the official command-line tool for developing with the KubeOrchestra platform. It streamlines local development, testing, and contribution workflows.

## Features

- 🚀 **Quick Setup**: Clone and configure UI/Core repositories with a single command
- 🔥 **Hot Reload**: Development environment with automatic code reloading
- 🍴 **Fork Support**: Seamless workflow for external contributors
- 📦 **Production Testing**: Test latest production images locally
- 🐳 **Docker Integration**: Automated container orchestration with PostgreSQL
- 🔧 **Auto-Install**: Automatic installation of Docker, Git, npm, and Go

## Installation

### Via Go
```bash
go install github.com/kubeorchestra/cli@latest
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

# Start services with latest Docker images
orchcli start

# Start in background (detached mode)
orchcli start -d

# View logs
orchcli logs

# Check status
orchcli status

# Stop services
orchcli stop
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
orchcli start

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
orchcli start

# Create PR from your fork
cd ui
git checkout -b feature/my-feature
git push origin feature/my-feature
```

## Commands

| Command | Description |
|---------|-------------|
| `orchcli init` | Initialize environment (production or development) |
| `orchcli start` | Start KubeOrchestra services |
| `orchcli stop` | Stop all services |
| `orchcli restart` | Restart services |
| `orchcli logs` | View service logs |
| `orchcli status` | Check service status and health |
| `orchcli debug` | Debug network connectivity between services |

### Init Command Flags

- **No flags** - Production setup (uses Docker images only, no repos cloned)
- `--fork-ui` - Clone official UI repo (KubeOrchestra/ui) for team members
- `--fork-ui=user/repo` - Clone UI from specified fork for external contributors
- `--fork-core` - Clone official Core repo (KubeOrchestra/core) for team members
- `--fork-core=user/repo` - Clone Core from specified fork for external contributors
- `--skip-deps` - Skip dependency installation
- `--auto-install` - Automatically install missing dependencies (enabled by default)

### Start Command Flags

- `-d, --detach` - Run services in background

### Stop Command Flags

- `-v, --volumes` - Remove volumes when stopping

### Logs Command Flags

- `-f, --follow` - Follow log output
- `--tail <n>` - Number of lines to show from the end (default: 100)
- `-t, --timestamps` - Show timestamps
- `--service <name>` - Show logs for specific service (ui, core, postgres)

### Restart Command

- `orchcli restart` - Restart all services
- `orchcli restart <service>` - Restart specific service (ui, core, postgres)

## Project Structure

```
cli/
├── cmd/                 # CLI commands
│   ├── root.go         # Root command
│   ├── init.go         # Repository initialization
│   ├── start.go        # Start services
│   ├── stop.go         # Stop services
│   ├── restart.go      # Restart services
│   ├── logs.go         # View logs
│   ├── status.go       # Check status
│   ├── debug.go        # Debug connectivity
│   └── utils.go        # Shared utilities
├── docker/             # Docker orchestration
│   ├── docker-compose.prod.yml     # Production mode
│   ├── docker-compose.dev.yml      # Development mode (both local)
│   ├── docker-compose.hybrid-ui.yml    # UI local, Core from image
│   └── docker-compose.hybrid-core.yml  # Core local, UI from image
├── scripts/            # Helper scripts
└── docs/               # Documentation
```

## Service Architecture

### Network Configuration
All services run on the same Docker network for seamless communication:
- PostgreSQL: `postgres:5432` (internal), `localhost:5432` (host)
- Core API: `core:3000` (internal), `localhost:3000` (host)
- UI: `ui:3001` (internal), `localhost:3001` (host)

### Service Dependencies
- Core waits for PostgreSQL to be healthy before starting
- UI depends on Core being available
- Health checks ensure proper startup sequencing

### Development Modes

1. **Production Mode** (no repos cloned)
   - All services run from Docker images
   - Uses `docker-compose.prod.yml`

2. **Full Development Mode** (both repos cloned)
   - UI and Core run locally with hot reload
   - Uses `docker-compose.dev.yml`
   - UI: Node.js with `npm run dev`
   - Core: Go with Air for hot reload

3. **Hybrid UI Mode** (only UI repo cloned)
   - UI runs locally with hot reload
   - Core runs from Docker image
   - Uses `docker-compose.hybrid-ui.yml`

4. **Hybrid Core Mode** (only Core repo cloned)
   - Core runs locally with hot reload
   - UI runs from Docker image
   - Uses `docker-compose.hybrid-core.yml`

## Auto-Installation Features

OrchCLI automatically installs missing dependencies:

- **Docker**: Installs Docker Engine and starts the daemon
- **Docker Compose**: Installs the latest version
- **Git**: Required for repository operations
- **Node.js & npm**: For UI development (when UI repo is cloned)
- **Go**: For Core development (when Core repo is cloned)

Supports automatic installation on:
- Debian/Ubuntu (via apt)
- macOS (via Homebrew)

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

## Docker Images

### Production Images
The production Docker images are published to GitHub Container Registry:

- `ghcr.io/kubeorchestra/ui:latest` - Frontend application (built from kubeorchestra/ui repo)
- `ghcr.io/kubeorchestra/core:latest` - Backend application (built from kubeorchestra/core repo)

Note: Docker images are built and published from their respective repositories, not from this CLI repo.
The CLI orchestrates these pre-built images using docker-compose.

## Requirements

- Docker & Docker Compose (auto-installed if missing)
- Git (auto-installed if missing)
- Go 1.21+ (auto-installed for Core development)
- Node.js 18+ (auto-installed for UI development)

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### For External Contributors

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

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