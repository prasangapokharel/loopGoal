#!/usr/bin/env node

/**
 * LoopGoal CLI & NPX Runner
 */

const { spawn, execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

const args = process.argv.slice(2);
const command = args[0] || 'help';

// If running 'install' or 'setup', run the installer script
if (command === 'install' || command === 'setup' || command === 'plugins') {
  require('./install.js');
  process.exit(0);
}

// Locate native Go binary
function findBinary() {
  const home = os.homedir();
  const candidates = [
    'loopgoal',
    path.join(home, 'go', 'bin', 'loopgoal'),
    path.join('/usr', 'local', 'bin', 'loopgoal'),
    path.join(home, '.local', 'bin', 'loopgoal'),
    path.resolve(__dirname, '..', 'loopgoal')
  ];

  for (const cand of candidates) {
    try {
      execSync(`which "${cand}" 2>/dev/null || stat "${cand}" 2>/dev/null`, { stdio: 'ignore' });
      return cand;
    } catch {
      // Continue searching
    }
  }
  return null;
}

const binary = findBinary();

if (binary) {
  // Delegate directly to native Go CLI
  const child = spawn(binary, args, { stdio: 'inherit' });
  child.on('exit', (code) => process.exit(code || 0));
} else {
  // Binary not found: run install or show instructions
  if (command === 'help' || command === '--help' || command === '-h') {
    console.log(`
\x1b[36mLoopGoal — Autonomous Development Loop for AI Coding Agents\x1b[0m

Usage:
  npx loopgoal install       Install skills & rules for Antigravity, Claude, Codex, Cursor
  npx loopgoal init          Initialize .loopgoal configuration in current repository
  npx loopgoal run [goal]    Execute autonomous development loop
  npx loopgoal status        Check loop progress and state
  npx loopgoal verify        Run configured verification checks
  npx loopgoal stop          Halt running loop

Quick setup:
  \x1b[32mnpx loopgoal install\x1b[0m
`);
    process.exit(0);
  }

  console.log(`\x1b[33mNative 'loopgoal' binary not found. Running installer first...\x1b[0m\n`);
  require('./install.js');
}
