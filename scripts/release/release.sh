#!/usr/bin/env bash
set -euo pipefail


# Use go run for gum
GUM_VERSION=v0.16.2

# Check if go is available
if ! command -v go &> /dev/null; then
    echo "Error: Go is required to run this script"
    echo "Please install Go from https://go.dev"
    exit 1
fi

# Function to run gum with go run
gum() {
    go run github.com/charmbracelet/gum@$GUM_VERSION "$@"
}

# Release script for project
# Usage: ./scripts/release.sh [version] [--dry-run]
# Example: ./scripts/release.sh
# Example: ./scripts/release.sh v1.2.3
# Example: ./scripts/release.sh --dry-run

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Parse arguments
VERSION=""
DRY_RUN=false

for arg in "$@"; do
    if [ "$arg" = "--dry-run" ]; then
        DRY_RUN=true
        echo -e "${YELLOW}DRY RUN MODE - No changes will be made${NC}"
    else
        VERSION="$arg"
    fi
done

# Get current version from flake.nix
CURRENT_VERSION=$(grep -m1 'version = "' flake.nix | sed 's/.*version = "\([^"]*\)".*/\1/')

# If no version provided, prompt for it using gum
if [ -z "$VERSION" ]; then
    # Calculate version suggestions
    IFS='.' read -r major minor patch <<< "$CURRENT_VERSION"
    NEXT_MAJOR="v$((major + 1)).0.0"
    NEXT_MINOR="v${major}.$((minor + 1)).0"
    NEXT_PATCH="v${major}.${minor}.$((patch + 1))"

    # Show current version
    gum style \
        --foreground 99 --bold \
        "Current version: v${CURRENT_VERSION}"
    echo ""

    # Let user choose version bump type (with TTY check)
    if [ -t 0 ] && [ -t 1 ]; then
        BUMP_TYPE=$(gum choose \
            --header "Select version bump type:" \
            "Patch ($NEXT_PATCH) - Bug fixes" \
            "Minor ($NEXT_MINOR) - New features" \
            "Major ($NEXT_MAJOR) - Breaking changes" \
            "Custom - Enter manually")
    else
        # No TTY, default to patch
        echo "No interactive terminal, defaulting to patch version"
        BUMP_TYPE="Patch ($NEXT_PATCH) - Bug fixes"
    fi

    case "$BUMP_TYPE" in
        *"Patch"*) VERSION="$NEXT_PATCH" ;;
        *"Minor"*) VERSION="$NEXT_MINOR" ;;
        *"Major"*) VERSION="$NEXT_MAJOR" ;;
        *"Custom"*)
            VERSION=$(gum input \
                --placeholder "v0.0.0" \
                --prompt "Enter custom version: " \
                --value "v")
            ;;
    esac
fi

# Validate version format (must start with 'v' followed by semantic versioning)
if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}Error: Version must be in format vX.Y.Z (e.g., v1.2.3)${NC}"
    exit 1
fi

# Display nice header
gum style \
    --foreground 212 --border-foreground 212 --border double \
    --align center --width 60 --margin "1 2" --padding "2 4" \
    "🚀 Release Script" "" \
    "$(gum style --foreground 99 "Version: ${VERSION}")"

echo ""


# Check if we're on master branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "master" ]; then
    gum style --foreground 214 --bold \
        "⚠️  Warning: Not on master branch (current: $CURRENT_BRANCH)"
    echo ""
    if ! gum confirm "Continue anyway?"; then
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

gum format "# Step 1: Updating vendor hash"
echo ""

# Calculate real vendor hash
CURRENT_HASH=$(grep -m1 'vendorHash = ' flake.nix | sed 's/.*vendorHash = "\([^"]*\)".*/\1/')

# Function to calculate vendor hash
calculate_vendor_hash() {
    # Try to build with current hash first
    if nix build . --no-link 2>/dev/null; then
        echo "$CURRENT_HASH"
        return 0
    fi

    # Create a temporary copy of flake.nix with fakeHash
    cp flake.nix flake.nix.bak

    # Replace vendor hash with fakeHash
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's|vendorHash = "[^"]*"|vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="|g' flake.nix
    else
        sed -i 's|vendorHash = "[^"]*"|vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="|g' flake.nix
    fi

    # Build will fail but give us the correct hash
    BUILD_OUTPUT=$(nix build . --no-link 2>&1 || true)

    # Restore original flake.nix
    mv flake.nix.bak flake.nix

    # Extract the correct hash
    if echo "$BUILD_OUTPUT" | grep -q "got:"; then
        echo "$BUILD_OUTPUT" | grep "got:" | sed 's/.*got: *//' | tail -1
    else
        return 1
    fi
}

