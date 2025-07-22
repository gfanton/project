#!/usr/bin/env bash
set -euo pipefail

# Script to update vendorHash in flake.nix with the correct hash
# This replaces pkgs.lib.fakeHash with the actual SHA256 hash needed for Nix builds

echo "Calculating vendorHash..."

# Create temporary backup of flake.nix
temp_file=$(mktemp)
cp flake.nix "$temp_file"

# Replace fakeHash with empty string to trigger hash mismatch
sed 's/pkgs\.lib\.fakeHash/""/g' flake.nix > flake.temp.nix
mv flake.temp.nix flake.nix

# Try to build and capture the error output containing the expected hash
if ! output=$(nix build .#project 2>&1); then
    # Extract the SHA256 hash from the error message
    vendor_hash=$(echo "$output" | grep -o 'sha256-[A-Za-z0-9+/=]\{44\}' | tail -1)
    
    if [ -n "$vendor_hash" ]; then
        echo "Found vendorHash: $vendor_hash"
        
        # Restore original flake.nix and update with correct hash
        cp "$temp_file" flake.nix
        sed -i.bak "s/pkgs\.lib\.fakeHash/\"$vendor_hash\"/g" flake.nix
        rm -f flake.nix.bak
        
        echo "Updated flake.nix with vendorHash: $vendor_hash"
        
        # Verify the build works with the new hash
        if nix build .#project --no-link; then
            echo "✅ Build verification successful"
        else
            echo "❌ Build failed with calculated hash"
            cp "$temp_file" flake.nix
            rm -f "$temp_file"
            exit 1
        fi
    else
        echo "❌ Could not extract vendorHash from build output"
        echo "Build output:"
        echo "$output" | head -20
        cp "$temp_file" flake.nix
        rm -f "$temp_file"
        exit 1
    fi
else
    echo "❌ Expected build to fail for hash calculation"
    cp "$temp_file" flake.nix
    rm -f "$temp_file"
    exit 1
fi

# Clean up
rm -f "$temp_file"
echo "Done!"