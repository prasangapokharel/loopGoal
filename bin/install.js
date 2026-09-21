#!/usr/bin/env node

/**
 * LoopGoal Universal Plugin Installer
 * Installs LoopGoal skills and always-on enforcer rules across:
 * - Google Antigravity & Gemini (~/.gemini/config/)
 * - Claude Code (~/.claude/)
 * - OpenAI Codex (~/.codex/)
 * - Cursor (.cursor/rules/)
 */

const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

const HOME = os.homedir();
const ROOT_DIR = path.resolve(__dirname, '..');

// ASCII Banner
console.log(`
\x1b[36m  _                               _ 
 | |                             | |
 | | ___   ___  _ __   __ _  ___ | |
 | |/ _ \\ / _ \\| '_ \\ / _\` |/ _ \\| |
 | | (_) | (_) | |_) | (_| | (_) | |
 |_|\\___/ \\___/| .__/ \\__, |\\___/|_|
               | |     __/ |        
               |_|    |___/         \x1b[0m
 \x1b[1mLoopGoal\x1b[0m — Autonomous Development Loop for AI Coding Agents
`);

function ensureDir(dirPath) {
  if (!fs.existsSync(dirPath)) {
    fs.mkdirSync(dirPath, { recursive: true });
  }
}

function copyFile(src, dest) {
  ensureDir(path.dirname(dest));
  fs.copyFileSync(src, dest);
}

function writeFile(dest, content) {
  ensureDir(path.dirname(dest));
  fs.writeFileSync(dest, content, 'utf8');
}

// 1. Resolve source files
const skillSrc = [
  path.join(ROOT_DIR, 'skills', 'loopgoal', 'SKILL.md'),
  path.join(ROOT_DIR, '.agents', 'skills', 'loopgoal', 'SKILL.md'),
  path.join(ROOT_DIR, 'skill', 'SKILL.md')
].find(p => fs.existsSync(p)) || path.join(ROOT_DIR, 'skills', 'loopgoal', 'SKILL.md');
const ruleSrc = path.join(ROOT_DIR, 'plugins', 'loopgoal', 'rules', 'loopgoal.md');
const claudeSrc = path.join(ROOT_DIR, 'adapters', 'claude', 'CLAUDE.md');
const codexSrc = path.join(ROOT_DIR, 'adapters', 'codex', 'CODEX.md');
const cursorSrc = path.join(ROOT_DIR, 'adapters', 'cursor', 'loopgoal.mdc');
const pluginJsonSrc = path.join(ROOT_DIR, 'plugins', 'loopgoal', 'plugin.json');

let installedCount = 0;

console.log('\x1b[33mScanning agent environments and installing LoopGoal...\x1b[0m\n');

// Target 1: Google Antigravity / Gemini CLI
try {
  const geminiConfig = path.join(HOME, '.gemini', 'config');
  ensureDir(path.join(geminiConfig, 'plugins', 'loopgoal', 'rules'));
  ensureDir(path.join(geminiConfig, 'plugins', 'loopgoal', 'skills', 'loopgoal'));
  ensureDir(path.join(geminiConfig, 'skills', 'loopgoal'));
  ensureDir(path.join(geminiConfig, 'rules'));

  if (fs.existsSync(skillSrc)) {
    copyFile(skillSrc, path.join(geminiConfig, 'skills', 'loopgoal', 'SKILL.md'));
    copyFile(skillSrc, path.join(geminiConfig, 'plugins', 'loopgoal', 'skills', 'loopgoal', 'SKILL.md'));
  }
  if (fs.existsSync(ruleSrc)) {
    copyFile(ruleSrc, path.join(geminiConfig, 'rules', 'loopgoal.md'));
    copyFile(ruleSrc, path.join(geminiConfig, 'plugins', 'loopgoal', 'rules', 'loopgoal.md'));
  }
  if (fs.existsSync(pluginJsonSrc)) {
    copyFile(pluginJsonSrc, path.join(geminiConfig, 'plugins', 'loopgoal', 'plugin.json'));
  }
  console.log(' \x1b[32m✓\x1b[0m Google Antigravity / Gemini plugin installed (~/.gemini/config/)');
  installedCount++;
} catch (err) {
  console.log(` \x1b[31m✗\x1b[0m Antigravity install warning: ${err.message}`);
}

// Target 2: Claude Code
try {
  const claudeDir = path.join(HOME, '.claude');
  const claudeCommands = path.join(claudeDir, 'commands');
  const claudeSkills = path.join(claudeDir, 'skills', 'loopgoal');
  ensureDir(claudeCommands);
  ensureDir(claudeSkills);

  if (fs.existsSync(claudeSrc)) {
    copyFile(claudeSrc, path.join(claudeCommands, 'loopgoal.md'));
  }
  if (fs.existsSync(skillSrc)) {
    copyFile(skillSrc, path.join(claudeSkills, 'SKILL.md'));
  }
  console.log(' \x1b[32m✓\x1b[0m Claude Code skill & /loopgoal command installed (~/.claude/)');
  installedCount++;
} catch (err) {
  console.log(` \x1b[31m✗\x1b[0m Claude Code install warning: ${err.message}`);
}

