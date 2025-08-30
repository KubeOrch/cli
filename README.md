# OrchCLI - KubeOrchestra Developer CLI

OrchCLI is a command-line tool for developing with the KubeOrchestra platform. It streamlines local development, testing, and contribution workflows.

## Installation

### Quick Install
```bash
# Install latest version
curl -sfL https://raw.githubusercontent.com/KubeOrchestra/cli/main/install.sh | sh

# Or with wget
wget -qO- https://raw.githubusercontent.com/KubeOrchestra/cli/main/install.sh | sh

# Install specific version
curl -sfL https://raw.githubusercontent.com/KubeOrchestra/cli/main/install.sh | ORCHCLI_VERSION=v0.0.2 sh

# Install to custom directory
curl -sfL https://raw.githubusercontent.com/KubeOrchestra/cli/main/install.sh | ORCHCLI_INSTALL_DIR=~/.local/bin sh

# Uninstall
curl -sfL https://raw.githubusercontent.com/KubeOrchestra/cli/main/install.sh | sh -s -- --uninstall
```

### Via NPM
```bash
npm install -g @kubeorchestra/cli
```

### Via Go
```bash
go install github.com/kubeorchestra/cli@latest
```

### From Source
```bash
git clone https://github.com/KubeOrchestra/cli
cd cli
make install
```

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

## Quick Start

### Production Mode
```bash
# Initialize and start services
orchcli init
orchcli start -d

# Access application
# UI: http://localhost:3001
# API: http://localhost:3000

# View logs
orchcli logs -f

# Stop services
orchcli stop
```

### Development Mode
```bash
# Clone repositories for development
orchcli init --fork-ui --fork-core

# Start PostgreSQL
orchcli start -d

# Start Core (Terminal 1)
cd core && air

# Start UI (Terminal 2)  
cd ui && npm run dev

# Access: UI at localhost:3001, API at localhost:3000
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

# Start all services (Core with hot reload)
orchcli start -d

# Edit Core files locally - changes auto-reload
# Access: UI at localhost:3001, API at localhost:3000
```

## License

Apache-2.0