# Calculate vendor hash
gum style --foreground 99 "🔍 Checking vendor hash..."

# Try current hash first
if nix build . --no-link >/dev/null 2>&1; then
    VENDOR_HASH="$CURRENT_HASH"
    gum style --foreground 46 "✓ Current vendor hash is valid"
else
    gum style --foreground 99 "⚡ Calculating new vendor hash..."

    # Calculate vendor hash by temporarily using fake hash
    VENDOR_HASH=$(calculate_vendor_hash)
    if [ -z "$VENDOR_HASH" ]; then
        gum style --foreground 196 "❌ Failed to calculate vendor hash"
        exit 1
    fi

    gum style --foreground 214 "📦 New vendor hash: $(gum style --foreground 99 --bold "$VENDOR_HASH")"
fi


# Update vendor hash in flake.nix
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

echo ""
gum format "# Step 2: Updating version in flake.nix"
echo ""

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

echo ""
gum format "# Step 3: Testing build"
echo ""

# Test that the build works
if [ "$DRY_RUN" = true ]; then
    gum style --foreground 214 "Would test nix build (skipping in dry-run mode)"
else
    # Function to test build
    test_build() {
        if nix build --no-link 2>/dev/null; then
            echo "success"
        else
            echo "failed"
        fi
    }

    BUILD_RESULT=$(gum spin \
        --spinner dot \
        --title "Testing nix build..." \
        --show-output \
        -- bash -c "$(declare -f test_build); test_build")

    if [ "$BUILD_RESULT" != "success" ]; then
        gum style --foreground 196 "❌ Build failed"
        echo "Reverting changes..."
        git checkout -- flake.nix
        exit 1
    fi

    gum style --foreground 46 "✓ Build successful!"
fi

echo ""
gum format "# Step 4: Creating release commit"
echo ""

# Default commit message
DEFAULT_COMMIT_MSG="chore: release ${VERSION}

- Update vendor hash to real value
- Update version in flake.nix"

# Stage changes
if [ "$DRY_RUN" = true ]; then
    echo "Would stage: flake.nix"
    echo "Would create commit with message:"
    echo "$DEFAULT_COMMIT_MSG"
else
    git add flake.nix

    # Use default commit message automatically
    COMMIT_MSG="$DEFAULT_COMMIT_MSG"

    # Create commit
    git commit -m "$COMMIT_MSG"
fi

echo ""
gum format "# Step 5: Creating tag"
echo ""

# Create annotated tag
if [ "$DRY_RUN" = true ]; then
    gum style --foreground 214 "Would create tag: $VERSION"
else
    # Create tag with default message
    TAG_MSG="Release ${VERSION}"
    git tag -a "$VERSION" -m "$TAG_MSG"
    gum style --foreground 46 "✓ Created tag: $VERSION"
fi

# Show release summary
if [ "$DRY_RUN" = true ]; then
    echo ""
    gum style --foreground 214 "Would create release $VERSION (dry run completed)"
else
    echo ""
    gum style \
        --border normal \
        --border-foreground 99 \
        --padding "1 2" \
        --width 60 \
        "$(gum format "**📋 Release Summary**")

$(gum style --foreground 245 "Version:  $(gum style --foreground 99 --bold "$VERSION")
Branch:   $CURRENT_BRANCH
Commit:   $(git log -1 --format=%h)
Tag:      $VERSION

Ready to push:
  • git push origin $CURRENT_BRANCH
  • git push origin $VERSION")"
fi

# Success message
echo ""
if [ "$DRY_RUN" = true ]; then
    gum style \
        --foreground 214 --border double --border-foreground 214 \
        --align center --width 60 --padding "1 2" \
        "✅ Dry run completed successfully!"
else
    gum style \
        --foreground 46 --border double --border-foreground 46 \
        --align center --width 60 --padding "1 2" \
        "✅ Release ${VERSION} prepared successfully!"

    echo ""
    gum format "**To complete the release:**
1. Review the changes: \`git show HEAD\`
2. Push the commit: \`git push origin $CURRENT_BRANCH\`
3. Push the tag: \`git push origin $VERSION\`
4. GitHub Actions will automatically build and release binaries"
fi
