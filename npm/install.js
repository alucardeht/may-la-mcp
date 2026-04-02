#!/usr/bin/env node

'use strict';

const https = require('node:https');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const { execSync } = require('node:child_process');

const stderr = (msg) => process.stderr.write(msg);
const stderrln = (msg) => process.stderr.write(msg + '\n');

const PLATFORM_MAP = {
  darwin: 'darwin',
  linux: 'linux',
  win32: 'windows',
};

const ARCH_MAP = {
  x64: 'amd64',
  arm64: 'arm64',
};

const UNSUPPORTED_COMBOS = new Set(['linux_arm64', 'windows_arm64']);

function detectPlatform() {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];

  if (!platform) {
    throw new Error(`Unsupported platform: ${process.platform}`);
  }
  if (!arch) {
    throw new Error(`Unsupported architecture: ${process.arch}`);
  }
  if (UNSUPPORTED_COMBOS.has(`${platform}_${arch}`)) {
    throw new Error(
      `Unsupported platform combination: ${platform} ${arch}\n` +
        `  Please open an issue at https://github.com/alucardeht/may-la-mcp/issues`
    );
  }

  return { platform, arch };
}

function resolveInstallDir() {
  const sudoUser = process.env.SUDO_USER;
  if (sudoUser) {
    const homeBase = process.platform === 'darwin' ? '/Users' : '/home';
    return path.join(homeBase, sudoUser, '.mayla');
  }
  return path.join(os.homedir(), '.mayla');
}

function readPackageVersion() {
  const pkgPath = path.join(__dirname, 'package.json');
  const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
  return pkg.version;
}

function buildDownloadUrl(binaryName, version, platform, arch) {
  const suffix = platform === 'windows' ? '.exe' : '';
  const fileName = `${binaryName}-${platform}_${arch}${suffix}`;
  return `https://github.com/alucardeht/may-la-mcp/releases/download/v${version}/${fileName}`;
}

function buildAuthHeaders() {
  const token = process.env.MAYLA_GITHUB_TOKEN || process.env.GITHUB_TOKEN;
  if (token) {
    return { Authorization: `Bearer ${token}` };
  }
  return {};
}

function downloadFile(url, destPath, maxRedirects = 5) {
  return new Promise((resolve, reject) => {
    const tmpPath = destPath + '.tmp';

    function follow(currentUrl, redirectsLeft) {
      const parsedUrl = new URL(currentUrl);
      const options = {
        hostname: parsedUrl.hostname,
        path: parsedUrl.pathname + parsedUrl.search,
        headers: {
          'User-Agent': 'may-la-mcp-installer',
          ...buildAuthHeaders(),
        },
      };

      const req = https.get(options, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          if (redirectsLeft === 0) {
            reject(new Error('Too many redirects'));
            return;
          }
          follow(res.headers.location, redirectsLeft - 1);
          return;
        }

        if (res.statusCode === 403 || res.statusCode === 429) {
          reject(
            new Error(
              `GitHub API rate limited (HTTP ${res.statusCode}).\n` +
                `  Set MAYLA_GITHUB_TOKEN or GITHUB_TOKEN env var to authenticate.`
            )
          );
          return;
        }

        if (res.statusCode !== 200) {
          reject(new Error(`HTTP ${res.statusCode} downloading ${currentUrl}`));
          return;
        }

        const totalBytes = parseInt(res.headers['content-length'] || '0', 10);
        let receivedBytes = 0;

        const fileStream = fs.createWriteStream(tmpPath);

        res.on('data', (chunk) => {
          receivedBytes += chunk.length;
        });

        res.pipe(fileStream);

        fileStream.on('finish', () => {
          fileStream.close(() => {
            fs.renameSync(tmpPath, destPath);
            resolve({ receivedBytes, totalBytes });
          });
        });

        fileStream.on('error', (err) => {
          fs.unlink(tmpPath, () => {});
          reject(err);
        });
      });

      req.on('error', (err) => {
        fs.unlink(tmpPath, () => {});
        reject(err);
      });
    }

    follow(url, maxRedirects);
  });
}

