#!/usr/bin/env bash
set -euo pipefail

# Release helper script for OpsVault.
# Usage: ./scripts/release.sh [patch|minor|major] [--yes]

BUMP_TYPE="${1:-}"
AUTO_YES="${2:-}"

if [[ ! "$BUMP_TYPE" =~ ^(patch|minor|major)$ ]]; then
  echo "Usage: $0 <patch|minor|major> [--yes]"
  exit 1
fi

# 1. Ensure working tree is clean
if [[ -n "$(git status --porcelain)" ]]; then
  echo "Error: Git working directory is not clean. Please commit or stash your changes first."
  exit 1
fi

# 2. Fetch latest tags from remote
echo "-> Fetching latest tags from remote..."
git fetch --tags --quiet || true

# 3. Determine latest version tag via GitHub CLI (gh) first, fallback to git tag
LATEST_TAG=""
if command -v gh >/dev/null 2>&1; then
  LATEST_TAG=$(gh release view --json tagName --jq '.tagName' 2>/dev/null || true)
fi

if [[ -z "$LATEST_TAG" ]]; then
  LATEST_TAG=$(git tag -l "v*" --sort=-v:refname | head -n 1 || true)
fi

if [[ -z "$LATEST_TAG" ]]; then
  LATEST_TAG="v0.0.0"
  echo "-> No existing release tag found. Starting from $LATEST_TAG"
else
  echo "-> Latest release tag: $LATEST_TAG"
fi

# 4. Parse semantic version (vMAJOR.MINOR.PATCH)
VERSION_CLEAN="${LATEST_TAG#v}"
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION_CLEAN"
MAJOR="${MAJOR:-0}"
MINOR="${MINOR:-0}"
PATCH="${PATCH%%[-+]*}"
PATCH="${PATCH:-0}"

case "$BUMP_TYPE" in
  major)
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
    ;;
  minor)
    MINOR=$((MINOR + 1))
    PATCH=0
    ;;
  patch)
    PATCH=$((PATCH + 1))
    ;;
esac

NEW_TAG="v${MAJOR}.${MINOR}.${PATCH}"
echo "-> Next version ($BUMP_TYPE): $NEW_TAG"

# 5. Generate release notes preview from git commits
if git rev-parse "$LATEST_TAG" >/dev/null 2>&1; then
  COMMIT_LOG=$(git log "${LATEST_TAG}..HEAD" --pretty=format:"- %s (%h)" --no-merges)
else
  COMMIT_LOG=$(git log -n 20 --pretty=format:"- %s (%h)" --no-merges)
fi

if [[ -z "$COMMIT_LOG" ]]; then
  COMMIT_LOG="- Maintenance release and build updates"
fi

echo ""
echo "================ Release Notes Preview ($NEW_TAG) ================"
echo "$COMMIT_LOG"
echo "=================================================================="
echo ""

# 6. Confirm before tagging and pushing
if [[ "$AUTO_YES" != "--yes" && "$AUTO_YES" != "-y" ]]; then
  read -r -p "Create and push tag $NEW_TAG to trigger GitHub Actions release? [y/N] " CONFIRM
  if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
    echo "Release cancelled."
    exit 0
  fi
fi

# 7. Create annotated tag and push to origin
git tag -a "$NEW_TAG" -m "Release $NEW_TAG" -m "$COMMIT_LOG"
git push origin "refs/tags/$NEW_TAG"

echo "✅ Tag $NEW_TAG pushed successfully!"
if command -v gh >/dev/null 2>&1; then
  echo "-> You can watch the GitHub Action progress with: gh run watch"
  echo "-> Or view releases with: gh release view $NEW_TAG --web"
fi

