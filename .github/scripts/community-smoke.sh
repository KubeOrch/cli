#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  echo "usage: KUBEORCH_CORE_IMAGE=<ref> KUBEORCH_UI_IMAGE=<ref> $0 <orchcli> [amd64|arm64]" >&2
}

cli_input="${1:-${ORCHCLI_BIN:-}}"
expected_arch="${2:-${EXPECTED_ARCH:-}}"
core_image="${KUBEORCH_CORE_IMAGE:-}"
ui_image="${KUBEORCH_UI_IMAGE:-}"
smoke_mode="${KUBEORCH_SMOKE_MODE:-published}"
playwright_package="${PLAYWRIGHT_CLI_PACKAGE:-@playwright/cli@0.1.19}"

if [[ -z "$cli_input" || -z "$core_image" || -z "$ui_image" ]]; then
  usage
  exit 2
fi

case "$smoke_mode" in
  candidate | published) ;;
  *)
    echo "KUBEORCH_SMOKE_MODE must be candidate or published" >&2
    exit 2
    ;;
esac

if [[ "$smoke_mode" == "published" ]]; then
  immutable_ref='^.+@sha256:[0-9a-f]{64}$'
  if [[ ! "$core_image" =~ $immutable_ref || ! "$ui_image" =~ $immutable_ref ]]; then
    echo "Published smokes require digest-pinned Core and UI image references" >&2
    exit 2
  fi
fi

for command_name in docker curl node npx; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "Required command is missing: $command_name" >&2
    exit 2
  fi
done

