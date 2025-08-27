# Publishing Workflow

This document describes the complete workflow for publishing OrchCLI to both GitHub and npm.

## Quick Release Commands

For experienced maintainers, here's the quick command sequence:

```bash
# 1. Update version in package.json to 0.0.2
vim package.json

# 2. Commit version bump
git add package.json
git commit -m "chore: bump version to v0.0.2"
git push origin main

# 3. Build all binaries
VERSION=0.0.2 make build-all
cd dist && sha256sum * > checksums.txt && cd ..

# 4. Create and push tag (triggers GitHub Action)
git tag v0.0.2
git push origin v0.0.2

# 5. Publish to npm
make npm-publish
```

## Detailed Workflow

### Phase 1: Pre-Release Preparation

1. **Run tests**:
```bash
make test
make lint
```

2. **Update changelog** (if maintaining one):
```bash
echo "## v0.0.2 - $(date +%Y-%m-%d)" >> CHANGELOG.md
echo "- Your changes here" >> CHANGELOG.md
```

3. **Update version**:
```bash
# Edit package.json
# Change "version": "0.0.1" to "version": "0.0.2"
```

### Phase 2: Building Release Artifacts

1. **Clean previous builds**:
```bash
rm -rf dist/
rm -f *.tgz
```

2. **Build binaries**:
```bash
# Set version for build
export VERSION=0.0.2

# Build all platforms
make build-all

# Verify builds
ls -lh dist/
```

3. **Generate checksums**:
```bash
cd dist
sha256sum * > checksums.txt
cat checksums.txt
cd ..
```

### Phase 3: GitHub Release

#### Option A: Manual Release

1. Go to: https://github.com/KubeOrchestra/cli/releases/new
2. Tag: `v0.0.2`
3. Title: `OrchCLI v0.0.2`
4. Upload all files from `dist/`
5. Add release notes
6. Publish

#### Option B: Using GitHub CLI

```bash
# Install GitHub CLI if needed
# brew install gh (macOS)
# apt install gh (Ubuntu/Debian)

# Authenticate
gh auth login

# Create release
gh release create v0.0.2 \
  --title "OrchCLI v0.0.2" \
  --notes-file RELEASE_NOTES.md \
  dist/*
```

#### Option C: Automated via Git Tag

```bash
# Create tag
git tag -a v0.0.2 -m "Release v0.0.2"

# Push tag (triggers GitHub Action)
git push origin v0.0.2
```

### Phase 4: NPM Publishing

1. **Build npm package**:
```bash
make npm-build
```

2. **Create package**:
```bash
npm pack
# Creates: kubeorchestra-cli-0.0.2.tgz
```

3. **Test locally**:
```bash
# Install globally
npm install -g ./kubeorchestra-cli-0.0.2.tgz

# Test
orchcli --version
# Should show: OrchCLI 0.0.2 ...

# Uninstall test version
npm uninstall -g @kubeorchestra/cli
```

4. **Login to npm**:
```bash
npm login
# Username: kubeorchestra
# Password: [your password]
# Email: [your email]
```

5. **Publish**:
```bash
npm publish --access public

# Or using Makefile
make npm-publish
```

6. **Verify publication**:
```bash
# Check npm registry
npm view @kubeorchestra/cli

# Test installation
npm install -g @kubeorchestra/cli
orchcli --version
```

## Platform-Specific Binary Names

The postinstall script expects these exact binary names in GitHub releases:

| Platform | Architecture | Binary Name |
|----------|-------------|-------------|
| macOS | Intel (x64) | `orchcli_darwin_amd64` |
| macOS | Apple Silicon | `orchcli_darwin_arm64` |
| Linux | x64 | `orchcli_linux_amd64` |
| Linux | ARM64 | `orchcli_linux_arm64` |
| Windows | x64 | `orchcli_windows_amd64.exe` |

## NPM Package Structure

When published, the npm package contains:

```
@kubeorchestra/cli/
├── package.json
├── README.md
├── LICENSE
└── npm/
    ├── bin/
    │   └── orchcli          # Shell wrapper script
    └── scripts/
        ├── postinstall.js   # Downloads correct binary
        └── prepack.js       # Pre-package script
```

Note: Binaries are NOT included in the npm package. They're downloaded during installation.

## How NPM Installation Works

1. User runs: `npm install -g @kubeorchestra/cli`
2. NPM downloads the package (small, ~100KB)
3. Postinstall script runs:
   - Detects user's platform (Linux/macOS/Windows)
   - Downloads appropriate binary from GitHub release
   - Places binary as `npm/bin/orchcli-bin`
4. NPM creates symlink: `/usr/local/bin/orchcli` → wrapper script
5. When user runs `orchcli`, wrapper executes the binary

## Version Synchronization

Keep versions synchronized across:

1. **package.json**: `"version": "0.0.2"`
2. **GitHub tag**: `v0.0.2`
3. **Binary version**: Built with `VERSION=0.0.2`

## Rollback Procedure

If a release has issues:

### Rollback NPM:
```bash
# Unpublish broken version (within 72 hours)
npm unpublish @kubeorchestra/cli@0.0.2

# Or deprecate it
npm deprecate @kubeorchestra/cli@0.0.2 "Critical bug, use 0.0.1"
```

### Rollback GitHub:
1. Go to releases page
2. Edit the problematic release
3. Check "This is a pre-release" or delete it

## Security Considerations

1. **Always create checksums**: Include `checksums.txt` in releases
2. **Sign commits**: Use GPG signing for release commits
3. **Two-factor auth**: Enable 2FA on both GitHub and npm
4. **Verify binaries**: Test each platform binary before release

## Common Issues

### Issue: NPM postinstall fails

**Cause**: GitHub release not found or binaries misnamed

**Fix**: Ensure GitHub release exists with correct binary names

### Issue: Wrong version shown

**Cause**: Binary built without VERSION variable

**Fix**: Always use `VERSION=x.x.x make build-all`

### Issue: NPM publish fails with 403

**Cause**: Not logged in or no access to @kubeorchestra

**Fix**: 
```bash
npm login
npm whoami  # Should show: kubeorchestra
```

## Release Frequency

- **Patch releases** (0.0.x): Bug fixes, weekly if needed
- **Minor releases** (0.x.0): New features, bi-weekly/monthly
- **Major releases** (x.0.0): Breaking changes, quarterly

## Maintenance Commands

```bash
# View all releases
gh release list

# Download specific release
gh release download v0.0.2

# View npm versions
npm view @kubeorchestra/cli versions

# Check download stats
npm view @kubeorchestra/cli downloads
```

## Contact

- GitHub Issues: https://github.com/KubeOrchestra/cli/issues
- NPM Support: https://www.npmjs.com/package/@kubeorchestra/cli