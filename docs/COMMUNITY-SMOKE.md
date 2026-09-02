# Community Release Smoke Tests

The Community release gate validates one CLI, Core, and UI release set together.
It covers installation, generated Compose configuration, service health, and a
real browser request from UI to Core.

## Evidence Levels

Pull requests run candidate validation on native Linux AMD64 and Linux ARM64
runners. The workflow builds Core and UI from the commits recorded in
`.github/workflows/community-smoke.yml`, builds the candidate CLI, and exercises:

```text
orchcli init -> orchcli start -d -> orchcli status -> browser request -> orchcli stop
```

The same workflow tests the shell and npm installers on Linux AMD64 and hosted
macOS ARM64 against an existing published CLI release. Both paths must select
the host-native release asset and verify its SHA256 entry from `checksums.txt`.

After compatible component releases exist, run **Community Release Smoke**
manually with:

- the published CLI tag;
- the digest-pinned Core multi-architecture image; and
- the digest-pinned UI multi-architecture image.

Published mode rejects mutable image references and verifies that both indexes
contain `linux/amd64` and `linux/arm64`. Its evidence is uploaded separately for
each native Linux architecture. The supplied image references are expectations:
published mode unsets the candidate overrides and fails unless the released CLI
generates those exact defaults.

## Apple Silicon Runtime Gate

GitHub-hosted ARM64 macOS runners do not support nested virtualization, so they
cannot run Docker Desktop or the full Community Compose stack. They still run
the native and npm installation checks. The complete macOS ARM64 gate must run
on a physical Apple Silicon host with Docker Desktop, Node.js, and `npx`:

```bash
export KUBEORCH_CORE_IMAGE='ghcr.io/kubeorch/core:vX.Y.Z@sha256:<digest>'
export KUBEORCH_UI_IMAGE='ghcr.io/kubeorch/ui:vX.Y.Z@sha256:<digest>'
export KUBEORCH_SMOKE_MODE=published
export KUBEORCH_EVIDENCE_DIR="$PWD/output/playwright/community-smoke-darwin-arm64"

bash .github/scripts/community-smoke.sh "$(command -v orchcli)" arm64
```

The script always tears down the Compose stack. On failure it preserves bounded
container diagnostics. On success it records `evidence.json`, CLI output,
resolved images, status output, and Playwright request evidence under the chosen
evidence directory. These files do not contain credentials.

## Image Overrides

Generated production and hybrid Compose files retain immutable defaults.
`KUBEORCH_CORE_IMAGE` and `KUBEORCH_UI_IMAGE` exist only so release engineering
can validate candidate or newly published component images before changing
those defaults. Normal installations should not set them.

The default pins are updated only after all published-mode evidence passes for
the compatibility set. The release issue must record the exact CLI version,
component digests, host architecture, duration, and evidence run URLs.
