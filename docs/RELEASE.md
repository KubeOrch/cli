# Release Guide for OrchCLI

This guide explains how to create a new release of OrchCLI for distribution via GitHub releases and npm.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Release Process](#release-process)
- [Manual Release Steps](#manual-release-steps)
- [Automated Release with GitHub Actions](#automated-release-with-github-actions)
- [NPM Publishing](#npm-publishing)
- [Post-Release Verification](#post-release-verification)

## Prerequisites

Before creating a release, ensure you have:

1. **Go 1.21+** installed for building binaries
2. **Node.js 14+** and npm installed
3. **Git** configured with push access to the repository
4. **NPM account** with access to `@kubeorchestra` organization (for npm publishing)
5. **GitHub access** to create releases in the KubeOrchestra/cli repository

## Release Process

### Step 1: Update Version

1. Update version in `package.json`:
```json
{
  "version": "0.0.2"  // Update this
}
```

2. Commit the version change:
```bash
git add package.json
git commit -m "chore: bump version to v0.0.2"
git push origin main
```

### Step 2: Build Binaries

Build binaries for all platforms:

```bash
# Clean previous builds
rm -rf dist/

# Build for all platforms with version
VERSION=0.0.2 make build-all

# Verify binaries were created
ls -lh dist/
```

Expected output:
```
orchcli_darwin_amd64      # macOS Intel
orchcli_darwin_arm64      # macOS Apple Silicon  
orchcli_linux_amd64       # Linux x64
orchcli_linux_arm64       # Linux ARM
orchcli_windows_amd64.exe # Windows x64
```

### Step 3: Create Checksums

```bash
cd dist
sha256sum * > checksums.txt
cd ..
```

## Manual Release Steps

### Creating GitHub Release

1. **Navigate to releases page**: 
   - Go to https://github.com/KubeOrchestra/cli/releases/new

2. **Create a new tag**:
   - Click "Choose a tag"
   - Type `v0.0.2` (match your version)
   - Select "Create new tag on publish"

3. **Fill release details**:
   - **Release title**: `OrchCLI v0.0.2`
   - **Target branch**: `main`

4. **Upload binaries**:
   - Click "Attach binaries by dropping them here"
   - Upload all files from `dist/` folder:
     - `orchcli_darwin_amd64`
     - `orchcli_darwin_arm64`
     - `orchcli_linux_amd64`
     - `orchcli_linux_arm64`
     - `orchcli_windows_amd64.exe`
     - `checksums.txt`

5. **Add release notes**:

```markdown
## Installation

### Via npm (Recommended)
\`\`\`bash
npm install -g @kubeorchestra/cli
\`\`\`

### Direct Download
Download the appropriate binary for your platform from the assets below.

#### Linux (x64)
\`\`\`bash
wget https://github.com/KubeOrchestra/cli/releases/download/v0.0.2/orchcli_linux_amd64
chmod +x orchcli_linux_amd64
sudo mv orchcli_linux_amd64 /usr/local/bin/orchcli
\`\`\`

#### macOS (Intel)
\`\`\`bash
curl -L https://github.com/KubeOrchestra/cli/releases/download/v0.0.2/orchcli_darwin_amd64 -o orchcli
chmod +x orchcli
sudo mv orchcli /usr/local/bin/
\`\`\`

#### macOS (Apple Silicon)
\`\`\`bash
curl -L https://github.com/KubeOrchestra/cli/releases/download/v0.0.2/orchcli_darwin_arm64 -o orchcli
chmod +x orchcli
sudo mv orchcli /usr/local/bin/
\`\`\`

#### Windows
Download `orchcli_windows_amd64.exe` and add to your PATH.

## What's Changed
- [Add your changes here]
- Bug fixes and improvements

## Checksums
See `checksums.txt` in release assets for SHA256 verification.
```

6. **Publish**:
   - Check "Set as the latest release"
   - Click "Publish release"

## Automated Release with GitHub Actions

The repository includes a GitHub Actions workflow that automatically creates releases when you push a version tag.

### Using Automated Release

1. **Tag the release locally**:
```bash
git tag v0.0.2
git push origin v0.0.2
```

2. **GitHub Actions will automatically**:
   - Build binaries for all platforms
   - Create checksums
   - Create a GitHub release
   - Upload all binaries

3. **Monitor the action**:
   - Go to https://github.com/KubeOrchestra/cli/actions
   - Watch the "Release" workflow progress

## NPM Publishing

After creating the GitHub release, publish to npm:

### Step 1: Prepare NPM Package

```bash
# Build the npm package
make npm-build

# Create package tarball
npm pack
```

### Step 2: Test Locally

```bash
# Install locally to test
npm install -g ./kubeorchestra-cli-0.0.2.tgz

# Test the installation
orchcli --version
```

### Step 3: Publish to NPM

```bash
# Login to npm (if not already logged in)
npm login
# Username: kubeorchestra
# Enter password and email

# Verify login
npm whoami

# Publish the package
npm publish --access public

# Or use the Makefile
make npm-publish
```

### Step 4: Verify NPM Package

```bash
# View the published package
npm view @kubeorchestra/cli

# Test installation from npm
npm install -g @kubeorchestra/cli
orchcli --version
```

## Post-Release Verification

### 1. Verify GitHub Release

- Check https://github.com/KubeOrchestra/cli/releases
- Ensure all binaries are uploaded
- Download and test a binary

### 2. Verify NPM Package

```bash
# Clean install from npm
npm uninstall -g @kubeorchestra/cli
npm install -g @kubeorchestra/cli
orchcli --version
```

### 3. Test Binary Download

The npm postinstall script should download the correct binary:

```bash
# Test on different platforms
# The postinstall script will:
# 1. Detect platform (darwin/linux/windows)
# 2. Download from: https://github.com/KubeOrchestra/cli/releases/download/v0.0.2/orchcli_[platform]_[arch]
# 3. Install to npm/bin/orchcli-bin
```

## Version Naming Convention

- **GitHub tags**: Use `v` prefix (e.g., `v0.0.2`)
- **NPM version**: No prefix (e.g., `0.0.2`)
- **Binary naming**: `orchcli_[os]_[arch]` (e.g., `orchcli_linux_amd64`)

## Troubleshooting

### Binary Not Found After NPM Install

If users report "OrchCLI binary not found!" after npm install:

1. **Check GitHub release exists**:
   - URL: `https://github.com/KubeOrchestra/cli/releases/tag/v{version}`

2. **Check binary naming matches**:
   - Expected: `orchcli_linux_amd64`, `orchcli_darwin_amd64`, etc.

3. **Review postinstall script**:
   - File: `npm/scripts/postinstall.js`
   - Verify download URL construction

### NPM Publish Fails

If npm publish fails:

1. **Check authentication**:
```bash
npm whoami
```

2. **Check package name availability**:
```bash
npm view @kubeorchestra/cli
```

3. **Ensure version is bumped**:
   - Version in package.json must be higher than published version

## Release Checklist

- [ ] Update version in `package.json`
- [ ] Build binaries with `make build-all`
- [ ] Create checksums
- [ ] Create GitHub release with all binaries
- [ ] Test binary download from release
- [ ] Build npm package with `make npm-build`
- [ ] Test npm package locally
- [ ] Publish to npm registry
- [ ] Verify npm installation works
- [ ] Update documentation if needed
- [ ] Announce release (optional)

## Support

For issues with releases:
- GitHub Issues: https://github.com/KubeOrchestra/cli/issues
- NPM Package: https://www.npmjs.com/package/@kubeorchestra/cli