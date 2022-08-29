const path = require('path');
const fs = require('fs');
const os = require('os');

function nativeGoBinaryName() {
    let suffix = '';
    if (os.platform() === 'win32') {
        suffix = '.exe';
    }
    return `${os.platform()}-${os.arch()}${suffix}`;
}

const PREBUILDS_PATH = path.join(__dirname, '..', 'prebuilds');
const nativeBinaryFilename = nativeGoBinaryName();

if (!fs.existsSync(PREBUILDS_PATH)) {
    console.error('WARNING: directory containing precompiled Go binaries does not exist. Please run "npm run build" first.');
    return;
}

let nativeBinaryFound = false;
for (const entry of fs.readdirSync(PREBUILDS_PATH, { withFileTypes: true })) {
    if (!entry.isFile()) {
        continue;
    }

    if (entry.name !== nativeBinaryFilename) {
        fs.unlinkSync(path.join(PREBUILDS_PATH, entry.name));
    } else {
        nativeBinaryFound = true;
    }
}

if (!nativeBinaryFound) {
    console.error(`WARNING: could not find the precompiled Go binary for your platform (${nativeBinaryFilename})!`);
}