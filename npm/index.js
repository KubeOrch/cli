const path = require('path');
const os = require('os');

const ext = os.platform() === 'win32' ? '.exe' : '';
const binPath = path.join(__dirname, 'bin', `orchcli-bin${ext}`);

module.exports = { binPath };
