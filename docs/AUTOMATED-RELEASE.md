# Automated Release Process

This document explains how the automated release process works for OrchCLI.

## Overview

The release process is **fully automated** using GitHub Actions. When you push a version tag, GitHub Actions will:

1. Build binaries for all platforms
2. Create a GitHub release with all binaries
3. Optionally publish to npm (can be triggered separately)

## How It Works

### Step 1: Push a Version Tag

```bash
# First, commit any changes
git add .
git commit -m "feat: add new feature"
git push origin main

# Create and push a version tag
git tag v0.0.2
git push origin v0.0.2
```

### Step 2: GitHub Actions Takes Over

When you push a tag starting with `v`, the **Release workflow** (`.github/workflows/release.yml`) automatically:

1. **Extracts version** from tag (v0.0.2 → 0.0.2)
2. **Updates package.json** with the version
3. **Builds binaries** for all platforms:
   - `orchcli_darwin_amd64` (macOS Intel)
   - `orchcli_darwin_arm64` (macOS Apple Silicon)
   - `orchcli_linux_amd64` (Linux x64)
   - `orchcli_linux_arm64` (Linux ARM)
   - `orchcli_windows_amd64.exe` (Windows)
4. **Creates checksums** for verification
5. **Creates GitHub release** with all binaries attached

### Step 3: NPM Publishing (Optional)

The **NPM Publish workflow** (`.github/workflows/npm-publish.yml`) can be triggered:

#### Option A: Automatically on Release
When a GitHub release is published, it automatically publishes to npm.

#### Option B: Manual Trigger
Go to Actions → "Publish to NPM" → Run workflow → Enter version

**Note**: You need to set up `NPM_TOKEN` secret in repository settings.

## Setting Up NPM Token

1. **Get NPM token**:
```bash
npm login
npm token create --read-only=false
```

2. **Add to GitHub secrets**:
   - Go to: Settings → Secrets and variables → Actions
   - Add new secret: `NPM_TOKEN`
   - Value: Your npm token

## Complete Flow Diagram

```
Developer                GitHub Actions              NPM Registry
    |                           |                          |
    |--push tag v0.0.2--------->|                          |
    |                           |                          |
    |                    [Build Binaries]                  |
    |                    [Create Release]                  |
    |                           |                          |
    |                    [Trigger NPM Publish]             |
    |                           |------------------------->|
    |                           |                          |
    |                     Release Created            Package Published
```

## What Gets Built

### GitHub Release Contents
- `orchcli_darwin_amd64` - macOS Intel binary
- `orchcli_darwin_arm64` - macOS Apple Silicon binary  
- `orchcli_linux_amd64` - Linux x64 binary
- `orchcli_linux_arm64` - Linux ARM64 binary
- `orchcli_windows_amd64.exe` - Windows binary
- `checksums.txt` - SHA256 checksums

### NPM Package Contents
- `package.json` - Package metadata
- `README.md` - Documentation
- `LICENSE` - License file
- `npm/bin/orchcli` - Shell wrapper script
- `npm/scripts/postinstall.js` - Downloads binary from GitHub

**Note**: NPM package does NOT include binaries. They're downloaded during installation.

## How Users Install

### Via NPM
```bash
npm install -g @kubeorchestra/cli
```

When users run this:
1. NPM downloads the small package (~50KB)
2. Postinstall script runs
3. Detects user's platform
4. Downloads appropriate binary from GitHub release
5. Places it as `npm/bin/orchcli-bin`
6. Ready to use!

### Direct Download
Users can also download binaries directly from GitHub releases.

## Version Naming

- **Git tags**: Use `v` prefix (e.g., `v0.0.2`)
- **NPM version**: No prefix (e.g., `0.0.2`)
- **Binary naming**: `orchcli_[platform]_[arch]`

## Quick Release Commands

For a new release:

```bash
# 1. Make sure you're on main branch with latest changes
git checkout main
git pull origin main

# 2. Create and push tag (this triggers everything!)
git tag v0.0.2
git push origin v0.0.2

# 3. Watch the magic happen
# Go to: https://github.com/KubeOrchestra/cli/actions
```

## Manual NPM Publish

If automatic npm publish fails:

```bash
# Update package.json version
npm version 0.0.2 --no-git-tag-version

# Build wrapper only (no binary)
make npm-build

# Publish
npm publish --access public
```

## Rollback Process

If something goes wrong:

### Rollback GitHub Release
```bash
# Delete the tag locally and remotely
git tag -d v0.0.2
git push origin :refs/tags/v0.0.2
```

### Rollback NPM
```bash
# Deprecate bad version
npm deprecate @kubeorchestra/cli@0.0.2 "Has issues, use 0.0.1"
```

## Benefits of This Approach

1. **One command release**: Just push a tag
2. **Consistent versioning**: Tag version propagates everywhere
3. **No manual building**: GitHub Actions handles all platforms
4. **Small npm package**: Downloads binaries on-demand
5. **Reproducible builds**: Same build environment every time
6. **Automatic checksums**: For security verification

## Monitoring Releases

- **GitHub Actions**: https://github.com/KubeOrchestra/cli/actions
- **GitHub Releases**: https://github.com/KubeOrchestra/cli/releases
- **NPM Package**: https://www.npmjs.com/package/@kubeorchestra/cli

## Troubleshooting

### Release workflow fails
- Check Go version in workflow matches your code
- Ensure all platforms can build (test locally with `make build-all`)

### NPM publish fails
- Check `NPM_TOKEN` secret is set correctly
- Ensure version doesn't already exist on npm

### Users can't install via npm
- Verify GitHub release exists with correct binary names
- Check postinstall script has correct download URLs