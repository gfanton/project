#!/usr/bin/env bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_info() { echo -e "${BLUE}ℹ ${1}${NC}"; }
print_success() { echo -e "${GREEN}✅ ${1}${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ ${1}${NC}"; }
print_error() { echo -e "${RED}❌ ${1}${NC}"; }

# Help message
show_help() {
    cat << EOF
Usage: $0 <version>

Release script for the project tool.

This script will:
1. Switch to master branch
2. Ensure working directory is clean
3. Run tests to ensure everything works locally  
4. Calculate the correct vendorHash for the Go dependencies
5. Update flake.nix with the real vendorHash
6. Create a release commit with the version
7. Tag the release with the provided version
8. Push the commit and tag to trigger CI

Arguments:
    <version>    Version to release (e.g., v1.2.3, v0.1.0)

Examples:
    $0 v1.2.3
    $0 v0.1.0

Requirements:
    - Git repository with clean working directory
    - Nix installed for vendorHash calculation
    - Make available for running tests
EOF
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to calculate vendorHash
calculate_vendor_hash() {
    print_info "Calculating vendorHash..."
    
    # Create a temporary flake.nix with empty vendorHash to get the actual hash
    local temp_file=$(mktemp)
    cp flake.nix "$temp_file"
    
    # Replace fakeHash with empty string to trigger hash calculation
    sed 's/pkgs\.lib\.fakeHash/""/g' flake.nix > flake.temp.nix
    mv flake.temp.nix flake.nix
    
    # Build to get the correct hash (this will fail but give us the hash)
    local hash_output
    if ! hash_output=$(nix build .#project 2>&1); then
        # Extract the hash from the error message
        local vendor_hash
        vendor_hash=$(echo "$hash_output" | grep -o 'sha256-[A-Za-z0-9+/=]\{44\}' | tail -1)
        
        if [[ -n "$vendor_hash" ]]; then
            print_success "Found vendorHash: $vendor_hash"
            # Restore original flake.nix and then update with correct hash
            cp "$temp_file" flake.nix
            sed -i.bak "s/pkgs\.lib\.fakeHash/\"$vendor_hash\"/g" flake.nix
            rm -f flake.nix.bak
            
            # Verify the build works with correct hash
            print_info "Verifying build with correct vendorHash..."
            if nix build .#project --no-link; then
                print_success "Build verification successful"
                echo "$vendor_hash"
            else
                print_error "Build failed even with calculated hash"
                cp "$temp_file" flake.nix
                rm -f "$temp_file"
                exit 1
            fi
        else
            print_error "Could not extract vendorHash from build output"
            cp "$temp_file" flake.nix
            rm -f "$temp_file"
            exit 1
        fi
    else
        print_error "Expected build to fail for hash calculation, but it succeeded"
        cp "$temp_file" flake.nix
        rm -f "$temp_file"
        exit 1
    fi
    
    rm -f "$temp_file"
}

# Main release function
main() {
    local version="$1"
    
    # Validate version format
    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$ ]]; then
        print_error "Invalid version format. Expected format: vX.Y.Z (e.g., v1.2.3)"
        exit 1
    fi
    
    print_info "Starting release process for $version"
    
    # Check prerequisites
    if ! command_exists git; then
        print_error "Git is required but not installed"
        exit 1
    fi
    
    if ! command_exists nix; then
        print_error "Nix is required but not installed"
        exit 1
    fi
    
    if ! command_exists make; then
        print_error "Make is required but not installed"
        exit 1
    fi
    
    # Ensure we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "Not in a git repository"
        exit 1
    fi
    
    # Check if version tag already exists
    if git rev-parse "$version" >/dev/null 2>&1; then
        print_error "Version tag $version already exists"
        exit 1
    fi
    
    # Switch to master branch
    print_info "Switching to master branch..."
    local current_branch
    current_branch=$(git branch --show-current)
    
    if [[ "$current_branch" != "master" ]]; then
        git checkout master
        print_success "Switched to master branch"
    else
        print_success "Already on master branch"
    fi
    
    # Ensure working directory is clean
    print_info "Checking working directory status..."
    if [[ -n $(git status --porcelain) ]]; then
        print_error "Working directory is not clean. Please commit or stash changes."
        git status --short
        exit 1
    fi
    print_success "Working directory is clean"
    
    # Pull latest changes
    print_info "Pulling latest changes..."
    git pull origin master
    print_success "Updated with latest changes"
    
    # Run tests to ensure everything works
    print_info "Running tests..."
    if make test; then
        print_success "Tests passed"
    else
        print_error "Tests failed. Please fix issues before releasing."
        exit 1
    fi
    
    # Run linting
    print_info "Running linting..."
    if make lint; then
        print_success "Linting passed"
    else
        print_error "Linting failed. Please fix issues before releasing."
        exit 1
    fi
    
    # Calculate and update vendorHash
    local vendor_hash
    vendor_hash=$(calculate_vendor_hash)
    
    # Update version in flake.nix if needed
    print_info "Updating version in flake.nix to $version..."
    local version_number="${version#v}"  # Remove 'v' prefix
    sed -i.bak "s/version = \"[^\"]*\"/version = \"$version_number\"/g" flake.nix
    rm -f flake.nix.bak
    
    # Verify final build works
    print_info "Final build verification..."
    if make build; then
        print_success "Build successful"
    else
        print_error "Final build failed"
        exit 1
    fi
    
    # Create release commit
    print_info "Creating release commit..."
    git add flake.nix
    git commit -m "Release $version

- Update vendorHash to $vendor_hash
- Set version to $version_number"
    
    # Create version tag
    print_info "Creating version tag $version..."
    git tag -a "$version" -m "Release $version"
    
    print_success "Release $version prepared successfully"
    
    # Ask for confirmation before pushing
    echo
    print_warning "Ready to push release $version to origin."
    print_info "This will trigger the CI/CD pipeline to build and publish the release."
    echo
    read -p "Do you want to push now? [y/N]: " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "Pushing commit and tag to origin..."
        git push origin master
        git push origin "$version"
        
        print_success "🎉 Release $version has been pushed successfully!"
        print_info "CI/CD pipeline should start building the release now."
        print_info "Check GitHub Actions: https://github.com/gfanton/project/actions"
    else
        print_warning "Release commit and tag created locally but not pushed."
        print_info "To push later, run:"
        print_info "  git push origin master"
        print_info "  git push origin $version"
        print_info ""
        print_info "To undo the release (if not pushed yet):"
        print_info "  git tag -d $version"
        print_info "  git reset --hard HEAD~1"
    fi
}

# Handle script arguments
if [[ $# -eq 0 ]] || [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
    show_help
    exit 0
fi

if [[ $# -ne 1 ]]; then
    print_error "Exactly one argument (version) is required"
    show_help
    exit 1
fi

main "$1"