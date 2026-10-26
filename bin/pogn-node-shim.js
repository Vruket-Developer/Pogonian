#!/usr/bin/env node

/**
 * Pogonian Native Binary Shim (Zero-Dependency)
 * Seamless cross-platform launcher for 'pogn' and 'pogonian'.
 *
 * Execution Hierarchy:
 * 1. Local prebuilt binary in package bin/ directory
 * 2. Cached prebuilt binary in ~/.pogonian/bin/
 * 3. Auto-download prebuilt binary from GitHub Releases for current OS/Arch
 * 4. Fallback: Local 'go build' if Go toolchain is installed
 * 5. Helpful error prompt with direct download instructions
 */

const { spawnSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');
const https = require('https');
const http = require('http');

const pkg = require('../package.json');
const VERSION = pkg.version || '0.2.1';
const rootDir = path.resolve(__dirname, '..');
const isWin = process.platform === 'win32';
const ext = isWin ? '.exe' : '';

// 1. Resolve Platform and Architecture
function getTargetPlatformArch() {
    let platform = '';
    switch (process.platform) {
        case 'linux': platform = 'linux'; break;
        case 'darwin': platform = 'darwin'; break;
        case 'win32': platform = 'windows'; break;
        default: platform = process.platform;
    }

    let arch = '';
    switch (process.arch) {
        case 'x64': arch = 'amd64'; break;
        case 'arm64': arch = 'arm64'; break;
        default: arch = process.arch;
    }

    return { platform, arch };
}

const { platform, arch } = getTargetPlatformArch();
const binaryFilename = `pogn-${platform}-${arch}${ext}`;
const localBinaryPath = path.join(__dirname, isWin ? 'pogn.exe' : 'pogn');

// User cache directory: ~/.pogonian/bin/pogn-v0.2.0-linux-amd64
const cacheDir = path.join(os.homedir(), '.pogonian', 'bin');
const cachedBinaryPath = path.join(cacheDir, `pogn-v${VERSION}-${platform}-${arch}${ext}`);

// Helper: check if file exists and is executable
function isValidBinary(filePath) {
    if (!fs.existsSync(filePath)) return false;
    try {
        const stats = fs.statSync(filePath);
        return stats.size > 10000; // Real Go binary is several megabytes
    } catch {
        return false;
    }
}

// Download file following HTTP redirects (e.g. GitHub Releases -> AWS S3)
function downloadFile(url, destPath) {
    return new Promise((resolve, reject) => {
        function fetchUrl(currentUrl, redirectCount = 0) {
            if (redirectCount > 8) {
                return reject(new Error('Too many redirects while downloading binary.'));
            }

            const client = currentUrl.startsWith('http:') ? http : https;
            const req = client.get(currentUrl, {
                headers: {
                    'User-Agent': `pogonian-npm-shim/v${VERSION} (${platform}; ${arch})`,
                    'Accept': 'application/octet-stream, */*'
                }
            }, (res) => {
                if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
                    return fetchUrl(res.headers.location, redirectCount + 1);
                }

                if (res.statusCode !== 200) {
                    return reject(new Error(`HTTP ${res.statusCode}: Failed to fetch ${currentUrl}`));
                }

                const tmpPath = `${destPath}.tmp.${Date.now()}`;
                const file = fs.createWriteStream(tmpPath);

                res.pipe(file);

                file.on('finish', () => {
                    file.close(() => {
                        try {
                            if (!isWin) {
                                fs.chmodSync(tmpPath, 0o755);
                            }
                            fs.renameSync(tmpPath, destPath);
                            resolve();
                        } catch (err) {
                            reject(err);
                        }
                    });
                });

                file.on('error', (err) => {
                    try { fs.unlinkSync(tmpPath); } catch {}
                    reject(err);
                });
            });

            req.on('error', reject);
            req.setTimeout(25000, () => {
                req.destroy(new Error('Connection timed out while downloading Pogonian binary.'));
            });
        }

        fetchUrl(url, 0);
    });
}

// Helper: check if Go compiler is installed locally
function hasLocalGo() {
    try {
        const check = spawnSync('go', ['version'], { stdio: 'ignore' });
        return check.status === 0;
    } catch {
        return false;
    }
}

async function ensureBinary() {
    // Stage 1: Check bundled or previously compiled binary in local package bin/
    if (isValidBinary(localBinaryPath)) {
        return localBinaryPath;
    }

    // Stage 2: Check global user cache
    if (isValidBinary(cachedBinaryPath)) {
        return cachedBinaryPath;
    }

    // Stage 3: Attempt downloading prebuilt release from GitHub
    const releaseUrl = `https://github.com/Vruket-Developer/Pogonian/releases/download/v${VERSION}/${binaryFilename}`;
    try {
        if (!fs.existsSync(cacheDir)) {
            fs.mkdirSync(cacheDir, { recursive: true });
        }

        console.log(`\x1b[36m[pogonian]\x1b[0m Downloading native binary for ${platform}/${arch} (v${VERSION})...`);
        await downloadFile(releaseUrl, cachedBinaryPath);

        if (isValidBinary(cachedBinaryPath)) {
            console.log(`\x1b[32m[pogonian]\x1b[0m Successfully downloaded and verified ${binaryFilename}`);
            return cachedBinaryPath;
        }
    } catch (downloadErr) {
        // Download failed or release not yet attached - proceed to fallback
    }

    // Stage 4: Fallback to local Go build if Go toolchain is installed
    if (hasLocalGo()) {
        console.log(`\x1b[36m[pogonian]\x1b[0m Compiling Pogonian binary from source with Go...`);
        const buildRes = spawnSync('go', ['build', '-o', localBinaryPath, './cmd/pogn'], {
            cwd: rootDir,
            stdio: 'inherit',
        });
        if (buildRes.status === 0 && isValidBinary(localBinaryPath)) {
            return localBinaryPath;
        }
    }

    // Stage 5: Both prebuilt download and local Go build failed
    console.error(`\n\x1b[1;31m[pogonian] Error:\x1b[0m Unable to find or download a native Pogonian binary for ${platform}/${arch}.`);
    console.error(`\nPlease ensure one of the following:`);
    console.error(`  1. Install the Go toolchain (v1.22+): https://go.dev/dl`);
    console.error(`  2. Download the prebuilt '${binaryFilename}' from:`);
    console.error(`     https://github.com/Vruket-Developer/Pogonian/releases/tag/v${VERSION}`);
    console.error(`     and place it at: ${cachedBinaryPath}\n`);
    process.exit(1);
}

// Run dispatcher
(async () => {
    try {
        const executable = await ensureBinary();
        const result = spawnSync(executable, process.argv.slice(2), {
            stdio: 'inherit',
        });
        process.exit(result.status !== null ? result.status : (result.error ? 1 : 0));
    } catch (err) {
        console.error(`\x1b[1;31m[pogonian] Fatal launcher error:\x1b[0m`, err);
        process.exit(1);
    }
})();
