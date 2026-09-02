# OrchCLI - KubeOrch Developer CLI

[![Apache 2.0 License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CNCF Aspiring](https://img.shields.io/badge/CNCF-Aspiring-blue.svg)](https://www.cncf.io/projects/)

OrchCLI is a command-line tool for developing with the KubeOrch platform. It streamlines local development, testing, and contribution workflows for cloud-native applications.

## Installation

### Native Install (Recommended)

**macOS, Linux, WSL:**
```bash
curl -sfL https://kubeorch.dev/install.sh | sh
```

**Windows PowerShell:**
```powershell
irm https://kubeorch.dev/install.ps1 | iex
```

**Windows CMD:**
```batch
curl -fsSL https://kubeorch.dev/install.cmd -o install.cmd && install.cmd && del install.cmd
```

### Via NPM
```bash
npm install -g @kubeorch/cli
```

### Via Go
```bash
go install github.com/kubeorch/cli@latest
```

### From Source
```bash
git clone https://github.com/KubeOrch/cli
cd cli
make install
```

### Advanced Options
```bash
# Install specific version
curl -sfL https://kubeorch.dev/install.sh | ORCHCLI_VERSION=v0.0.5 sh

# Install to custom directory
curl -sfL https://kubeorch.dev/install.sh | ORCHCLI_INSTALL_DIR=~/.local/bin sh

# Uninstall
curl -sfL https://kubeorch.dev/install.sh | sh -s -- --uninstall
```

## Features

- **Concurrent Operations** - Fast parallel execution for repository cloning and dependency installation
- **Safe Configuration Management** - File locking prevents corruption during concurrent access
- **Reliable Project Discovery** - Resolves `.kubeorch/project.json` from the current directory or any child directory
- **Existing Checkout Support** - Adopts local UI/Core repositories without cloning or overwriting them
- **Fast Local Iteration** - UI changes hot-reload; Core runs directly on the host for quick restarts

## Commands

| Command | Description |
|---------|-------------|
| `orchcli init` | Initialize environment |
| `orchcli start` | Start services |
| `orchcli stop` | Stop services |
| `orchcli restart [service]` | Restart services |
| `orchcli logs` | View service logs |
| `orchcli status` | Check service status |
| `orchcli exec <service> [command]` | Execute command in service container |
| `orchcli debug` | Debug service connectivity |

### Common Flags

- `orchcli start -d` - Run services in background
- `orchcli stop -v` - Remove volumes when stopping  
- `orchcli logs -f` - Follow log output
- `orchcli logs --tail 50` - Show last 50 lines
- `orchcli init --fork-ui` - Clone UI repository
- `orchcli init --fork-core` - Clone Core repository
- `orchcli init --ui-path ./ui --core-path ./core` - Use existing repositories

## Quick Start

### Production Mode
```bash
# Initialize and start services
orchcli init
orchcli start -d

# Access application
# UI: http://localhost:3001
# API: http://localhost:3000/v1/api

# View logs
orchcli logs -f

# Stop services
orchcli stop
```

The currently published Core and UI `v0.0.3` images are pinned by digest and are
available for AMD64 only. Use source development mode on ARM64 until multi-arch
release images are published.

The shell and npm installers verify the selected release asset against the
published `checksums.txt`; they fail instead of compiling an unverified fallback
binary. Release compatibility is validated using the documented
[Community release smoke tests](docs/COMMUNITY-SMOKE.md).

### Development Mode
```bash
# Clone repositories, or adopt checkouts that already exist
orchcli init --fork-ui --fork-core
# orchcli init --ui-path ./ui --core-path ./core

# Start MongoDB in Docker
orchcli start -d

# Start Core (Terminal 1)
cd core && go run .
# Restart this process after changing Core code

# Start UI (Terminal 2)  
cd ui && npm run dev

# Access: UI at localhost:3001, API at localhost:3000/v1/api
```

### Frontend Development Only
```bash
# Clone UI repository
orchcli init --fork-ui

# Start backend services in Docker
orchcli start -d

# Start UI development server
cd ui && npm run dev
```

### Backend Development Only
```bash
# Clone Core repository
orchcli init --fork-core

# Start MongoDB and the published UI image
orchcli start -d

# Run Core on the host
cd core && go run .

# Access: UI at localhost:3001, API at localhost:3000/v1/api
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - System design and development modes
- [Configuration Management](docs/CONFIGURATION.md) - Config system with file locking
- [Concurrent Operations](docs/CONCURRENT-OPERATIONS.md) - Parallel task execution
- [Release Process](docs/RELEASE.md) - How to create releases
- [Publishing](docs/PUBLISHING.md) - NPM and GitHub release process
- [Automated Release](docs/AUTOMATED-RELEASE.md) - CI/CD pipeline details

## Contributing

See the [contributing guide](https://github.com/KubeOrch/.github/blob/main/CONTRIBUTING.md).

## License

[Apache 2.0](LICENSE)
