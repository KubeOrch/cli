# Publishing OrchCTL to npm

This guide explains how to publish the OrchCTL CLI to npm so users can install it via `npm install -g @kubeorchestra/orchcli`.

## Overview

The npm package acts as a wrapper that downloads the appropriate Go binary for the user's platform during installation.

## Directory Structure

```
cli/
├── main.go                  # Go CLI source
├── npm/                     # npm package files
│   ├── package.json
│   ├── bin/
│   │   └── orchcli.js       # Node.js wrapper
│   └── scripts/
│       └── postinstall.js   # Download binary script
├── .goreleaser.yml          # Build configuration
└── .github/
    └── workflows/
        └── release.yml      # Automated release
```

## Step 1: Create npm Package Files

### package.json
Create `npm/package.json`:

```json
{
  "name": "@kubeorchestra/orchcli",
  "version": "0.1.0",
  "description": "OrchCTL - KubeOrchestra Developer CLI",
  "bin": {
    "orchcli": "./bin/orchcli.js"
  },
  "scripts": {
    "postinstall": "node scripts/postinstall.js"
  },
  "files": [
    "bin/",
    "scripts/",
    "README.md"
  ],
  "keywords": [
    "kubernetes",
    "orchestration",
    "cli",
    "kubeorchestra",
    "developer-tools"
  ],
  "repository": {
    "type": "git",
    "url": "https://github.com/KubeOrchestra/cli"
  },
  "bugs": {
    "url": "https://github.com/KubeOrchestra/cli/issues"
  },
  "homepage": "https://github.com/KubeOrchestra/cli#readme",
  "author": "KubeOrchestra",
  "license": "Apache-2.0",
  "engines": {
    "node": ">=14.0.0"
  }
}
```

### Binary Wrapper
Create `npm/bin/orchcli.js`:

```javascript
#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

// Determine binary name based on platform
const platform = process.platform;
const binaryName = platform === 'win32' ? 'orchcli.exe' : 'orchcli';
const binaryPath = path.join(__dirname, '..', 'binaries', binaryName);

// Check if binary exists
if (!fs.existsSync(binaryPath)) {
  console.error('Error: orchcli binary not found.');
  console.error('Please try reinstalling: npm install -g @kubeorchestra/orchcli');
  process.exit(1);
}

// Spawn the binary with all arguments
const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  env: process.env
});

// Handle exit
child.on('exit', (code) => {
  process.exit(code);
});

// Handle errors
child.on('error', (err) => {
  console.error('Failed to start orchcli:', err);
  process.exit(1);
});
```

### Post-Install Script
Create `npm/scripts/postinstall.js`:

```javascript
#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// Get package version
const { version } = require('../package.json');

// Platform mapping
const PLATFORM_MAP = {
  'darwin-x64': 'darwin-amd64',
  'darwin-arm64': 'darwin-arm64',
  'linux-x64': 'linux-amd64',
  'linux-arm64': 'linux-arm64',
  'linux-arm': 'linux-arm',
  'win32-x64': 'windows-amd64',
  'win32-ia32': 'windows-386'
};

// Get platform
const platform = `${process.platform}-${process.arch}`;
const binaryName = PLATFORM_MAP[platform];

if (!binaryName) {
  console.error(`Unsupported platform: ${platform}`);
  console.error('Please visit https://github.com/KubeOrchestra/cli/releases to download manually.');
  process.exit(1);
}

// Construct download URL
const ext = process.platform === 'win32' ? '.exe' : '';
const binaryUrl = `https://github.com/KubeOrchestra/cli/releases/download/v${version}/orchcli-${binaryName}${ext}`;

// Prepare paths
const binariesDir = path.join(__dirname, '..', 'binaries');
const outputFile = path.join(binariesDir, `orchcli${ext}`);

// Create binaries directory
if (!fs.existsSync(binariesDir)) {
  fs.mkdirSync(binariesDir, { recursive: true });
}

console.log(`Downloading orchcli v${version} for ${platform}...`);
console.log(`URL: ${binaryUrl}`);

// Download function
function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    
    https.get(url, (response) => {
      // Handle redirects
      if (response.statusCode === 302 || response.statusCode === 301) {
        file.close();
        fs.unlinkSync(dest);
        return download(response.headers.location, dest).then(resolve).catch(reject);
      }
      
      if (response.statusCode !== 200) {
        file.close();
        fs.unlinkSync(dest);
        reject(new Error(`Failed to download: ${response.statusCode}`));
        return;
      }
      
      response.pipe(file);
      
      file.on('finish', () => {
        file.close();
        // Make binary executable on Unix-like systems
        if (process.platform !== 'win32') {
          fs.chmodSync(dest, 0o755);
        }
        resolve();
      });
    }).on('error', (err) => {
      file.close();
      fs.unlinkSync(dest);
      reject(err);
    });
  });
}

