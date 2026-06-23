#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');
const os = require('os');

const REPO = 'KubeOrch/cli';
const VERSION = require('../../package.json').version;

function getPlatform() {
  const platform = os.platform();
  const arch = os.arch();

  const platformMap = {
    'darwin': 'darwin',
    'linux': 'linux',
    'win32': 'windows'
  };

  const archMap = {
    'x64': 'amd64',
    'arm64': 'arm64'
  };

  const mappedPlatform = platformMap[platform];
  const mappedArch = archMap[arch];

  if (!mappedPlatform || !mappedArch) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }

  return { platform: mappedPlatform, arch: mappedArch };
}

function getBinaryExt() {
  return os.platform() === 'win32' ? '.exe' : '';
}

function downloadBinary(url, dest) {
  return new Promise((resolve, reject) => {
    console.log(`Downloading OrchCLI binary from ${url}...`);

    const file = fs.createWriteStream(dest);

    function followRedirects(currentUrl, redirectCount) {
      if (redirectCount > 5) {
        reject(new Error('Too many redirects'));
        return;
      }

      https.get(currentUrl, (response) => {
        if (response.statusCode === 302 || response.statusCode === 301) {
          file.close();
          followRedirects(response.headers.location, redirectCount + 1);
        } else if (response.statusCode === 200) {
          response.pipe(file);
          file.on('finish', () => {
            file.close(resolve);
          });
        } else {
          file.close();
          reject(new Error(`Failed to download: HTTP ${response.statusCode}`));
        }
      }).on('error', (err) => {
        file.close();
        reject(err);
      });
    }

    followRedirects(url, 0);
  });
}

function buildFromSource() {
  console.log('Building OrchCLI from source...');

  const ext = getBinaryExt();
  const projectRoot = path.join(__dirname, '..', '..');
  const buildOutput = path.join(projectRoot, `orchcli${ext}`);
  const binPath = path.join(__dirname, '..', 'bin', `orchcli-bin${ext}`);

  try {
    execSync('go version', { stdio: 'pipe' });

    console.log('Running go build...');
    const version = require('../../package.json').version;
    const buildDate = new Date().toISOString();
    const ldflags = `-X 'github.com/kubeorch/cli/cmd.version=${version}' -X 'github.com/kubeorch/cli/cmd.buildDate=${buildDate}'`;

    execSync(`go build -ldflags "${ldflags}" -o "${buildOutput}" main.go`, {
      cwd: projectRoot,
      stdio: 'inherit'
    });

    if (fs.existsSync(buildOutput)) {
      fs.renameSync(buildOutput, binPath);
      if (os.platform() !== 'win32') {
        fs.chmodSync(binPath, '755');
      }
      console.log('OrchCLI built successfully from source!');
      return true;
    }
  } catch (error) {
    console.log('Failed to build from source:', error.message);
    return false;
  }

  return false;
}

async function install() {
  const ext = getBinaryExt();
  const binDir = path.join(__dirname, '..', 'bin');
  const binPath = path.join(binDir, `orchcli-bin${ext}`);

  // Skip if binary already exists and is valid
  if (fs.existsSync(binPath)) {
    const stats = fs.statSync(binPath);
    if (stats.size > 1000) {
      console.log('OrchCLI binary already exists, skipping download.');
      return;
    }
    // Remove invalid binary
    fs.unlinkSync(binPath);
  }

  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  try {
    const { platform, arch } = getPlatform();
    const winExt = platform === 'windows' ? '.exe' : '';
    const binaryName = `orchcli_${platform}_${arch}${winExt}`;
    const downloadUrl = `https://github.com/${REPO}/releases/download/v${VERSION}/${binaryName}`;

    await downloadBinary(downloadUrl, binPath);

    // Validate download
    const stats = fs.statSync(binPath);
    if (stats.size < 1000) {
      fs.unlinkSync(binPath);
      throw new Error(`Downloaded file is too small (${stats.size} bytes) - release v${VERSION} may not have binaries`);
    }

    if (os.platform() !== 'win32') {
      fs.chmodSync(binPath, '755');
    }
    console.log(`OrchCLI v${VERSION} installed successfully!`);
  } catch (error) {
    console.log('Failed to download pre-built binary:', error.message);
    console.log('Attempting to build from source...');

    if (!buildFromSource()) {
      console.error('\n===============================================');
      console.error('Failed to install OrchCLI automatically.');
      console.error('Please install manually:');
      console.error(`  curl -sfL https://raw.githubusercontent.com/${REPO}/main/install.sh | sh`);
      console.error('Or download from:');
      console.error(`  https://github.com/${REPO}/releases`);
      console.error('===============================================\n');
      process.exit(1);
    }
  }
}

install().catch(error => {
  console.error('Installation failed:', error);
  process.exit(1);
});
