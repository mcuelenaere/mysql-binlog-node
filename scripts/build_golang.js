const { spawnSync } = require('child_process');
const path = require('path');

function run(...args) {
    const { error, status } = spawnSync(...args);
    if (error !== undefined) {
        throw error;
    } else if (status !== 0) {
        throw new Error(`process exited with status code ${status}`);
    }
}

const combinations = [
    ['linux', 'amd64', 'linux-x64'],
//    ['linux', 'arm', 'linux-arm'], // 3rd-party library does not support 32-bit
    ['linux', 'arm64', 'linux-arm64'],
    ['darwin', 'amd64', 'darwin-x64'],
    ['darwin', 'arm64', 'darwin-arm64'],
    ['windows', 'amd64', 'win32-x64'],
//    ['windows', '386', 'win32-x86'], // 3rd-party library does not support 32-bit
//    ['windows', 'arm64', 'win32-arm64'], // UPX does not support this platform
];

for (const [goOs, goArch, nodePlatformArch] of combinations) {
    console.log(`Building ${nodePlatformArch}...`);
    let outputFile = path.join(__dirname, '..', 'prebuilds', nodePlatformArch);
    if (nodePlatformArch.startsWith('win32')) {
        outputFile += '.exe';
    }

    run(
        'go', ['build', '-ldflags' , '-s -w', '-o', outputFile, '.'], {
            cwd: path.join(__dirname, '..', 'golang'),
            env: {
                ...process.env,
                'GOOS': goOs,
                'GOARCH': goArch,
            },
            stdio: 'inherit',
        }
    );
    run('upx', [outputFile]);
}
