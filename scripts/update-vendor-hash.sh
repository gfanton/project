#!/usr/bin/env bash
#
# Update vendorHash in flake.nix with the correct hash
#
# Usage: ./scripts/update-vendor-hash.sh [-h|--help]
#

set -o errexit
set -o nounset
set -o pipefail

# ---- Constants
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

# ---- Functions

err() {
    printf '%b\n' "${RED}$*${NC}" >&2
}

info() {
    printf '%b\n' "${GREEN}$*${NC}"
}

warn() {
    printf '%b\n' "${YELLOW}$*${NC}"
}

usage() {
    cat <<EOF
Usage: ${0##*/} [OPTIONS]

Update vendorHash in flake.nix with the correct hash.

Options:
    -h, --help    Show this help message
EOF
}

calculate_vendor_hash() {
    local build_output
    local backup_file="flake.nix.bak"

    # Ensure cleanup on any exit from function
    trap 'if [[ -f "${backup_file}" ]]; then mv "${backup_file}" flake.nix; fi' RETURN

    # Create a temporary copy of flake.nix with fakeHash
    cp flake.nix "${backup_file}"

    # Replace vendor hash with fakeHash
    if [[ "${OSTYPE}" == "darwin"* ]]; then
        sed -i '' 's|vendorHash = "[^"]*"|vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="|g' flake.nix
    else
        sed -i 's|vendorHash = "[^"]*"|vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="|g' flake.nix
    fi

    # Build will fail but give us the correct hash
    build_output="$(nix build . --no-link 2>&1 || true)"

    # Restore original flake.nix (handled by trap, but explicit for clarity)
    mv "${backup_file}" flake.nix
    trap - RETURN

    # Extract the correct hash
    if echo "${build_output}" | grep -q "got:"; then
        echo "${build_output}" | grep "got:" | sed 's/.*got: *//' | tail -1
    else
        return 1
    fi
}

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--help)
                usage
                exit 0
                ;;
            *)
                err "Unknown option: $1"
                usage >&2
                exit 1
                ;;
        esac
    done

    local current_hash
    current_hash="$(grep -m1 'vendorHash = ' flake.nix | sed 's/.*vendorHash = "\([^"]*\)".*/\1/')"
    printf '%s\n' "Current vendor hash: ${current_hash}"

    # Determine timeout command (GNU coreutils timeout vs macOS gtimeout)
    local timeout_cmd=""
    if command -v timeout >/dev/null 2>&1; then
        timeout_cmd="timeout 30s"
    elif command -v gtimeout >/dev/null 2>&1; then
        timeout_cmd="gtimeout 30s"
    fi

    # Try to build with current hash first
    printf '%s\n' "Testing current vendor hash..."
    if ${timeout_cmd} nix build .#project --no-link >/dev/null 2>&1; then
        info "Current vendor hash is valid, no update needed."
        exit 0
    fi

    # Current hash is invalid, calculate the correct one
    warn "Current vendor hash invalid, calculating correct hash..."

    local vendor_hash
    vendor_hash="$(calculate_vendor_hash)" || true
    if [[ -z "${vendor_hash}" ]]; then
        err "Failed to calculate vendor hash"
        printf '%s\n' "Please ensure Go dependencies are accessible and try again" >&2
        exit 1
    fi

    printf '%s\n' "New vendor hash: ${vendor_hash}"

    # Update vendor hash in flake.nix
    if [[ "${OSTYPE}" == "darwin"* ]]; then
        sed -i '' "s|vendorHash = .*|vendorHash = \"${vendor_hash}\";|" flake.nix
    else
        sed -i "s|vendorHash = .*|vendorHash = \"${vendor_hash}\";|" flake.nix
    fi

    info "Updated vendorHash in flake.nix"
    printf '%s\n' ""
    printf '%s\n' "To verify, run: nix build --no-link"
}

main "$@"