function formatBytes(bytes) {
  if (bytes === 0) return 'unknown size';
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function padEnd(str, length) {
  while (str.length < length) str += ' ';
  return str;
}

function makeBinaryExecutable(filePath) {
  fs.chmodSync(filePath, 0o755);
}

function removeQuarantine(filePath) {
  if (process.platform !== 'darwin') return;
  try {
    execSync(`xattr -d com.apple.quarantine "${filePath}"`, { stdio: 'ignore' });
  } catch {
    // quarantine attribute may not be present — safe to ignore
  }
}

function binaryExists(installDir, platform) {
  const suffix = platform === 'windows' ? '.exe' : '';
  const maylaPath = path.join(installDir, `mayla${suffix}`);
  return fs.existsSync(maylaPath);
}

function printHeader(platform, arch, version) {
  stderrln('');
  stderrln('  \u2554\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2557');
  stderrln('  \u2551              may-la MCP \u2014 Installation               \u2551');
  stderrln('  \u2560\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2563');
  stderrln(`  \u2551  Platform:  ${padEnd(`${platform} ${arch}`, 42)}\u2551`);
  stderrln(`  \u2551  Version:   ${padEnd(`v${version}`, 42)}\u2551`);
  stderrln('  \u255a\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u255d');
  stderrln('');
}

function printSuccessBox(installDir) {
  const displayDir = installDir.replace(os.homedir(), '~');
  stderrln(`  \u2713  Binaries installed to ${displayDir}/`);
  stderrln('');
  stderrln('  \u250c\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2510');
  stderrln('  \u2502                                                      \u2502');
  stderrln('  \u2502  \u2713 Ready! Add to your MCP client:                   \u2502');
  stderrln('  \u2502                                                      \u2502');
  stderrln('  \u2502  Claude Code:                                        \u2502');
  stderrln('  \u2502    claude mcp add may-la -- may-la-mcp               \u2502');
  stderrln('  \u2502                                                      \u2502');
  stderrln('  \u2502  Other MCP clients:                                  \u2502');
  stderrln('  \u2502    Command: may-la-mcp                              \u2502');
  stderrln('  \u2502                                                      \u2502');
  stderrln('  \u2514\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2518');
  stderrln('');
}

async function downloadBinary(name, url, destPath) {
  const label = padEnd(`  \u2b07  Downloading ${name}...`, 46);
  stderr(label);

  const { receivedBytes, totalBytes } = await downloadFile(url, destPath);
  const size = formatBytes(totalBytes || receivedBytes);
  stderrln(`\u2713 ${size}`);
}

async function main() {
  let platform, arch;

  try {
    ({ platform, arch } = detectPlatform());
  } catch (err) {
    stderrln(`\n  \u2717  ${err.message}\n`);
    return;
  }

  const version = readPackageVersion();
  const installDir = resolveInstallDir();

  printHeader(platform, arch, version);

  fs.mkdirSync(installDir, { recursive: true });

  const suffix = platform === 'windows' ? '.exe' : '';
  const binaries = [
    {
      name: 'mayla',
      url: buildDownloadUrl('mayla', version, platform, arch),
      dest: path.join(installDir, `mayla${suffix}`),
    },
  ];

  const existingBinaries = binaryExists(installDir, platform);
  let downloadFailed = false;

  for (const binary of binaries) {
    try {
      await downloadBinary(binary.name, binary.url, binary.dest);

      if (platform !== 'windows') {
        makeBinaryExecutable(binary.dest);
        removeQuarantine(binary.dest);
      }
    } catch (err) {
      downloadFailed = true;
      stderrln(`\u2717 failed`);
      stderrln(`\n  \u26a0  Failed to download ${binary.name}: ${err.message}`);
    }
  }

  if (downloadFailed) {
    if (existingBinaries) {
      stderrln('');
      stderrln('  \u26a0  Download failed, but existing binaries were found.');
      stderrln('      Using previously installed version.');
      stderrln('');
    } else {
      stderrln('');
      stderrln('  \u2717  Installation failed. No existing binaries found.');
      stderrln('');
      stderrln('  Manual installation:');
      stderrln('    https://github.com/alucardeht/may-la-mcp/releases');
      stderrln('');
      return;
    }
  }

  fs.writeFileSync(path.join(installDir, 'version'), `v${version}`, 'utf8');

  stderrln('');
  printSuccessBox(installDir);
}

main().catch((err) => {
  stderrln(`\n  \u2717  Unexpected error during installation: ${err.message}\n`);
});
