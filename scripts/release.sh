#!/usr/bin/env bash
set -euo pipefail

# Release script for project
# Usage: ./scripts/release.sh <version> [--dry-run]
# Example: ./scripts/release.sh v1.2.3
# Example: ./scripts/release.sh v1.2.3 --dry-run

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if version argument is provided
if [ $# -eq 0 ]; then
    echo -e "${RED}Error: Version argument required${NC}"
    echo "Usage: $0 <version> [--dry-run]"
    echo "Example: $0 v1.2.3"
    echo "Example: $0 v1.2.3 --dry-run"
    exit 1
fi

VERSION="$1"
DRY_RUN=false

# Check for dry-run flag
if [ $# -ge 2 ] && [ "$2" = "--dry-run" ]; then
    DRY_RUN=true
    echo -e "${YELLOW}DRY RUN MODE - No changes will be made${NC}"
fi

# Validate version format (must start with 'v' followed by semantic versioning)
if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}Error: Version must be in format vX.Y.Z (e.g., v1.2.3)${NC}"
    exit 1
fi

echo -e "${GREEN}Starting release process for version ${VERSION}${NC}"

# Check if we're on master branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "master" ]; then
    echo -e "${YELLOW}Warning: Not on master branch (current: $CURRENT_BRANCH)${NC}"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Release cancelled"
        exit 1
    fi
fi

# Check for uncommitted changes
if ! git diff --quiet || ! git diff --cached --quiet; then
    echo -e "${RED}Error: You have uncommitted changes${NC}"
    echo "Please commit or stash your changes before releasing"
    exit 1
fi

# Check if tag already exists
if git tag -l | grep -q "^${VERSION}$"; then
    echo -e "${RED}Error: Tag ${VERSION} already exists${NC}"
    exit 1
fi

echo -e "${GREEN}Step 1: Updating vendor hash...${NC}"

# Calculate real vendor hash
echo "Calculating vendor hash..."

# First check if we already have a real vendor hash
CURRENT_HASH=$(grep -m1 'vendorHash = ' flake.nix | sed 's/.*vendorHash = "\([^"]*\)".*/\1/')
echo "Current vendor hash: $CURRENT_HASH"

# Try to build with current hash first
if nix build . --no-link 2>/dev/null; then
    echo "Current vendor hash is valid, keeping it"
    VENDOR_HASH="$CURRENT_HASH"
else
    echo "Current vendor hash is invalid or dependencies changed, calculating new one..."

    # Create a temporary copy of flake.nix with fakeHash
    cp flake.nix flake.nix.bak

    # Replace vendor hash with fakeHash to trigger error with correct hash
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's|vendorHash = "[^"]*"|vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="|g' flake.nix
    else
        sed -i 's|vendorHash = "[^"]*"|vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="|g' flake.nix
    fi

    # Build will fail but give us the correct hash
    BUILD_OUTPUT=$(nix build . --no-link 2>&1 || true)

    # Restore original flake.nix
    mv flake.nix.bak flake.nix

    # Extract the correct hash from error message
    if echo "$BUILD_OUTPUT" | grep -q "got:"; then
        VENDOR_HASH=$(echo "$BUILD_OUTPUT" | grep "got:" | sed 's/.*got: *//' | tail -1)
        echo "Extracted new vendor hash: $VENDOR_HASH"
    else
        echo -e "${RED}Error: Failed to calculate vendor hash${NC}"
        echo "Build output:"
        echo "$BUILD_OUTPUT"
        exit 1
    fi
fi

echo "Vendor hash: $VENDOR_HASH"

# Update vendor hash in flake.nix
echo "Updating flake.nix with real vendor hash..."
if [ "$DRY_RUN" = true ]; then
    echo "Would update vendorHash to: $VENDOR_HASH"
else
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s|vendorHash = .*|vendorHash = \"$VENDOR_HASH\";|" flake.nix
    else
        # Linux
        sed -i "s|vendorHash = .*|vendorHash = \"$VENDOR_HASH\";|" flake.nix
    fi
fi

echo -e "${GREEN}Step 2: Updating version in flake.nix...${NC}"

# Extract version without 'v' prefix
VERSION_NO_V="${VERSION#v}"

# Update version in flake.nix
if [ "$DRY_RUN" = true ]; then
    echo "Would update version to: $VERSION_NO_V"
else
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s|version = \".*\"|version = \"$VERSION_NO_V\"|" flake.nix
    else
        # Linux
        sed -i "s|version = \".*\"|version = \"$VERSION_NO_V\"|" flake.nix
    fi
fi

echo -e "${GREEN}Step 3: Testing build...${NC}"

# Test that the build works with the new vendor hash
echo "Testing nix build..."
if [ "$DRY_RUN" = true ]; then
    echo "Would test nix build (skipping in dry-run mode)"
else
    if ! nix build --no-link 2>/dev/null; then
        echo -e "${RED}Error: Nix build failed with new vendor hash${NC}"
        echo "Reverting changes..."
        git checkout -- flake.nix
        exit 1
    fi
fi

echo -e "${GREEN}Build successful!${NC}"

echo -e "${GREEN}Step 4: Creating release commit...${NC}"

# Stage changes
if [ "$DRY_RUN" = true ]; then
    echo "Would stage: flake.nix"
    echo "Would create commit: chore: release ${VERSION}"
else
    git add flake.nix

    # Create commit
    git commit -m "chore: release ${VERSION}

- Update vendor hash to real value
- Update version in flake.nix"
fi

echo -e "${GREEN}Step 5: Creating tag...${NC}"

# Create annotated tag
if [ "$DRY_RUN" = true ]; then
    echo "Would create tag: $VERSION"
else
    git tag -a "$VERSION" -m "Release ${VERSION}"
fi

echo -e "${GREEN}Step 6: Pushing changes...${NC}"

# Push commit and tag
if [ "$DRY_RUN" = true ]; then
    echo "Would push to: origin $CURRENT_BRANCH"
    echo "Would push tag: $VERSION"
else
    echo "Pushing to remote..."
    git push origin "$CURRENT_BRANCH"
    git push origin "$VERSION"
fi

echo -e "${GREEN}✅ Release ${VERSION} completed successfully!${NC}"
echo ""
echo "Next steps:"
echo "1. GitHub Actions will automatically build and release binaries"
echo "2. Check the Actions tab for build progress"
echo "3. Once complete, the release will be available at:"
echo "   https://github.com/gfanton/project/releases/tag/${VERSION}"
