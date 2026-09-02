#!/usr/bin/env node

const crypto = require("crypto");
const fs = require("fs");
const https = require("https");
const os = require("os");
const path = require("path");

const REPO = "KubeOrch/cli";
const MAX_REDIRECTS = 5;
const MIN_BINARY_SIZE = 1000;

function getPlatform(platform = os.platform(), arch = os.arch()) {
  const platformMap = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows",
  };
  const archMap = {
    x64: "amd64",
    arm64: "arm64",
  };

  const mappedPlatform = platformMap[platform];
  const mappedArch = archMap[arch];
  if (!mappedPlatform || !mappedArch) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }
  return { platform: mappedPlatform, arch: mappedArch };
}

function getBinaryName(platform, arch) {
  const extension = platform === "windows" ? ".exe" : "";
  return `orchcli_${platform}_${arch}${extension}`;
}

function parseChecksums(content) {
  const checksums = new Map();
  for (const line of content.split(/\r?\n/)) {
    const match = line.trim().match(/^([a-fA-F0-9]{64})\s+\*?(.+)$/);
    if (match) {
      checksums.set(match[2], match[1].toLowerCase());
    }
  }
  return checksums;
}

function sha256File(filename) {
  const hash = crypto.createHash("sha256");
  hash.update(fs.readFileSync(filename));
  return hash.digest("hex");
}

function verifyChecksum(filename, binaryName, checksums) {
  const expected = checksums.get(binaryName);
  if (!expected) {
    throw new Error(`checksums.txt does not contain ${binaryName}`);
  }
  const actual = sha256File(filename);
  if (actual !== expected) {
    throw new Error(
      `Checksum mismatch for ${binaryName}: expected ${expected}, received ${actual}`,
    );
  }
  return actual;
}

function request(url, onResponse, redirectCount = 0) {
  if (redirectCount > MAX_REDIRECTS) {
    return Promise.reject(
      new Error(`Too many redirects while downloading ${url}`),
    );
  }

  return new Promise((resolve, reject) => {
    const parsedUrl = new URL(url);
    if (parsedUrl.protocol !== "https:") {
      reject(new Error(`Refusing non-HTTPS download URL: ${url}`));
      return;
    }
    const req = https.get(
      parsedUrl,
      { headers: { "User-Agent": "kubeorch-cli-npm-installer" } },
      (response) => {
        if ([301, 302, 303, 307, 308].includes(response.statusCode)) {
          const location = response.headers.location;
          response.resume();
          if (!location) {
            reject(
              new Error(`Redirect from ${url} did not include a location`),
            );
            return;
          }
          resolve(
            request(
              new URL(location, parsedUrl).toString(),
              onResponse,
              redirectCount + 1,
            ),
          );
          return;
        }
        if (response.statusCode !== 200) {
          response.resume();
          reject(
            new Error(`Failed to download ${url}: HTTP ${response.statusCode}`),
          );
          return;
        }
        resolve(onResponse(response));
      },
    );
    req.setTimeout(30000, () =>
      req.destroy(new Error(`Timed out downloading ${url}`)),
    );
    req.on("error", reject);
  });
}

function downloadText(url) {
  return request(
    url,
    (response) =>
      new Promise((resolve, reject) => {
        const chunks = [];
        response.on("data", (chunk) => chunks.push(chunk));
        response.on("end", () =>
          resolve(Buffer.concat(chunks).toString("utf8")),
        );
        response.on("error", reject);
      }),
  );
}

function downloadFile(url, destination) {
  return request(
    url,
    (response) =>
      new Promise((resolve, reject) => {
        const file = fs.createWriteStream(destination, { flags: "wx" });
        response.pipe(file);
        response.on("aborted", () =>
          reject(new Error(`Download aborted for ${url}`)),
        );
        response.on("error", reject);
        file.on("finish", () => file.close(resolve));
        file.on("error", (error) => {
          response.destroy();
          reject(error);
        });
      }),
  );
}

function replaceFile(temporaryPath, destination) {
  if (!fs.existsSync(destination)) {
    fs.renameSync(temporaryPath, destination);
    return;
  }

  try {
    fs.renameSync(temporaryPath, destination);
    return;
  } catch (error) {
    if (
      error.code !== "EEXIST" &&
      error.code !== "EPERM" &&
      error.code !== "EACCES"
    ) {
      throw error;
    }
  }

  const backupPath = `${destination}.previous-${process.pid}`;
  fs.rmSync(backupPath, { force: true });
  fs.renameSync(destination, backupPath);
  try {
    fs.renameSync(temporaryPath, destination);
    fs.rmSync(backupPath, { force: true });
  } catch (error) {
    fs.renameSync(backupPath, destination);
    throw error;
  }
}

async function install(options = {}) {
  const version = options.version || require("../../package.json").version;
  const detected = options.detected || getPlatform();
  const binaryName = getBinaryName(detected.platform, detected.arch);
  const extension = detected.platform === "windows" ? ".exe" : "";
  const binDir = options.binDir || path.join(__dirname, "..", "bin");
  const binPath = path.join(binDir, `orchcli-bin${extension}`);
  const temporaryPath = `${binPath}.download-${process.pid}`;
  const releaseBaseUrl =
    options.releaseBaseUrl ||
    `https://github.com/${REPO}/releases/download/v${version}`;
  const fetchText = options.downloadText || downloadText;
  const fetchFile = options.downloadFile || downloadFile;

  fs.mkdirSync(binDir, { recursive: true });
  fs.rmSync(temporaryPath, { force: true });

  try {
    console.log(
      `Verifying OrchCLI v${version} for ${detected.platform}/${detected.arch}...`,
    );
    const checksums = parseChecksums(
      await fetchText(`${releaseBaseUrl}/checksums.txt`),
    );

    if (fs.existsSync(binPath)) {
      try {
        verifyChecksum(binPath, binaryName, checksums);
        console.log(`OrchCLI v${version} is already installed and verified.`);
        return { binaryName, binPath };
      } catch {
        // Replace an incomplete or stale binary only after its replacement verifies.
      }
    }

    await fetchFile(`${releaseBaseUrl}/${binaryName}`, temporaryPath);
    const size = fs.statSync(temporaryPath).size;
    if (size < MIN_BINARY_SIZE) {
      throw new Error(
        `Downloaded ${binaryName} is unexpectedly small (${size} bytes)`,
      );
    }
    verifyChecksum(temporaryPath, binaryName, checksums);

    if (detected.platform !== "windows") {
      fs.chmodSync(temporaryPath, 0o755);
    }
    replaceFile(temporaryPath, binPath);
    console.log(
      `OrchCLI v${version} installed with a verified SHA256 checksum.`,
    );
    return { binaryName, binPath };
  } catch (error) {
    fs.rmSync(temporaryPath, { force: true });
    throw new Error(
      `Unable to install the published OrchCLI v${version} binary for ` +
        `${detected.platform}/${detected.arch}: ${error.message}`,
    );
  }
}

if (require.main === module) {
  install().catch((error) => {
    console.error(error.message);
    console.error(`Release assets: https://github.com/${REPO}/releases`);
    process.exit(1);
  });
}

module.exports = {
  getBinaryName,
  getPlatform,
  install,
  parseChecksums,
  sha256File,
  verifyChecksum,
};
