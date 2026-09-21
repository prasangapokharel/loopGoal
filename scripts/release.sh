#!/usr/bin/env bash
# ==============================================================================
# LoopGoal Maintainer Release Helper
# Validates tests, checks packaging, tags releases, and publishes to NPM
# ==============================================================================

set -e

CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}=== LoopGoal Production Release Helper ===${NC}\n"

# 1. Check for uncommitted changes
if [ -n "$(git status --porcelain)" ] && [ "$1" != "--check" ]; then
    echo -e "${YELLOW}Notice: Working tree has uncommitted changes:${NC}"
    git status -s
    echo ""
fi

# 2. Run test suite
echo -e "${YELLOW}Running Go test suite...${NC}"
go test ./...
echo -e " ${GREEN}✓${NC} All Go tests passed."

# 3. Read current version
PKG_VERSION=$(node -p "require('./package.json').version")
echo -e "\nCurrent LoopGoal version in package.json: ${CYAN}v${PKG_VERSION}${NC}"

if [ "$1" == "--check" ]; then
    echo -e "\n${YELLOW}Testing NPM package contents...${NC}"
    npm pack --dry-run
    echo -e "\n${GREEN}✓ Release pre-flight check passed! Ready for production release.${NC}"
    exit 0
fi

read -p "Enter release version (or press Enter to keep v${PKG_VERSION}): " NEW_VERSION
if [ -n "$NEW_VERSION" ]; then
    NEW_VERSION="${NEW_VERSION#v}"
    echo -e "Updating version to ${CYAN}v${NEW_VERSION}${NC}..."
    npm version "$NEW_VERSION" --no-git-tag-version
    PKG_VERSION="$NEW_VERSION"
fi

TAG="v${PKG_VERSION}"

# 4. Verify NPM packaging
echo -e "\n${YELLOW}Verifying NPM package contents...${NC}"
npm pack --dry-run
echo -e " ${GREEN}✓${NC} NPM package tarball verified."

# 5. Output commands
echo -e "\n${GREEN}=== Release Ready ===${NC}"
echo -e "1. Push to GitHub to trigger automated multi-platform binary builds:"
echo -e "   ${CYAN}git add .${NC}"
echo -e "   ${CYAN}git commit -m \"chore(release): release ${TAG}\"${NC}"
echo -e "   ${CYAN}git tag -a ${TAG} -m \"Release ${TAG}\"${NC}"
echo -e "   ${CYAN}git push origin main --tags${NC}"
echo ""
echo -e "2. Publish the NPM package:"
echo -e "   ${CYAN}npm publish --access public${NC}"
echo ""
echo -e "3. Skills.sh instantly detects ${CYAN}${TAG}${NC} via GitHub:"
echo -e "   ${CYAN}npx skills add prasangapokharel/loopGoal${NC}"