// Download and install
download(binaryUrl, outputFile)
  .then(() => {
    console.log('✓ orchcli installed successfully!');
    console.log('Run "orchcli --help" to get started.');
  })
  .catch((err) => {
    console.error('✗ Failed to download orchcli:', err.message);
    console.error('Please visit https://github.com/KubeOrchestra/cli/releases to download manually.');
    process.exit(1);
  });
```

### npm README
Create `npm/README.md`:

```markdown
# @kubeorchestra/orchcli

OrchCTL - The official CLI for KubeOrchestra development and deployment.

## Installation

```bash
npm install -g @kubeorchestra/orchcli
```

## Usage

```bash
# Initialize development environment
orchcli init

# Start local development
orchcli dev start

# Deploy to Kubernetes
orchcli deploy
```

## Documentation

Full documentation available at: https://github.com/KubeOrchestra/cli

## License

Apache-2.0
```

## Step 2: Configure GoReleaser

Create `.goreleaser.yml` in the repository root:

```yaml
project_name: orchcli

before:
  hooks:
    - go mod tidy

builds:
  - id: orchcli
    main: ./main.go
    binary: orchcli
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
      - arm
    goarm:
      - "7"
    ignore:
      - goos: windows
        goarch: arm64
      - goos: windows
        goarch: arm

archives:
  - id: orchcli-archive
    format: binary
    name_template: "{{ .Binary }}-{{ .Os }}-{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"

checksum:
  name_template: 'checksums.txt'

snapshot:
  name_template: "{{ incpatch .Version }}-next"

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'

release:
  github:
    owner: KubeOrchestra
    name: cli
  name_template: "v{{.Version}}"
  draft: false
  prerelease: auto
```

## Step 3: Setup GitHub Actions

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write
  packages: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  npm-publish:
    needs: goreleaser
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '18'
          registry-url: 'https://registry.npmjs.org'

      - name: Update npm package version
        run: |
          VERSION=${GITHUB_REF#refs/tags/v}
          cd npm
          npm version $VERSION --no-git-tag-version
          echo "Updated package.json to version $VERSION"

      - name: Publish to npm
        run: |
          cd npm
          npm publish --access public
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

## Step 4: Setup npm Authentication

### 1. Create npm Account
If you don't have one, create an account at https://www.npmjs.com/

### 2. Generate npm Token
```bash
# Login to npm
npm login

# Generate token
npm token create --read-only=false
```

### 3. Add Token to GitHub Secrets
1. Go to https://github.com/KubeOrchestra/cli/settings/secrets/actions
2. Click "New repository secret"
3. Name: `NPM_TOKEN`
4. Value: Your npm token

## Step 5: Publishing Process

### Manual Publishing (First Release)

```bash
# 1. Build and test locally
go build -o orchcli main.go
./orchcli --version

# 2. Login to npm
npm login

# 3. Publish npm package
cd npm
npm publish --access public
```

### Automated Publishing (Recommended)

```bash
# 1. Update version in npm/package.json
cd npm
npm version patch  # or minor, major

# 2. Commit changes
git add .
git commit -m "chore: bump version to 0.1.1"

# 3. Create and push tag
git tag v0.1.1
git push origin main
git push origin v0.1.1

# GitHub Actions will automatically:
# - Build binaries for all platforms
# - Create GitHub release with binaries
# - Publish npm package
```

## Step 6: Verify Installation

After publishing, users can install:

```bash
# Install globally
npm install -g @kubeorchestra/orchcli

# Verify installation
orchcli --version
orchcli --help
```

## Troubleshooting

### Binary Download Fails
- Check GitHub release exists with correct tag
- Verify binary names match pattern: `orchcli-{os}-{arch}`
- Check network/proxy settings

### npm Publish Fails
- Verify npm token is valid
- Check package name availability
- Ensure you're a member of @kubeorchestra org on npm

### Platform Not Supported
Add platform mapping in `postinstall.js`:
```javascript
const PLATFORM_MAP = {
  'your-platform': 'go-platform-name',
  // ...
};
```

## Version Management

### Semantic Versioning
- MAJOR version: Breaking changes
- MINOR version: New features (backwards compatible)
- PATCH version: Bug fixes

### Version Sync
Keep versions synchronized:
1. `npm/package.json` - npm package version
2. Git tags - `v0.1.0` format
3. Go binary version - Set in build flags

## Security Notes

1. **Checksums**: GoReleaser generates checksums.txt
2. **HTTPS Only**: Downloads use HTTPS
3. **Version Pinning**: npm package downloads specific version
4. **Error Handling**: Graceful failures with helpful messages

## Testing Before Release

```bash
# Test npm package locally
cd npm
npm pack
npm install -g kubeorchestra-orchcli-0.1.0.tgz

# Test binary download
node scripts/postinstall.js

# Verify installation
orchcli --version
```