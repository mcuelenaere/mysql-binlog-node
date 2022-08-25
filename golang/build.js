const { spawnSync } = require("child_process");
const path = require('path');

const destinationFile = path.resolve(process.cwd(), process.argv[2]);
const projectPath = path.resolve(__dirname, '..');

console.log('Building go library, output is ' + destinationFile);

const { status } = spawnSync(
    'go',
    ["build", "-buildmode", "c-archive", "-o", destinationFile],
    {
        cwd: path.join(projectPath, 'golang'),
        stdio: 'inherit',
    }
);
process.exit(status);