if [[ "$cli_input" == */* ]]; then
  cli_dir="$(cd "$(dirname "$cli_input")" && pwd)"
  cli="$cli_dir/$(basename "$cli_input")"
else
  cli="$(command -v "$cli_input" || true)"
fi
if [[ -z "$cli" || ! -x "$cli" ]]; then
  echo "OrchCLI executable was not found: $cli_input" >&2
  exit 2
fi

host_os="$(uname -s | tr '[:upper:]' '[:lower:]')"
host_machine="$(uname -m)"
case "$host_machine" in
  x86_64 | amd64) host_arch="amd64" ;;
  arm64 | aarch64) host_arch="arm64" ;;
  *)
    echo "Unsupported host architecture: $host_machine" >&2
    exit 2
    ;;
esac
if [[ -n "$expected_arch" && "$host_arch" != "$expected_arch" ]]; then
  echo "Host architecture is $host_arch, expected $expected_arch" >&2
  exit 1
fi
expected_arch="$host_arch"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
evidence_dir="${KUBEORCH_EVIDENCE_DIR:-$repo_root/output/playwright/community-smoke-$host_os-$host_arch}"
mkdir -p "$evidence_dir"
evidence_dir="$(cd "$evidence_dir" && pwd)"
project_dir="$(mktemp -d "${TMPDIR:-/tmp}/kubeorch-community-smoke.XXXXXX")"
session="kubeorch-community-${host_os}-${host_arch}-$$"
compose_file="$project_dir/docker/docker-compose.prod.yml"
start_epoch="$(date +%s)"
started=false

pwcli() {
  npx --yes --package "$playwright_package" playwright-cli -s="$session" "$@"
}

collect_diagnostics() {
  if [[ -f "$compose_file" ]]; then
    (
      cd "$project_dir"
      docker compose -f "$compose_file" ps --all
      docker compose -f "$compose_file" logs --no-color --tail 200
    ) >"$evidence_dir/compose-diagnostics.log" 2>&1 || true
  fi
}

cleanup() {
  exit_code="$?"
  set +e
  if [[ "$exit_code" -ne 0 ]]; then
    collect_diagnostics
  fi
  pwcli close >/dev/null 2>&1 || true
  if [[ "$started" == true ]]; then
    (
      cd "$project_dir"
      "$cli" stop --volumes
    ) >"$evidence_dir/stop.log" 2>&1 || true
  fi
  rm -f "$project_dir/.kubeorch/project.json"
  rmdir "$project_dir/.kubeorch" >/dev/null 2>&1 || true
  rm -f "$project_dir/docker/"*.yml
  rmdir "$project_dir/docker" "$project_dir/scripts" "$project_dir" >/dev/null 2>&1 || true
  exit "$exit_code"
}
trap cleanup EXIT

assert_platform() {
  image_ref="$1"
  image_name="$2"

  if [[ "$smoke_mode" == "published" ]]; then
    docker pull --platform "linux/$expected_arch" "$image_ref" >/dev/null
    raw_index="$(docker buildx imagetools inspect "$image_ref" --raw)"
    # shellcheck disable=SC2016
    IMAGE_INDEX="$raw_index" node -e '
      const index = JSON.parse(process.env.IMAGE_INDEX);
      const platforms = new Set(
        (index.manifests || [])
          .map((entry) => entry.platform || {})
          .filter((platform) => platform.os !== "unknown")
          .map((platform) => `${platform.os}/${platform.architecture}`)
      );
      for (const required of ["linux/amd64", "linux/arm64"]) {
        if (!platforms.has(required)) {
          throw new Error(`release index is missing ${required}`);
        }
      }
    '
  fi

  actual_platform="$(docker image inspect "$image_ref" --format '{{.Os}}/{{.Architecture}}')"
  if [[ "$actual_platform" != "linux/$expected_arch" ]]; then
    echo "$image_name resolved to $actual_platform, expected linux/$expected_arch" >&2
    exit 1
  fi
}

assert_platform "$core_image" Core
assert_platform "$ui_image" UI

if [[ "$smoke_mode" == "candidate" ]]; then
  export KUBEORCH_CORE_IMAGE="$core_image"
  export KUBEORCH_UI_IMAGE="$ui_image"
else
  # Published evidence must exercise the compatibility set embedded in the CLI.
  unset KUBEORCH_CORE_IMAGE KUBEORCH_UI_IMAGE
fi
port_offset="$(( $$ % 400 ))"
export KUBEORCH_CORE_PORT="${KUBEORCH_CORE_PORT:-$((31000 + port_offset))}"
export KUBEORCH_UI_PORT="${KUBEORCH_UI_PORT:-$((31400 + port_offset))}"
export KUBEORCH_MONGO_PORT="${KUBEORCH_MONGO_PORT:-$((31800 + port_offset))}"
export KUBEORCH_BROWSER_API_URL="${KUBEORCH_BROWSER_API_URL:-http://localhost:$KUBEORCH_CORE_PORT/v1/api}"
export KUBEORCH_CORS_ALLOWED_ORIGINS="${KUBEORCH_CORS_ALLOWED_ORIGINS:-http://localhost:$KUBEORCH_UI_PORT}"
export KUBEORCH_COMPOSE_PROJECT="${KUBEORCH_COMPOSE_PROJECT:-kubeorchestra-smoke-$host_arch-$$}"
export KUBEORCH_NETWORK_NAME="${KUBEORCH_NETWORK_NAME:-kubeorchestra-smoke-$host_arch-$$}"
export KUBEORCH_MONGODB_DATA_VOLUME="${KUBEORCH_MONGODB_DATA_VOLUME:-kubeorchestra-smoke-data-$host_arch-$$}"
export KUBEORCH_MONGODB_CONFIG_VOLUME="${KUBEORCH_MONGODB_CONFIG_VOLUME:-kubeorchestra-smoke-config-$host_arch-$$}"
export KUBEORCH_MONGODB_CONTAINER="${KUBEORCH_MONGODB_CONTAINER:-kubeorchestra-smoke-mongodb-$host_arch-$$}"
export KUBEORCH_CORE_CONTAINER="${KUBEORCH_CORE_CONTAINER:-kubeorchestra-smoke-core-$host_arch-$$}"
export KUBEORCH_UI_CONTAINER="${KUBEORCH_UI_CONTAINER:-kubeorchestra-smoke-ui-$host_arch-$$}"

(
  cd "$project_dir"
  "$cli" init
) | tee "$evidence_dir/init.log"

if [[ ! -f "$compose_file" ]]; then
  echo "orchcli init did not generate $compose_file" >&2
  exit 1
fi

(
  cd "$project_dir"
  docker compose -f "$compose_file" config --images
) | tee "$evidence_dir/compose-images.log"
grep -Fqx "$core_image" "$evidence_dir/compose-images.log"
grep -Fqx "$ui_image" "$evidence_dir/compose-images.log"

started=true
(
  cd "$project_dir"
  "$cli" start -d
) | tee "$evidence_dir/start.log"

wait_for_url() {
  label="$1"
  url="$2"
  attempts=0
  while [[ "$attempts" -lt 90 ]]; do
    if curl --fail --silent --show-error --output /dev/null "$url"; then
      return 0
    fi
    attempts=$((attempts + 1))
    sleep 2
  done
  echo "$label did not become healthy at $url" >&2
  return 1
}

api_origin="$KUBEORCH_BROWSER_API_URL"
ui_origin="http://localhost:$KUBEORCH_UI_PORT"
wait_for_url Core "http://127.0.0.1:$KUBEORCH_CORE_PORT/v1"
wait_for_url UI "http://127.0.0.1:$KUBEORCH_UI_PORT/login"

wait_for_service_health() {
  service="$1"
  attempts=0
  while [[ "$attempts" -lt 90 ]]; do
    container_id="$(
      cd "$project_dir"
      docker compose -f "$compose_file" ps --quiet "$service"
    )"
    if [[ -n "$container_id" ]]; then
      health="$(docker inspect "$container_id" --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}')"
      if [[ "$health" == "healthy" ]]; then
        return 0
      fi
      if [[ "$health" == "unhealthy" ]]; then
        echo "$service container became unhealthy" >&2
        return 1
      fi
    fi
    attempts=$((attempts + 1))
    sleep 2
  done
  echo "$service container did not become healthy" >&2
  return 1
}

for service in mongodb core ui; do
  wait_for_service_health "$service"
done

(
  cd "$project_dir"
  "$cli" status
) | tee "$evidence_dir/status.log"

pwcli install-browser chromium >"$evidence_dir/playwright-install.log"
pwcli open "$ui_origin/login" >"$evidence_dir/playwright-open.log"
pwcli snapshot >"$evidence_dir/playwright-snapshot.log"
pwcli requests >"$evidence_dir/playwright-requests.log"
if ! grep -F "$api_origin/auth/methods" "$evidence_dir/playwright-requests.log" | \
  grep -F '=> [200]' >/dev/null; then
  echo "Browser did not complete the configured Core authentication request" >&2
  exit 1
fi

cli_version="$($cli --version | head -n 1 | tr -d '\r')"
duration_seconds="$(( $(date +%s) - start_epoch ))"
export SMOKE_HOST_OS="$host_os"
export SMOKE_HOST_ARCH="$host_arch"
export SMOKE_MODE="$smoke_mode"
export SMOKE_CLI_VERSION="$cli_version"
export SMOKE_CORE_IMAGE="$core_image"
export SMOKE_UI_IMAGE="$ui_image"
export SMOKE_DURATION_SECONDS="$duration_seconds"
export SMOKE_BROWSER_REQUEST="$api_origin/auth/methods"
# shellcheck disable=SC2016
node -e '
  const fs = require("fs");
  const evidence = {
    schema_version: 1,
    mode: process.env.SMOKE_MODE,
    host: { os: process.env.SMOKE_HOST_OS, arch: process.env.SMOKE_HOST_ARCH },
    cli_version: process.env.SMOKE_CLI_VERSION,
    core_image: process.env.SMOKE_CORE_IMAGE,
    ui_image: process.env.SMOKE_UI_IMAGE,
    duration_seconds: Number(process.env.SMOKE_DURATION_SECONDS),
    browser_request: process.env.SMOKE_BROWSER_REQUEST,
    result: "passed"
  };
  fs.writeFileSync(process.argv[1], `${JSON.stringify(evidence, null, 2)}\n`);
' "$evidence_dir/evidence.json"

echo "Community smoke passed on $host_os/$host_arch in ${duration_seconds}s"
