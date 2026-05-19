"use strict";

const assert = require("assert");
const install = require("../scripts/install.js");
const pkg = require("../package.json");

assert.deepStrictEqual(install.getPlatformInfo("win32", "x64"), {
  platform: "windows",
  arch: "amd64",
  isWindows: true,
});

assert.deepStrictEqual(install.getPlatformInfo("win32", "arm64"), {
  platform: "windows",
  arch: "arm64",
  isWindows: true,
});

assert.deepStrictEqual(install.getPlatformInfo("darwin", "arm64"), {
  platform: "darwin",
  arch: "arm64",
  isWindows: false,
});

assert.deepStrictEqual(install.getPlatformInfo("linux", "x64"), {
  platform: "linux",
  arch: "amd64",
  isWindows: false,
});

assert.equal(install.getBinaryName("windows"), "gitlink-cli.exe");
assert.equal(install.getBinaryName("linux"), "gitlink-cli");
assert.equal(install.getBinaryName("darwin"), "gitlink-cli");

assert.equal(
  install.getArchiveName("windows", "amd64"),
  `gitlink-cli_${pkg.version}_windows_amd64.zip`
);
assert.equal(
  install.getArchiveName("windows", "arm64"),
  `gitlink-cli_${pkg.version}_windows_arm64.zip`
);
assert.equal(
  install.getArchiveName("linux", "amd64"),
  `gitlink-cli_${pkg.version}_linux_amd64.tar.gz`
);

assert.throws(
  () => install.getPlatformInfo("freebsd", "x64"),
  /Unsupported platform/
);

console.log("install helper tests passed");