// Target 3: OpenAI Codex
try {
  const codexDir = path.join(HOME, '.codex');
  const codexSkills = path.join(codexDir, 'skills', 'loopgoal');
  ensureDir(codexSkills);

  if (fs.existsSync(codexSrc)) {
    copyFile(codexSrc, path.join(codexDir, 'CODEX.md'));
  }
  if (fs.existsSync(skillSrc)) {
    copyFile(skillSrc, path.join(codexSkills, 'SKILL.md'));
  }
  console.log(' \x1b[32m✓\x1b[0m OpenAI Codex adapter installed (~/.codex/)');
  installedCount++;
} catch (err) {
  console.log(` \x1b[31m✗\x1b[0m Codex install warning: ${err.message}`);
}

// Target 4: Current Project Workspace (if run in project folder)
const cwd = process.cwd();
if (fs.existsSync(path.join(cwd, '.git')) && cwd !== ROOT_DIR) {
  try {
    const wsAgents = path.join(cwd, '.agents');
    ensureDir(path.join(wsAgents, 'skills', 'loopgoal'));
    ensureDir(path.join(wsAgents, 'rules'));
    ensureDir(path.join(cwd, '.cursor', 'rules'));

    if (fs.existsSync(skillSrc)) {
      copyFile(skillSrc, path.join(wsAgents, 'skills', 'loopgoal', 'SKILL.md'));
    }
    if (fs.existsSync(ruleSrc)) {
      copyFile(ruleSrc, path.join(wsAgents, 'rules', 'loopgoal.md'));
    }
    if (fs.existsSync(cursorSrc)) {
      copyFile(cursorSrc, path.join(cwd, '.cursor', 'rules', 'loopgoal.mdc'));
    }
    console.log(` \x1b[32m✓\x1b[0m Local workspace configured (${cwd})`);
    installedCount++;
  } catch (err) {
    console.log(` \x1b[31m✗\x1b[0m Local workspace warning: ${err.message}`);
  }
}

// Target 5: Go CLI Engine (loopgoal binary)
let goCliInstalled = false;
try {
  execSync('go version', { stdio: 'ignore' });
  const mainGoPath = path.join(ROOT_DIR, 'cmd', 'loopgoal', 'main.go');
  if (fs.existsSync(mainGoPath)) {
    console.log('\n\x1b[33mCompiling and installing native Go loopgoal supervisor...\x1b[0m');
    execSync(`go install ./cmd/loopgoal`, { cwd: ROOT_DIR, stdio: 'inherit' });
    console.log(' \x1b[32m✓\x1b[0m Native Go CLI compiled & installed to ~/go/bin/loopgoal');
    goCliInstalled = true;
  }
} catch {
  // Go compiler not found or build failed; try fetching precompiled release binary
}

if (!goCliInstalled) {
  try {
    const plat = process.platform === 'darwin' ? 'darwin' : (process.platform === 'win32' ? 'windows' : 'linux');
    const arch = process.arch === 'arm64' ? 'arm64' : 'amd64';
    const ext = plat === 'windows' ? 'zip' : 'tar.gz';
    const binExt = plat === 'windows' ? '.exe' : '';
    const releaseUrl = `https://github.com/prasangapokharel/loopGoal/releases/latest/download/loopgoal-${plat}-${arch}.${ext}`;
    const targetDir = path.join(HOME, '.local', 'bin');
    ensureDir(targetDir);
    const targetBin = path.join(targetDir, `loopgoal${binExt}`);

    if (!fs.existsSync(targetBin)) {
      console.log(`\n\x1b[33mGo compiler not detected. Downloading pre-compiled supervisor binary for ${plat}/${arch}...\x1b[0m`);
      const tempArchive = path.join(os.tmpdir(), `loopgoal-${Date.now()}.${ext}`);
      execSync(`curl -fsSL "${releaseUrl}" -o "${tempArchive}"`, { stdio: 'ignore' });
      if (ext === 'tar.gz') {
        execSync(`tar -xzf "${tempArchive}" -C "${targetDir}" loopgoal`, { stdio: 'ignore' });
        fs.chmodSync(targetBin, 0o755);
      }
      try { fs.unlinkSync(tempArchive); } catch {}
      console.log(` \x1b[32m✓\x1b[0m Native Go CLI downloaded & installed to ${targetBin}`);
      goCliInstalled = true;
    } else {
      goCliInstalled = true;
    }
  } catch (dlErr) {
    console.log(` \x1b[33m! Note:\x1b[0m Go CLI build skipped. In-agent skills (/loopgoal) are fully installed and ready!`);
  }
}

console.log(`\n\x1b[32m🎉 LoopGoal installation complete!\x1b[0m`);
console.log(`
\x1b[1mQuick Start:\x1b[0m
  • Inside Antigravity / Gemini / Claude: Type \x1b[36m/loopgoal [your goal]\x1b[0m
  • Inside any terminal project:          Run \x1b[36mloopgoal init\x1b[0m then \x1b[36mloopgoal run\x1b[0m
`);
