#!/usr/bin/env bash
# TPM entry point for proj tmux plugin

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Load the actual plugin
source "$CURRENT_DIR/plugins/proj-tmux/plugin/proj-tmux.tmux"