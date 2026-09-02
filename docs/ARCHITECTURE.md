# OrchCLI Runtime Architecture

## Runtime Contract

OrchCLI keeps stateful infrastructure in Docker and runs source checkouts on
the host for a fast edit/reload loop. The standard endpoints are:

| Service | Host endpoint | Container port |
|---|---|---|
| UI | `http://localhost:3001` | `3000` |
| Core API | `http://localhost:3000/v1/api` | `3000` |
| MongoDB | `mongodb://localhost:27017/kubeorchestra` | `27017` |

The UI browser API base URL is
`NEXT_PUBLIC_API_URL=http://localhost:3000/v1/api`. A containerized Core uses
`KUBEORCH_MONGO_URI=mongodb://mongodb:27017/kubeorchestra`; a host Core uses the
generated `core/config.yaml` and connects through `localhost:27017`.

## Modes

| Compose file | Mode | Docker services | Host services |
|---|---|---|---|
| `docker-compose.prod.yml` | Production | MongoDB, Core, UI | None |
| `docker-compose.dev.yml` | Full development | MongoDB | Core, UI |
| `docker-compose.hybrid-ui.yml` | UI development | MongoDB, Core | UI |
| `docker-compose.hybrid-core.yml` | Core development | MongoDB, UI | Core |

Full source development looks like this:

```text
Browser :3001 -> UI (npm run dev)
                    |
                    v
              Core :3000 (go run .)
                    |
                    v
             MongoDB :27017 (Docker)
```

Run it with:

```bash
orchcli init --ui-path ./ui --core-path ./core
orchcli start -d

# Terminal 1
cd core && go run .
# Restart this process after changing Core code

# Terminal 2
cd ui && npm run dev
```

## Image Policy

Generated Compose files use versioned, digest-pinned images. They do not use
floating `latest` tags. MongoDB is multi-arch. The currently published
KubeOrch Core and UI `v0.0.3` images contain AMD64 manifests only; production
and hybrid modes on ARM64 remain dependent on new multi-arch component
releases. Full source development works independently of those release images.

Release validation may set `KUBEORCH_CORE_IMAGE` and `KUBEORCH_UI_IMAGE` to
test an explicit candidate or digest-pinned compatibility set. These overrides
are not a user-facing version-selection mechanism. See
[Community Release Smoke Tests](COMMUNITY-SMOKE.md).

Release smokes also set isolated Compose project, network, port, and volume
names. Normal commands retain the documented defaults.

## Project Discovery

Every project-scoped command resolves the nearest `.kubeorch/project.json`
while walking toward the filesystem root. This makes commands work from nested
UI/Core directories and prevents an unrelated last-used project from being
selected. See [Configuration Management](CONFIGURATION.md) for the schema and
migration behavior.
