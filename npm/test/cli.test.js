"use strict";

const assert = require("assert");
const path = require("path");
const os = require("os");
const cli = require("../bin/cli.js");

assert.equal(cli.getBinaryName("win32"), "gitlink-cli.exe");
assert.equal(cli.getBinaryName("linux"), "gitlink-cli");
assert.equal(cli.getBinaryName("darwin"), "gitlink-cli");

assert.equal(
  cli.getBinaryPath("win32", "C:\\tmp\\gitlink"),
  path.join("C:\\tmp\\gitlink", "gitlink-cli.exe")
);

const message = cli.formatMissingBinaryError(
  "C:\\tmp\\gitlink-cli.exe",
  "win32",
  "x64"
);
assert.match(message, /binary not found/);
assert.match(message, /Platform: win32\/x64/);
assert.match(message, /npm install -g @gitlink-ai\/cli/);

let exitCode = null;
const stderr = {
  output: "",
  write(text) {
    this.output += text;
  },
};

cli.run(["version"], {
  binaryPath: path.join(os.tmpdir(), "gitlink-cli-test-missing-binary"),
  platform: "win32",
  arch: "x64",
  stderr,
  exit(code) {
    exitCode = code;
    return code;
  },
  execFileSync() {
    throw new Error("execFileSync should not be called for a missing binary");
  },
});

assert.equal(exitCode, 1);
assert.match(stderr.output, /gitlink-cli binary not found/);
assert.match(stderr.output, /Platform: win32\/x64/);

console.log("cli wrapper tests passed");
