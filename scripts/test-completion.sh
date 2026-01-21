#!/usr/bin/env bash
#
# test-completion.sh - Sets up and launches a test environment for zsh completion testing
#
# Usage: ./scripts/test-completion.sh
#
# Creates test projects in .cache/autocomplete-test and launches an isolated zsh
# session with proj completion configured.

set -o errexit
set -o nounset
set -o pipefail

readonly PROJ_SOURCE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly TEST_ROOT="${PROJ_SOURCE}/.cache/autocomplete-test"
readonly TEST_CODE_ROOT="${TEST_ROOT}/code"
readonly TEST_WORKSPACE_ROOT="${TEST_CODE_ROOT}/.workspace"
readonly ZDOTDIR="${TEST_ROOT}/.zsh"

setup_test_data() {
  echo "Setting up test data..."
  mkdir -p "${ZDOTDIR}" "${TEST_CODE_ROOT}" "${TEST_WORKSPACE_ROOT}"

  local proj
  for proj in gfanton/test-project gfanton/another-project example/demo-app; do
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
  done

  (
    cd "${TEST_CODE_ROOT}/gfanton/test-project" \
      && git branch feature-branch \
      && git branch fix-bug \
      && mkdir -p "${TEST_WORKSPACE_ROOT}/gfanton/test-project" \
      && git worktree add -q "${TEST_WORKSPACE_ROOT}/gfanton/test-project/feature-branch" feature-branch \
      && git worktree add -q "${TEST_WORKSPACE_ROOT}/gfanton/test-project/fix-bug" fix-bug
  )
}

build_proj() {
  echo "Building proj..."
  (cd "${PROJ_SOURCE}" && go build -o ./build/proj ./cmd/proj)
}

setup_zshrc() {
  mkdir -p "${ZDOTDIR}"
  cat > "${ZDOTDIR}/.zshrc" << ZSHRC
autoload -Uz compinit && compinit -d "${ZDOTDIR}/.zcompdump"
zstyle ':completion:*' menu select
PS1='[autocomplete-test] %~ %# '
eval "\$(${PROJ_SOURCE}/build/proj init zsh)"
ZSHRC
}

main() {
  if [[ ! -d "${TEST_CODE_ROOT}" ]]; then
    setup_test_data
  fi

  build_proj
  setup_zshrc

  export ZDOTDIR
  export PROJECT_ROOT="${TEST_CODE_ROOT}"
  export HOME="${TEST_ROOT}"

  echo "Starting test zsh..."
  exec zsh
}

main "$@"
