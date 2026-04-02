#!/usr/bin/env node

'use strict';

const path = require('node:path');
const os = require('node:os');
const fs = require('node:fs');
const { spawn } = require('node:child_process');

const binaryName = process.platform === 'win32' ? 'mayla.exe' : 'mayla';
const binaryPath = path.join(os.homedir(), '.mayla', binaryName);

if (!fs.existsSync(binaryPath)) {
  process.stderr.write(
    `\n  \u2717  may-la binary not found at: ${binaryPath}\n\n` +
      `  The binary may not have been downloaded during installation.\n` +
      `  Try reinstalling:\n\n` +
      `    npm install -g @alucardeht/may-la-mcp\n\n` +
      `  Or download manually from:\n` +
      `    https://github.com/alucardeht/may-la-mcp/releases\n\n`
  );
  process.exit(1);
}

const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
});

process.on('SIGINT', () => {
  child.kill('SIGINT');
});

process.on('SIGTERM', () => {
  child.kill('SIGTERM');
});

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
  } else {
    process.exit(code ?? 0);
  }
});

child.on('error', (err) => {
  process.stderr.write(`\n  \u2717  Failed to start may-la: ${err.message}\n\n`);
  process.exit(1);
});
