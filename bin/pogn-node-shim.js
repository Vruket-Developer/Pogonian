#!/usr/bin/env node

const { spawnSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const rootDir = path.resolve(__dirname, '..');
const binaryPath = path.join(__dirname, process.platform === 'win32' ? 'pogn.exe' : 'pogn');

// Automatically compile Go binary if not yet built
if (!fs.existsSync(binaryPath)) {
    const buildRes = spawnSync('go', ['build', '-o', binaryPath, './cmd/pogn'], {
        cwd: rootDir,
        stdio: 'inherit',
    });
    if (buildRes.status !== 0) {
        console.error('Failed to compile Pogonian CLI binary.');
        process.exit(buildRes.status || 1);
    }
}

// Execute the Pogonian native binary forwarding all CLI arguments and stdin/stdout/stderr
const result = spawnSync(binaryPath, process.argv.slice(2), {
    stdio: 'inherit',
});

process.exit(result.status !== null ? result.status : (result.error ? 1 : 0));
