#!/usr/bin/env bash
#
# test-tmux.sh - Sets up and launches a test environment for tmux plugin testing
#
# Usage: ./scripts/test-tmux.sh
#
# Creates test projects in .cache/tmux-test and launches an isolated tmux
# session with the proj-tmux plugin loaded.

set -o errexit
set -o nounset
set -o pipefail

readonly PROJ_SOURCE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly TEST_ROOT="${PROJ_SOURCE}/.cache/tmux-test"
readonly TEST_CODE_ROOT="${TEST_ROOT}/code"
readonly TEST_WORKSPACE_ROOT="${TEST_CODE_ROOT}/.workspace"
readonly TMUX_SOCKET="${TEST_ROOT}/tmux.sock"
readonly TMUX_CONF="${TEST_ROOT}/tmux.conf"

setup_test_data() {
  echo "Setting up test data..."
  mkdir -p "${TEST_CODE_ROOT}" "${TEST_WORKSPACE_ROOT}"

  local proj
  for proj in gfanton/test-project gfanton/another-project example/demo-app; do
    if [[ ! -d "${TEST_CODE_ROOT}/${proj}/.git" ]]; then
      mkdir -p "${TEST_CODE_ROOT}/${proj}"
      (
        cd "${TEST_CODE_ROOT}/${proj}" \
          && git init -q \
          && git config user.email "test@test.com" \
          && git config user.name "Test" \
          && echo "# ${proj}" > README.md \
          && git add README.md \
          && git commit -qm "init"
      )
    fi
  done

  # Set up workspaces for test-project
  if [[ ! -d "${TEST_WORKSPACE_ROOT}/gfanton/test-project/feature-branch" ]]; then
    (
      cd "${TEST_CODE_ROOT}/gfanton/test-project" \
        && git branch -f feature-branch 2>/dev/null || true \
        && git branch -f fix-bug 2>/dev/null || true \
        && mkdir -p "${TEST_WORKSPACE_ROOT}/gfanton/test-project" \
        && git worktree add -q "${TEST_WORKSPACE_ROOT}/gfanton/test-project/feature-branch" feature-branch 2>/dev/null || true \
        && git worktree add -q "${TEST_WORKSPACE_ROOT}/gfanton/test-project/fix-bug" fix-bug 2>/dev/null || true
    )
  fi
}

build_binaries() {
  echo "Building proj and proj-tmux..."
  (cd "${PROJ_SOURCE}" && make build-all)
}

setup_tmux_conf() {
  mkdir -p "${TEST_ROOT}"
  cat > "${TMUX_CONF}" << CONF
# Test tmux configuration for proj-tmux plugin

# Set prefix to Ctrl+A for testing (avoid conflict with nested tmux)
set -g prefix C-a
unbind C-b
bind C-a send-prefix

# Basic settings
set -g mouse on
set -g status-style 'bg=blue fg=white'
set -g status-left '[test-tmux] '
set -g status-right '#{pane_current_path}'

# Set environment for proj binaries
set-environment -g PROJ_BIN "${PROJ_SOURCE}/build/proj"
set-environment -g PROJ_TMUX_BIN "${PROJ_SOURCE}/build/proj-tmux"
set-environment -g PROJECT_ROOT "${TEST_CODE_ROOT}"

# Load the plugin
run-shell "${PROJ_SOURCE}/plugins/proj-tmux/plugin/proj-tmux.tmux"
CONF
}

cleanup() {
  if [[ -S "${TMUX_SOCKET}" ]]; then
    tmux -S "${TMUX_SOCKET}" kill-server 2>/dev/null || true
  fi
}

main() {
  trap cleanup EXIT

  setup_test_data
  build_binaries
  setup_tmux_conf

  # Kill any existing test tmux server
  if [[ -S "${TMUX_SOCKET}" ]]; then
    echo "Killing existing test tmux server..."
    tmux -S "${TMUX_SOCKET}" kill-server 2>/dev/null || true
    sleep 0.5
  fi

  echo ""
  echo "Starting isolated tmux session..."
  echo "  Socket: ${TMUX_SOCKET}"
  echo "  Config: ${TMUX_CONF}"
  echo "  Project root: ${TEST_CODE_ROOT}"
  echo ""
  echo "Keybindings (prefix is Ctrl+A):"
  echo "  Ctrl+A Ctrl+P  - Session picker"
  echo "  Ctrl+A Ctrl+W  - Window picker"
  echo "  Ctrl+A W       - Workspace menu"
  echo ""

  export PROJECT_ROOT="${TEST_CODE_ROOT}"
  exec tmux -S "${TMUX_SOCKET}" -f "${TMUX_CONF}" new-session -s test
}

main "$@"
