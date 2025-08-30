#!/usr/bin/env node

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

console.log('Preparing OrchCLI package...');

const projectRoot = path.join(__dirname, '..', '..');
const binDir = path.join(__dirname, '..', 'bin');

// Ensure bin directory exists
if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

// Don't build binary in prepack - postinstall will handle it
console.log('Skipping binary build - postinstall will handle it');

console.log('Package preparation complete!');