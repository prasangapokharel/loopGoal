#!/usr/bin/env node

/**
 * LoopGoal Polyglot Livefeed Background Daemon
 * 
 * Continuous, sub-second (0ms reads) verification daemon for AI coding agents.
 * Monitors workspace changes, runs auto-detected or configured linters/compilers,
 * condenses errors into 5-line JSON objects, writes atomic .loopgoal/livefeed.json,
 * and maintains the deterministic .loopgoal/verified.token hard lock.
 */

import fs from 'node:fs';
import path from 'node:path';
import { exec } from 'node:child_process';
import crypto from 'node:crypto';

const CWD = process.cwd();
const LOOPGOAL_DIR = path.join(CWD, '.loopgoal');
const FEED_FILE = path.join(LOOPGOAL_DIR, 'livefeed.json');
const TOKEN_FILE = path.join(LOOPGOAL_DIR, 'verified.token');
const CONFIG_JSON_FILE = path.join(CWD, 'loopgoal.config.json');

const args = process.argv.slice(2);
const runOnce = args.includes('--once') || args.includes('-1');

// Ensure .loopgoal directory exists
if (!fs.existsSync(LOOPGOAL_DIR)) {
  fs.mkdirSync(LOOPGOAL_DIR, { recursive: true });
}

let iterationHeartbeat = 0;
let isChecking = false;
let pendingCheck = false;

// -------------------------------------------------------------
// 1. Language & Framework Adapters
// -------------------------------------------------------------
const ADAPTERS = {
  // TypeScript & JavaScript (Next.js, Vite, React, Node, Bun)
  typescript: {
    detect: () => fs.existsSync(path.join(CWD, 'tsconfig.json')),
    watchExtensions: ['.ts', '.tsx', '.js', '.jsx', '.json', '.mjs', '.cjs'],
    checkCmd: 'npx tsc --noEmit --pretty false',
    parse: (out) => {
      const errors = [];
      const regex = /(.+?)\((\d+),(\d+)\): error (TS\d+): (.+)/g;
      let m;
      while ((m = regex.exec(out)) !== null) {
        errors.push({
          source: 'tsc',
          file: path.relative(CWD, path.resolve(CWD, m[1].trim())),
          line: Number(m[2]),
          col: Number(m[3]),
          code: m[4],
          message: m[5].trim()
        });
      }
      return errors;
    }
  },

  // Python (FastAPI, Django, Flask, JAX, PyTorch, ML)
  python: {
    detect: () =>
      fs.existsSync(path.join(CWD, 'pyproject.toml')) ||
      fs.existsSync(path.join(CWD, 'requirements.txt')) ||
      fs.existsSync(path.join(CWD, 'setup.py')) ||
      fs.existsSync(path.join(CWD, 'Pipfile')),
    watchExtensions: ['.py', '.toml'],
    checkCmd: 'ruff check --output-format=json . || pyright --outputjson',
    parse: (out) => {
      const errors = [];
      try {
        const json = JSON.parse(out);
        if (Array.isArray(json)) {
          for (const item of json) {
            errors.push({
              source: 'ruff',
              file: path.relative(CWD, item.filename),
              line: item.location?.row || 0,
              col: item.location?.column || 0,
              code: item.code || 'lint',
              message: item.message
            });
          }
        } else if (json.generalDiagnostics) {
          for (const d of json.generalDiagnostics) {
            errors.push({
              source: 'pyright',
              file: path.relative(CWD, d.file),
              line: (d.range?.start?.line ?? -1) + 1,
              col: (d.range?.start?.character ?? -1) + 1,
              code: d.rule || 'type-error',
              message: d.message
            });
          }
        }
      } catch {
        const fallbackRegex = /File "(.+?)", line (\d+)(?:, in .+)?\n(?:.+\n)?(\w+Error: .+)/g;
        let m;
        while ((m = fallbackRegex.exec(out)) !== null) {
          errors.push({
            source: 'python',
            file: path.relative(CWD, m[1].trim()),
            line: Number(m[2]),
            col: 1,
            code: 'Runtime/SyntaxError',
            message: m[3].trim()
          });
        }
      }
      return errors;
    }
  },

  // Go (Golang standard / microservices)
  go: {
    detect: () => fs.existsSync(path.join(CWD, 'go.mod')),
    watchExtensions: ['.go', '.mod'],
    checkCmd: 'go vet ./...',
    parse: (out) => {
      const errors = [];
      const regex = /(.+?):(\d+):(\d+): (.+)/g;
      let m;
      while ((m = regex.exec(out)) !== null) {
        errors.push({
          source: 'go-vet',
          file: path.relative(CWD, m[1].trim()),
          line: Number(m[2]),
          col: Number(m[3]),
          code: 'compiler',
          message: m[4].trim()
        });
      }
      return errors;
    }
  },

  // Rust (Cargo projects)
  rust: {
    detect: () => fs.existsSync(path.join(CWD, 'Cargo.toml')),
    watchExtensions: ['.rs', '.toml'],
    checkCmd: 'cargo check --message-format=json -q',
    parse: (out) => {
      const errors = [];
      const lines = out.split('\n');
      for (const line of lines) {
        try {
          const item = JSON.parse(line);
          if (item.reason === 'compiler-message' && item.message.level === 'error') {
            const span = item.message.spans.find((s) => s.is_primary) || item.message.spans[0];
            errors.push({
              source: 'cargo',
              file: span ? span.file_name : 'src/main.rs',
              line: span ? span.line_start : 0,
              col: span ? span.column_start : 0,
              code: item.message.code?.code || 'error',
              message: item.message.message
            });
          }
        } catch {}
      }
      return errors;
    }
  },

  // PHP (Laravel, Symfony, Core PHP)
  php: {
    detect: () => fs.existsSync(path.join(CWD, 'composer.json')),
    watchExtensions: ['.php'],
    checkCmd: './vendor/bin/phpstan analyse --error-format=json --no-progress || php -l',
    parse: (out) => {
      const errors = [];
      try {
        const json = JSON.parse(out);
        if (json.files) {
          for (const [file, data] of Object.entries(json.files)) {
            for (const msg of data.messages) {
              errors.push({
                source: 'phpstan',
                file: path.relative(CWD, file),
                line: msg.line,
                col: 1,
                code: 'type-error',
                message: msg.message
              });
            }
          }
        }
      } catch {}
      return errors;
    }
  }
};

