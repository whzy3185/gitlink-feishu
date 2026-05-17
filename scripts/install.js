#!/usr/bin/env node

"use strict";

const os = require("os");
const path = require("path");
const fs = require("fs");
const https = require("https");
const http = require("http");
const { execSync } = require("child_process");

const PACKAGE = require("../package.json");
const VERSION = PACKAGE.version;
const BINARY_NAME = "gitlink-cli";

const RELEASE_BASE = "https://www.gitlink.org.cn";
const REPO_OWNER = "Gitlink";
const REPO_NAME = "gitlink-cli";

function getPlatformInfo() {
  const platform = os.platform();
  const arch = os.arch();
  const platformMap = { darwin: "darwin", linux: "linux", win32: "windows" };
  const archMap = { x64: "amd64", arm64: "arm64" };
  const goPlatform = platformMap[platform];
  const goArch = archMap[arch];
  if (!goPlatform || !goArch) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }
  return { platform: goPlatform, arch: goArch, isWindows: platform === "win32" };
}

function fetch(url, options = {}) {
  return new Promise((resolve, reject) => {
    const maxRedirects = options.maxRedirects || 5;
    let redirectCount = 0;
    function doRequest(currentUrl) {
      const mod = currentUrl.startsWith("https") ? https : http;
      const req = mod.get(currentUrl, (res) => {
        if ((res.statusCode === 301 || res.statusCode === 302 || res.statusCode === 307 || res.statusCode === 308) && res.headers.location) {
          redirectCount++;
          if (redirectCount > maxRedirects) { reject(new Error("Too many redirects")); return; }
          let redirectUrl = res.headers.location;
          if (redirectUrl.startsWith("/")) {
            const parsed = new URL(currentUrl);
            redirectUrl = `${parsed.protocol}//${parsed.host}${redirectUrl}`;
          }
          doRequest(redirectUrl);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`HTTP ${res.statusCode}`));
          return;
        }
        if (options.json) {
          let body = "";
          res.on("data", (chunk) => (body += chunk));
          res.on("end", () => {
            try { resolve(JSON.parse(body)); } catch (e) { reject(new Error("JSON parse failed")); }
          });
        } else {
          const chunks = [];
          res.on("data", (chunk) => chunks.push(chunk));
          res.on("end", () => resolve(Buffer.concat(chunks)));
        }
      });
      req.on("error", reject);
      req.setTimeout(30000, () => { req.destroy(); reject(new Error("Timeout")); });
    }
    doRequest(url);
  });
}

async function findReleaseAsset(platform, arch) {
  const archiveName = `gitlink-cli_${VERSION}_${platform}_${arch}.tar.gz`;
  const tagName = `v${VERSION}`;
  const apiUrl = `${RELEASE_BASE}/api/${REPO_OWNER}/${REPO_NAME}/releases.json`;

  try {
    const releases = await fetch(apiUrl, { json: true });
    const rlist = Array.isArray(releases) ? releases : (releases && releases.releases ? releases.releases : []);
    let release = rlist.find(r => r.tag_name === tagName || r.tag_name === VERSION);
    if (!release && rlist.length > 0) release = rlist[0];

    if (release && release.attachments) {
      let asset = release.attachments.find(a => a.title === archiveName || a.filename === archiveName);
      if (!asset) {
        const pattern = `_${platform}_${arch}.tar.gz`;
        asset = release.attachments.find(a => (a.title || a.filename || "").endsWith(pattern));
      }
      if (asset) {
        let url = asset.url || `${RELEASE_BASE}/api/attachments/${asset.id}`;
        if (url.startsWith("/")) url = RELEASE_BASE + url;
        return url;
      }
    }
  } catch (e) {}

  return `${RELEASE_BASE}/api/${REPO_OWNER}/${REPO_NAME}/releases/${tagName}/assets/${archiveName}`;
}

async function downloadAndExtract(url, destDir, platform) {
  const data = await fetch(url);
  const archivePath = path.join(destDir, "download.tar.gz");
  fs.writeFileSync(archivePath, data);
  execSync(`tar -xzf "${archivePath}" -C "${destDir}"`, { stdio: "pipe" });
  fs.unlinkSync(archivePath);

  const binaryPath = path.join(destDir, BINARY_NAME);
  if (!fs.existsSync(binaryPath)) {
    const files = fs.readdirSync(destDir);
    for (const file of files) {
      const subPath = path.join(destDir, file, BINARY_NAME);
      if (fs.existsSync(subPath)) { fs.renameSync(subPath, binaryPath); break; }
    }
  }
  if (!fs.existsSync(binaryPath)) throw new Error("Binary not found after extraction");
  fs.chmodSync(binaryPath, 0o755);
}

async function main() {
  const { platform, arch } = getPlatformInfo();
  const binDir = path.join(__dirname, "..", "bin");
  if (!fs.existsSync(binDir)) { fs.mkdirSync(binDir, { recursive: true }); }

  const binaryPath = path.join(binDir, BINARY_NAME);
  if (fs.existsSync(binaryPath)) {
    try {
      const output = execSync(`"${binaryPath}" version`, { encoding: "utf-8", stdio: "pipe", timeout: 5000 });
      if (output.includes(VERSION)) {
        console.log(`${BINARY_NAME} v${VERSION} already installed.`);
        return;
      }
    } catch (e) {}
    fs.unlinkSync(binaryPath);
  }

  try {
    const downloadUrl = await findReleaseAsset(platform, arch);
    await downloadAndExtract(downloadUrl, binDir, platform);
    console.log(`${BINARY_NAME} v${VERSION} installed.`);
  } catch (err) {
    // Don't fail npm install — binary can be installed later
    console.warn(`⚠  ${BINARY_NAME} binary download failed: ${err.message}`);
    console.warn(`   Skills are installed. You can install the binary manually later:`);
    console.warn(`   npm run postinstall`);
  }
}

main();
