#!/usr/bin/env bash
# ==============================================================================
# LoopGoal Universal Installer
# Installs LoopGoal for Antigravity, Gemini, Claude Code, Codex, and Cursor
# ==============================================================================

set -e

CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${CYAN}"
cat << "EOF"
  _                               _ 
 | |                             | |
 | | ___   ___  _ __   __ _  ___ | |
 | |/ _ \ / _ \| '_ \ / _` |/ _ \| |
 | | (_) | (_) | |_) | (_| | (_) | |
 |_|\___/ \___/| .__/ \__, |\___/|_|
               | |     __/ |        
               |_|    |___/         
EOF
echo -e " LoopGoal — Autonomous Development Loop for AI Coding Agents${NC}\n"

HOME_DIR="${HOME}"
REPO_RAW="https://raw.githubusercontent.com/prasangapokharel/loopGoal/main"

# 1. Antigravity & Gemini Global Configuration
echo -e "${YELLOW}Installing Google Antigravity & Gemini plugin...${NC}"
GEMINI_DIR="${HOME_DIR}/.gemini/config"
mkdir -p "${GEMINI_DIR}/plugins/loopgoal/rules"
mkdir -p "${GEMINI_DIR}/plugins/loopgoal/skills/loopgoal"
mkdir -p "${GEMINI_DIR}/skills/loopgoal"
mkdir -p "${GEMINI_DIR}/rules"

curl -fsSL "${REPO_RAW}/.agents/skills/loopgoal/SKILL.md" -o "${GEMINI_DIR}/skills/loopgoal/SKILL.md" 2>/dev/null || true
curl -fsSL "${REPO_RAW}/.agents/skills/loopgoal/SKILL.md" -o "${GEMINI_DIR}/plugins/loopgoal/skills/loopgoal/SKILL.md" 2>/dev/null || true
curl -fsSL "${REPO_RAW}/plugins/loopgoal/rules/loopgoal.md" -o "${GEMINI_DIR}/rules/loopgoal.md" 2>/dev/null || true
curl -fsSL "${REPO_RAW}/plugins/loopgoal/rules/loopgoal.md" -o "${GEMINI_DIR}/plugins/loopgoal/rules/loopgoal.md" 2>/dev/null || true
curl -fsSL "${REPO_RAW}/plugins/loopgoal/plugin.json" -o "${GEMINI_DIR}/plugins/loopgoal/plugin.json" 2>/dev/null || true

echo -e " ${GREEN}✓${NC} Antigravity / Gemini plugin installed to ${GEMINI_DIR}"

# 2. Claude Code
echo -e "${YELLOW}Installing Claude Code skill...${NC}"
CLAUDE_DIR="${HOME_DIR}/.claude"
mkdir -p "${CLAUDE_DIR}/commands"
mkdir -p "${CLAUDE_DIR}/skills/loopgoal"
curl -fsSL "${REPO_RAW}/adapters/claude/CLAUDE.md" -o "${CLAUDE_DIR}/commands/loopgoal.md" 2>/dev/null || true
curl -fsSL "${REPO_RAW}/.agents/skills/loopgoal/SKILL.md" -o "${CLAUDE_DIR}/skills/loopgoal/SKILL.md" 2>/dev/null || true
echo -e " ${GREEN}✓${NC} Claude Code skill installed to ${CLAUDE_DIR}"

# 3. OpenAI Codex
echo -e "${YELLOW}Installing OpenAI Codex adapter...${NC}"
CODEX_DIR="${HOME_DIR}/.codex"
mkdir -p "${CODEX_DIR}/skills/loopgoal"
curl -fsSL "${REPO_RAW}/adapters/codex/CODEX.md" -o "${CODEX_DIR}/CODEX.md" 2>/dev/null || true
curl -fsSL "${REPO_RAW}/.agents/skills/loopgoal/SKILL.md" -o "${CODEX_DIR}/skills/loopgoal/SKILL.md" 2>/dev/null || true
echo -e " ${GREEN}✓${NC} Codex adapter installed to ${CODEX_DIR}"

# 4. If inside a git repository, configure local workspace
if [ -d ".git" ]; then
    echo -e "${YELLOW}Configuring current workspace...${NC}"
    mkdir -p .agents/skills/loopgoal .agents/rules .cursor/rules
    curl -fsSL "${REPO_RAW}/.agents/skills/loopgoal/SKILL.md" -o .agents/skills/loopgoal/SKILL.md 2>/dev/null || true
    curl -fsSL "${REPO_RAW}/plugins/loopgoal/rules/loopgoal.md" -o .agents/rules/loopgoal.md 2>/dev/null || true
    curl -fsSL "${REPO_RAW}/adapters/cursor/loopgoal.mdc" -o .cursor/rules/loopgoal.mdc 2>/dev/null || true
    echo -e " ${GREEN}✓${NC} Workspace initialized with LoopGoal skills and rules"
fi

# 5. Compile and install Go binary if Go exists
if command -v go &> /dev/null; then
    echo -e "${YELLOW}Go detected! Installing native loopgoal CLI...${NC}"
    go install github.com/prasangapokharel/loopGoal/cmd/loopgoal@latest 2>/dev/null || true
    echo -e " ${GREEN}✓${NC} Native CLI installed to ~/go/bin/loopgoal"
fi

echo -e "\n${GREEN}🎉 LoopGoal installed successfully!${NC}"
echo -e "Inside any agent (Antigravity / Gemini / Claude), trigger with: ${CYAN}/loopgoal [your goal]${NC}"
echo -e "In any terminal project, initialize with:             ${CYAN}loopgoal init${NC}"
