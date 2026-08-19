# OrchCLI Project Configuration

## Canonical Project Marker

`orchcli init` writes `.kubeorch/project.json` in the project root. Lifecycle
commands walk upward from the current working directory and use the nearest
marker, so `orchcli start`, `stop`, `status`, `logs`, `restart`, `exec`, and
`debug` work from the root or any directory below it.

Example full-development marker:

```json
{
  "version": 1,
  "ui_path": "ui",
  "core_path": "core",
  "mode": "development"
}
```

Paths inside the project are stored relative to the marker. Absolute paths are
supported when a checkout lives outside the project root.

## Initialization

Create a production-image project:

```bash
orchcli init
```

Clone source repositories:

```bash
orchcli init --fork-ui --fork-core
```

Adopt existing source repositories without cloning or overwriting them:

```bash
orchcli init --ui-path ./ui --core-path ./core
```

Use `--skip-deps` when dependencies are already installed. Re-running either
the production or existing-checkout form refreshes generated Compose files and
the marker without overwriting `core/config.yaml` or `ui/.env.local`.

## Development Modes

| Source paths | Mode | Docker services | Host services |
|---|---|---|---|
| None | `production` | MongoDB, Core, UI | None |
| UI only | `ui-dev` | MongoDB, Core | UI |
| Core only | `core-dev` | MongoDB, UI | Core |
| UI and Core | `development` | MongoDB | UI and Core |

## Legacy Registry

Versions before the project marker stored `orchcli-config.json` beside the CLI
executable. OrchCLI still reads a matching legacy entry when the current
directory is inside that registered project. It never uses the old
`current_project` value as a fallback for an unrelated directory. Re-run
`orchcli init` in a legacy project to create the canonical marker.

## Errors

Marker errors identify the exact file and repair command. OrchCLI rejects
invalid JSON, unsupported marker versions, mode/path mismatches, and configured
source paths that no longer exist instead of silently selecting another mode.

## Testing

```bash
go test ./...
docker compose -f cmd/docker/docker-compose.dev.yml config --quiet
docker compose -f cmd/docker/docker-compose.prod.yml config --quiet
docker compose -f cmd/docker/docker-compose.hybrid-ui.yml config --quiet
docker compose -f cmd/docker/docker-compose.hybrid-core.yml config --quiet
```
