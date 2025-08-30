#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');
const os = require('os');

const REPO = 'KubeOrchestra/cli';
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
  
  return `${mappedPlatform}_${mappedArch}`;
}

function downloadBinary(url, dest) {
  return new Promise((resolve, reject) => {
    console.log(`Downloading OrchCLI binary from ${url}...`);
    
    const file = fs.createWriteStream(dest);
    
    https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Follow redirect
        https.get(response.headers.location, (redirectResponse) => {
          redirectResponse.pipe(file);
          file.on('finish', () => {
            file.close(resolve);
          });
        }).on('error', reject);
      } else if (response.statusCode === 200) {
        response.pipe(file);
        file.on('finish', () => {
          file.close(resolve);
        });
      } else {
        reject(new Error(`Failed to download: ${response.statusCode}`));
      }
    }).on('error', reject);
  });
}

function buildFromSource() {
  console.log('Building OrchCLI from source...');
  
  const projectRoot = path.join(__dirname, '..', '..');
  const buildOutput = path.join(projectRoot, 'orchcli');
  const binPath = path.join(__dirname, '..', 'bin', 'orchcli-bin');
  
  try {
    // Check if Go is installed
    execSync('go version', { stdio: 'pipe' });
    
    // Build the binary with version info
    console.log('Running go build...');
    const version = require('../../package.json').version;
    const buildDate = new Date().toISOString();
    const ldflags = `-X 'github.com/kubeorchestra/cli/cmd.version=${version}' -X 'github.com/kubeorchestra/cli/cmd.buildDate=${buildDate}'`;
    
    execSync(`go build -ldflags "${ldflags}" -o ${buildOutput} main.go`, { 
      cwd: projectRoot,
      stdio: 'inherit'
    });
    
    // Move to bin directory
    if (fs.existsSync(buildOutput)) {
      fs.renameSync(buildOutput, binPath);
      fs.chmodSync(binPath, '755');
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
  const binDir = path.join(__dirname, '..', 'bin');
  const binPath = path.join(binDir, 'orchcli-bin');
  
  // Skip if binary already exists
  if (fs.existsSync(binPath)) {
    console.log('OrchCLI binary already exists, skipping download.');
    return;
  }
  
  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }
  
  try {
    // Try to download pre-built binary
    const platform = getPlatform();
    const binaryName = `orchcli_${platform}`;
    const downloadUrl = `https://github.com/${REPO}/releases/download/v${VERSION}/${binaryName}`;
    
    await downloadBinary(downloadUrl, binPath);
    fs.chmodSync(binPath, '755');
    console.log('OrchCLI installed successfully!');
  } catch (error) {
    console.log('Failed to download pre-built binary:', error.message);
    console.log('Attempting to build from source...');
    
    // Try building from source
    if (!buildFromSource()) {
      console.error('\n===============================================');
      console.error('Failed to install OrchCLI automatically.');
      console.error('Please install manually by:');
      console.error('1. Cloning the repository: git clone https://github.com/KubeOrchestra/cli.git');
      console.error('2. Running: make install');
      console.error('===============================================\n');
      process.exit(1);
    }
  }
}

// Run installation
install().catch(error => {
  console.error('Installation failed:', error);
  process.exit(1);
});