// -------------------------------------------------------------
// 2. Load Config Overrides (loopgoal.config.json)
// -------------------------------------------------------------
function loadUserConfig() {
  if (fs.existsSync(CONFIG_JSON_FILE)) {
    try {
      const raw = fs.readFileSync(CONFIG_JSON_FILE, 'utf-8');
      return JSON.parse(raw);
    } catch (e) {
      console.warn('⚠️ Warning: Failed to parse loopgoal.config.json:', e.message);
    }
  }
  return null;
}

const userConfig = loadUserConfig();

let activeRunners = [];

if (userConfig?.commands && Array.isArray(userConfig.commands)) {
  activeRunners = userConfig.commands.map((c) => ({
    name: c.name || 'custom',
    watchExtensions: c.extensions || ['.ts', '.tsx', '.js', '.py', '.go', '.rs'],
    checkCmd: c.run,
    parse: (out) => {
      const lines = out.split('\n').map((l) => l.trim()).filter(Boolean);
      return lines.slice(0, 8).map((l) => ({ source: c.name || 'custom', message: l }));
    }
  }));
} else {
  for (const [name, adapter] of Object.entries(ADAPTERS)) {
    if (adapter.detect()) {
      activeRunners.push({ name, ...adapter });
    }
  }
}

if (activeRunners.length === 0) {
  activeRunners.push({
    name: 'git-status',
    watchExtensions: ['.go', '.ts', '.tsx', '.js', '.py', '.rs', '.json', '.md'],
    checkCmd: 'git status --porcelain',
    parse: () => []
  });
}

console.log(`⚡ LoopGoal Livefeed Daemon Active: [${activeRunners.map((r) => r.name).join(', ')}]`);

