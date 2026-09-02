const assert = require("assert");
const crypto = require("crypto");
const fs = require("fs");
const os = require("os");
const path = require("path");
const test = require("node:test");

const {
  getBinaryName,
  getPlatform,
  install,
  parseChecksums,
  verifyChecksum,
} = require("./postinstall");

test("maps supported npm platforms to release asset names", () => {
  assert.deepStrictEqual(getPlatform("linux", "x64"), {
    platform: "linux",
    arch: "amd64",
  });
  assert.deepStrictEqual(getPlatform("darwin", "arm64"), {
    platform: "darwin",
    arch: "arm64",
  });
  assert.strictEqual(getBinaryName("linux", "amd64"), "orchcli_linux_amd64");
  assert.strictEqual(
    getBinaryName("windows", "arm64"),
    "orchcli_windows_arm64.exe",
  );
});

test("rejects unsupported platforms and architectures", () => {
  assert.throws(() => getPlatform("freebsd", "x64"), /Unsupported platform/);
  assert.throws(() => getPlatform("linux", "ppc64"), /Unsupported platform/);
});

test("parses GNU and BSD checksum formats", () => {
  const first = "a".repeat(64);
  const second = "b".repeat(64);
  const parsed = parseChecksums(
    `${first}  orchcli_linux_amd64\n${second} *orchcli_darwin_arm64\r\n`,
  );
  assert.strictEqual(parsed.get("orchcli_linux_amd64"), first);
  assert.strictEqual(parsed.get("orchcli_darwin_arm64"), second);
});

test("verifies the selected release asset checksum", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "orchcli-checksum-"));
  const binary = path.join(dir, "orchcli");
  fs.writeFileSync(binary, "verified bytes");
  const digest = crypto
    .createHash("sha256")
    .update("verified bytes")
    .digest("hex");
  const checksums = new Map([["orchcli_linux_amd64", digest]]);

  assert.strictEqual(
    verifyChecksum(binary, "orchcli_linux_amd64", checksums),
    digest,
  );
  assert.throws(
    () => verifyChecksum(binary, "orchcli_darwin_arm64", checksums),
    /does not contain/,
  );
  fs.rmSync(dir, { recursive: true, force: true });
});

test("installs only a binary matching the published checksum", async () => {
  const binDir = fs.mkdtempSync(path.join(os.tmpdir(), "orchcli-install-"));
  const bytes = Buffer.alloc(2048, 7);
  const digest = crypto.createHash("sha256").update(bytes).digest("hex");

  const result = await install({
    version: "9.9.9",
    detected: { platform: "linux", arch: "amd64" },
    binDir,
    downloadText: async () => `${digest}  orchcli_linux_amd64\n`,
    downloadFile: async (_url, destination) =>
      fs.writeFileSync(destination, bytes),
  });

  assert.strictEqual(result.binaryName, "orchcli_linux_amd64");
  assert.deepStrictEqual(fs.readFileSync(result.binPath), bytes);
  fs.rmSync(binDir, { recursive: true, force: true });
});

test("removes a corrupt download and preserves an existing binary", async () => {
  const binDir = fs.mkdtempSync(path.join(os.tmpdir(), "orchcli-corrupt-"));
  const binPath = path.join(binDir, "orchcli-bin");
  const existing = Buffer.alloc(2048, 3);
  const expected = Buffer.alloc(2048, 4);
  const digest = crypto.createHash("sha256").update(expected).digest("hex");
  fs.writeFileSync(binPath, existing);

  await assert.rejects(
    install({
      version: "9.9.9",
      detected: { platform: "linux", arch: "amd64" },
      binDir,
      downloadText: async () => `${digest}  orchcli_linux_amd64\n`,
      downloadFile: async (_url, destination) =>
        fs.writeFileSync(destination, Buffer.alloc(2048, 5)),
    }),
    /Checksum mismatch/,
  );

  assert.deepStrictEqual(fs.readFileSync(binPath), existing);
  assert.strictEqual(
    fs.existsSync(`${binPath}.download-${process.pid}`),
    false,
  );
  fs.rmSync(binDir, { recursive: true, force: true });
});
