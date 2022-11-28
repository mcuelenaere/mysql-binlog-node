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
//    ['linux', '386', 'linux-ia32', true], // 3rd-party library does not support 32-bit
    ['linux', 'amd64', 'linux-x64', true],
//    ['linux', 'arm', 'linux-arm', true], // 3rd-party library does not support 32-bit
    ['linux', 'arm64', 'linux-arm64', true],
    ['darwin', 'amd64', 'darwin-x64', true],
    ['darwin', 'arm64', 'darwin-arm64', false],
    ['windows', 'amd64', 'win32-x64', true],
//    ['windows', '386', 'win32-ia32', true], // 3rd-party library does not support 32-bit
    ['windows', 'arm64', 'win32-arm64', false],
];

for (const [goOs, goArch, nodePlatformArch, enableUpxCompression] of combinations) {
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
    if (enableUpxCompression) {
        run('upx', [outputFile]);
    }
}
