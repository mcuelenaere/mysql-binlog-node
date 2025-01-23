const { spawnSync } = require('child_process');
const path = require('path');
const fs = require('fs');

function run(...args) {
    const { error, status } = spawnSync(...args);
    if (error !== undefined) {
        throw error;
    } else if (status !== 0) {
        throw new Error(`process exited with status code ${status}`);
    }
}

function discoverGoVersion() {
    const goModPath = path.join(__dirname, '..', 'golang', 'go.mod');
    const goMod = fs.readFileSync(goModPath, 'utf8');
    const goVersionMatch = goMod.match(/^go\s+(\d+\.\d+)/m);
    if (goVersionMatch === null) {
        throw new Error('failed to discover Go version');
    }
    return goVersionMatch[1];
}

const combinations = [
//    ['linux', '386', 'linux-ia32'], // 3rd-party library does not support 32-bit
    ['linux', 'amd64', 'linux-x64'],
//    ['linux', 'arm', 'linux-arm'], // 3rd-party library does not support 32-bit
    ['linux', 'arm64', 'linux-arm64'],
    ['darwin', 'amd64', 'darwin-x64'],
    ['darwin', 'arm64', 'darwin-arm64'],
    ['windows', 'amd64', 'win32-x64'],
//    ['windows', '386', 'win32-ia32'], // 3rd-party library does not support 32-bit
    ['windows', 'arm64', 'win32-arm64'],
];

let commands = [
    'echo "Downloading go dependencies..."',
    'go mod download',
];
for (const [goOs, goArch, nodePlatformArch] of combinations) {
    let outputName = nodePlatformArch;
    if (nodePlatformArch.startsWith('win32')) {
        outputName += '.exe';
    }

    commands.push(`echo "Building ${nodePlatformArch}..."`);
    commands.push(`env GOOS=${goOs} GOARCH=${goArch} go build -ldflags "-s -w" -o /build/${outputName} .`);
}

const goVersion = discoverGoVersion();

run(
    'docker', [
        'run', '--rm', '-t',
        '-v', `${path.join(__dirname, '..', 'prebuilds')}:/build`,
        '-v', `${path.join(__dirname, '..', 'golang')}:/go/src:ro`,
        '-w', '/go/src',
        `golang:${goVersion}`,
        '/bin/sh', '-c', commands.join(' && ')
    ], {
        cwd: path.join(__dirname, '..', 'golang'),
        stdio: 'inherit',
    }
);