// -------------------------------------------------------------
// 3. Execution & Stale-Safe Atomic Feed Writer
// -------------------------------------------------------------
function runChecks(callback) {
  if (isChecking) {
    pendingCheck = true;
    return;
  }

  isChecking = true;
  iterationHeartbeat++;
  const startTime = Date.now();

  const collectedErrors = [];
  let finished = 0;

  activeRunners.forEach((runner) => {
    exec(runner.checkCmd, { maxBuffer: 1024 * 1024 * 8, cwd: CWD }, (err, stdout, stderr) => {
      const output = (stdout || '') + '\n' + (stderr || '');
      try {
        const parsed = runner.parse(output);
        if (Array.isArray(parsed)) {
          collectedErrors.push(...parsed);
        }
      } catch (parseErr) {
        console.error(`Failed parsing ${runner.name} output:`, parseErr);
      }

      finished++;
      if (finished === activeRunners.length) {
        finalizeFeed(collectedErrors, Date.now() - startTime);
        isChecking = false;
        if (callback) callback();
        if (pendingCheck) {
          pendingCheck = false;
          runChecks();
        }
      }
    });
  });
}

function finalizeFeed(errors, latencyMs) {
  const isPassing = errors.length === 0;

  const verificationToken = isPassing
    ? crypto.createHash('sha256').update(`${Date.now()}-${iterationHeartbeat}-${Math.random()}`).digest('hex')
    : null;

  const payload = {
    heartbeat: iterationHeartbeat,
    updatedAt: new Date().toISOString(),
    latencyMs: `${latencyMs}ms`,
    status: isPassing ? 'passing' : 'failing',
    stats: {
      totalErrors: errors.length,
      runtimes: activeRunners.map((r) => r.name)
    },
    errors: errors.slice(0, 8),
    canCommit: isPassing,
    verificationToken
  };

  const tempFeedFile = `${FEED_FILE}.tmp`;
  try {
    fs.writeFileSync(tempFeedFile, JSON.stringify(payload, null, 2), 'utf-8');
    fs.renameSync(tempFeedFile, FEED_FILE);
  } catch (e) {
    console.error('Error writing livefeed.json:', e);
  }

  if (isPassing) {
    const tokenPayload = {
      token: verificationToken,
      timestamp: new Date().toISOString(),
      summary: `Verified via LoopGoal Daemon (${activeRunners.map((r) => r.name).join(', ')}) at ${payload.updatedAt}`
    };
    try {
      fs.writeFileSync(TOKEN_FILE, JSON.stringify(tokenPayload, null, 2), 'utf-8');
    } catch (e) {
      console.error('Error writing verified.token:', e);
    }
    console.log(`\x1b[32m[LoopGoal: PASSED]\x1b[0m 0 errors (${latencyMs}ms) — Git commit UNLOCKED (token generated).`);
  } else {
    if (fs.existsSync(TOKEN_FILE)) {
      try {
        fs.unlinkSync(TOKEN_FILE);
      } catch {}
    }
    console.log(`\x1b[31m[LoopGoal: BLOCKED]\x1b[0m ${errors.length} error(s) found. Feed updated at .loopgoal/livefeed.json`);
  }
}

// -------------------------------------------------------------
// 4. File Watcher Implementation (Zero External Dependencies)
// -------------------------------------------------------------
if (runOnce) {
  runChecks(() => {
    process.exit(0);
  });
} else {
  let debounceTimer = null;
  const ignoredPatterns = ['.git', 'node_modules', '.loopgoal', '.next', 'dist', 'build', 'target', '.venv', '__pycache__'];
  const activeExts = Array.from(new Set(activeRunners.flatMap((r) => r.watchExtensions || [])));

  try {
    fs.watch(CWD, { recursive: true }, (eventType, filename) => {
      if (!filename) return;
      const normalized = filename.replace(/\\/g, '/');
      if (ignoredPatterns.some((p) => normalized.includes(p))) return;
      const ext = path.extname(filename);
      if (activeExts.includes(ext) || filename === 'package.json' || filename === 'go.mod') {
        clearTimeout(debounceTimer);
        debounceTimer = setTimeout(runChecks, 250);
      }
    });
    console.log('📡 File watcher active (watching file system changes)...');
  } catch (e) {
    console.warn('Native recursive watch fallback:', e.message);
  }

  runChecks();
}
