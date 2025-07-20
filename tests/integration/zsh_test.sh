#!/usr/bin/env bats

# Test setup
setup() {
  # Create temporary test environment
  export TEST_HOME=$(mktemp -d)
  export PROJECT_ROOT="$TEST_HOME/code"
  export PROJECT_CONFIG="$TEST_HOME/.projectrc"
  
  # Create project binary path
  export PROJECT_BIN="$PWD/build/proj"
  
  # Create test project structure
  mkdir -p "$PROJECT_ROOT/user1/project1"
  mkdir -p "$PROJECT_ROOT/user1/project2"
  mkdir -p "$PROJECT_ROOT/user2/project3"
  
  # Initialize git repos
  (cd "$PROJECT_ROOT/user1/project1" && git init --quiet)
  (cd "$PROJECT_ROOT/user1/project2" && git init --quiet)
  (cd "$PROJECT_ROOT/user2/project3" && git init --quiet)
  
  # Create config file
  cat > "$PROJECT_CONFIG" <<EOF
root = "$PROJECT_ROOT"
user = "testuser"
EOF
  
  # Export config path for project binary
  export PROJECT_CONFIG_FILE="$PROJECT_CONFIG"
  
  # Source the zsh init script
  export ZDOTDIR="$TEST_HOME"
  cat > "$ZDOTDIR/.zshrc" <<'EOF'
# Minimal zsh config for testing
autoload -U compinit && compinit -u
setopt NO_BEEP
EOF
}

teardown() {
  rm -rf "$TEST_HOME"
}

# Test that proj init generates valid zsh code
@test "proj init generates valid zsh code" {
  run "$PROJECT_BIN" init zsh
  [ "$status" -eq 0 ]
  [ -n "$output" ]
  
  # Check that output contains expected functions
  [[ "$output" =~ "__project_p()" ]]
  [[ "$output" =~ "__project_p_complete()" ]]
  [[ "$output" =~ "alias p=" ]]
}

# Test that zsh can source the init script
@test "zsh can source proj init script" {
  # Generate init script
  "$PROJECT_BIN" init zsh > "$TEST_HOME/project.zsh"
  
  # Test sourcing in zsh
  run zsh -c "source $TEST_HOME/project.zsh && type __project_p"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "__project_p is a shell function" ]]
}

# Test basic completion functionality
@test "proj query returns expected results" {
  cd "$TEST_HOME"
  run "$PROJECT_BIN" query "project1"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "user1/project1" ]]
}

# Test fuzzy search
@test "proj query supports fuzzy search" {
  cd "$TEST_HOME"
  run "$PROJECT_BIN" query "proj3"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "user2/project3" ]]
}

# Test exclude current directory
@test "proj query excludes current directory" {
  cd "$PROJECT_ROOT/user1/project1"
  run "$PROJECT_BIN" query --exclude "$PWD" --limit 10 ""
  [ "$status" -eq 0 ]
  # Verify current directory is excluded
  [[ ! "$output" =~ "user1/project1" ]]
  # Verify other projects are included
  [[ "$output" =~ "user1/project2" ]] || [[ "$output" =~ "user2/project3" ]]
}

# Test limit functionality
@test "proj query respects limit" {
  cd "$TEST_HOME"
  run "$PROJECT_BIN" query --limit 2 "project"
  [ "$status" -eq 0 ]
  # Count lines in output
  line_count=$(echo "$output" | wc -l | tr -d ' ')
  [ "$line_count" -eq 2 ]
}

# Test p command integration with zsh
@test "p command works in zsh" {
  # Generate and source init script
  "$PROJECT_BIN" init zsh > "$TEST_HOME/project.zsh"
  
  # Create a test script that uses the p function
  cat > "$TEST_HOME/test_p.zsh" <<'EOF'
source $TEST_HOME/project.zsh
export PROJECT_CONFIG_FILE="$PROJECT_CONFIG"

# Override __project_cd to just echo the path
__project_cd() {
  echo "Would change to: $1"
}

# Test the p function
p project2
EOF
  
  # Set env vars for the script
  export TEST_HOME PROJECT_CONFIG
  run zsh "$TEST_HOME/test_p.zsh"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Would change to: $PROJECT_ROOT/user1/project2" ]]
}

# Test completion generation
@test "zsh completions work" {
  # Generate init script
  "$PROJECT_BIN" init zsh > "$TEST_HOME/project.zsh"
  
  # Test that the init script contains expected content
  [[ "$(cat "$TEST_HOME/project.zsh")" =~ "__project_p_complete" ]]
  [[ "$(cat "$TEST_HOME/project.zsh")" =~ "alias p=__project_p" ]]
  
  # Test sourcing in interactive mode
  run zsh -i -c "
    export PROJECT_CONFIG_FILE='$PROJECT_CONFIG'
    source '$TEST_HOME/project.zsh'
    type __project_p
  "
  [ "$status" -eq 0 ]
}