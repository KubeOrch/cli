#!/bin/sh
set -e

# Docker-based installation tests for OrchCLI
# Run: ./test/docker-install-test.sh
#
# Prerequisites: Docker must be running

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$(dirname "$SCRIPT_DIR")"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0

run_test() {
    local name="$1"
    local image="$2"
    local cmd="$3"

    printf "${YELLOW}[TEST]${NC} %-40s " "$name"

    if output=$(docker run --rm "$image" sh -c "$cmd" 2>&1); then
        printf "${GREEN}PASS${NC}\n"
        PASS=$((PASS + 1))
    else
        printf "${RED}FAIL${NC}\n"
        echo "  Output: $output"
        FAIL=$((FAIL + 1))
    fi
}

echo "================================================"
echo "  OrchCLI Docker Installation Tests"
echo "================================================"
echo ""

# Test 1: npm install on Node 20 Alpine
run_test "npm install (node:20-alpine)" \
    "node:20-alpine" \
    "npm install -g @kubeorch/cli && orchcli --version"

# Test 2: npm install on Node 20 Debian
run_test "npm install (node:20-bookworm)" \
    "node:20-bookworm" \
    "npm install -g @kubeorch/cli && orchcli --version"

# Test 3: npm install on Node 24 Alpine
run_test "npm install (node:24-alpine)" \
    "node:24-alpine" \
    "npm install -g @kubeorch/cli && orchcli --version"

# Test 4: curl install on Alpine
run_test "curl install (alpine:3.20)" \
    "alpine:3.20" \
    "apk add --no-cache curl && curl -sfL https://raw.githubusercontent.com/KubeOrch/cli/main/install.sh | sh && orchcli --version"

# Test 5: curl install on Ubuntu
run_test "curl install (ubuntu:24.04)" \
    "ubuntu:24.04" \
    "apt-get update -qq && apt-get install -y -qq curl > /dev/null 2>&1 && curl -sfL https://raw.githubusercontent.com/KubeOrch/cli/main/install.sh | sh && orchcli --version"

# Test 6: wget install on Debian
run_test "wget install (debian:bookworm-slim)" \
    "debian:bookworm-slim" \
    "apt-get update -qq && apt-get install -y -qq wget > /dev/null 2>&1 && wget -qO- https://raw.githubusercontent.com/KubeOrch/cli/main/install.sh | sh && orchcli --version"

echo ""
echo "================================================"
echo "  Results: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC}"
echo "================================================"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
