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

// Support daemon / watch command even without native Go binary
if (command === 'daemon' || command === 'watch') {
  const daemonScript = path.resolve(__dirname, '..', 'loopgoal-daemon.mjs');
  if (fs.existsSync(daemonScript)) {
    const child = spawn(process.execPath, [daemonScript, ...args.slice(1)], { stdio: 'inherit' });
    child.on('exit', (code) => process.exit(code || 0));
    return;
  }
}

// Support version command even without native binary
if (command === 'version' || command === '--version' || command === '-v') {
  try {
    const pkg = require('../package.json');
    console.log(`loopgoal v${pkg.version} (${process.platform}/${process.arch})`);
    process.exit(0);
  } catch {}
}

// Locate native Go binary across platforms
function findBinary() {
  const home = os.homedir();
  const isWin = process.platform === 'win32';
  const binName = isWin ? 'loopgoal.exe' : 'loopgoal';

  // 1. Check if loopgoal is in PATH
  try {
    const whichCmd = isWin ? `where ${binName}` : `which ${binName}`;
    const out = execSync(whichCmd, { stdio: ['ignore', 'pipe', 'ignore'] }).toString().trim().split('\n')[0].trim();
    if (out && fs.existsSync(out)) return out;
  } catch {}

  // 2. Search common locations
  const candidates = [
    path.join(home, 'go', 'bin', binName),
    path.join('/usr', 'local', 'bin', binName),
    path.join(home, '.local', 'bin', binName),
    path.join(home, '.loopgoal', 'bin', binName),
    path.resolve(__dirname, '..', binName),
    path.resolve(__dirname, binName)
  ];

  if (isWin) {
    const appData = process.env.APPDATA || path.join(home, 'AppData', 'Roaming');
    const localAppData = process.env.LOCALAPPDATA || path.join(home, 'AppData', 'Local');
    candidates.push(
      path.join(appData, 'npm', binName),
      path.join(localAppData, 'Programs', 'loopgoal', binName)
    );
  }

  for (const cand of candidates) {
    if (fs.existsSync(cand)) {
      return cand;
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
  // Binary not found: show help or run installer
  if (command === 'help' || command === '--help' || command === '-h') {
    console.log(`
\x1b[36mLoopGoal — Autonomous Development Loop for AI Coding Agents\x1b[0m

Usage:
  npx loopgoal install       Install skills & rules for Antigravity, Claude, Codex, Cursor
  npx loopgoal init          Initialize .loopgoal configuration in current repository
  npx loopgoal run [goal]    Execute autonomous development loop
  npx loopgoal verify        Execute project verification and generate commit token
  npx loopgoal select <file> Lock a single target file for the current iteration
  npx loopgoal hook install  Install Git hard enforcement hooks (pre-commit, pre-push)
  npx loopgoal rollback      Restore working tree to clean state (discard unverified edits)
  npx loopgoal daemon        Run polyglot background livefeed daemon (.loopgoal/livefeed.json)
  npx loopgoal livefeed      Display livefeed verification status or token
  npx loopgoal mcp           Run Model Context Protocol (MCP) server over stdio
  npx loopgoal scan          Inspect and categorize repository inventory
  npx loopgoal plan          Display current task map, pending queue, and evidence
  npx loopgoal test          Execute pre-flight gate checks
  npx loopgoal status        Check loop progress and state
  npx loopgoal stop          Halt running loop
  npx loopgoal version       Show version

Quick setup:
  \x1b[32mnpx loopgoal install\x1b[0m
`);
    process.exit(0);
  }

  console.log(`\x1b[33mNative 'loopgoal' binary not found. Running installer first...\x1b[0m\n`);
  require('./install.js');
}
