#!/usr/bin/env bats

# Test actual proj-tmux session management functionality

load '../helpers/test_cleanup.sh'

setup() {
    # Initialize isolated environment for testing
    export TEST_TMUX_DIR=$(mktemp -d -t tmux-test-XXXXXX)
    export TEST_TMUX_SOCKET="$TEST_TMUX_DIR/test-socket"
    export TEST_PROJECT_DIR="$TEST_TMUX_DIR/projects"
    export TMUX_TMPDIR="$TEST_TMUX_DIR"
    
    mkdir -p "$TEST_PROJECT_DIR"
    
    # Create test project structure
    mkdir -p "$TEST_PROJECT_DIR/testorg/testproject"
    cd "$TEST_PROJECT_DIR/testorg/testproject"
    git init -q
    git config user.name "Test User"
    git config user.email "test@example.com"
    echo "# Test Project" > README.md
    git add . && git commit -q -m "Initial commit"
    
    # Use absolute path for proj-tmux binary
    PROJ_TMUX_BINARY="/Users/gfanton/code/.workspace/gfanton/project.feat/tmux-integration/build/proj-tmux"
    
    # Build proj-tmux if needed
    if [ ! -f "$PROJ_TMUX_BINARY" ]; then
        cd /Users/gfanton/code/.workspace/gfanton/project.feat/tmux-integration
        make build-tmux
        cd "$TEST_PROJECT_DIR/testorg/testproject"
    fi
    
    export PROJ_TMUX_BINARY
}

teardown() {
    # Cleanup isolated environment
    if [[ -n "${TEST_TMUX_SOCKET:-}" ]]; then
        tmux -S "$TEST_TMUX_SOCKET" kill-server 2>/dev/null || true
    fi
    
    if [[ -n "${TEST_TMUX_DIR:-}" && -d "$TEST_TMUX_DIR" ]]; then
        rm -rf "$TEST_TMUX_DIR"
    fi
}

@test "proj-tmux session create works" {
    cd "$TEST_PROJECT_DIR/testorg/testproject"
    
    # Set tmux socket for proj-tmux to use isolated environment
    export TMUX_SOCKET="$TEST_TMUX_SOCKET"
    
    # Create session without switching (since we're not in tmux)
    run env TMUX="" PROJECT_ROOT="$TEST_PROJECT_DIR" "$PROJ_TMUX_BINARY" session create --switch=false testorg/testproject
    [ "$status" -eq 0 ]
    
    # Verify session was created
    run tmux -S "$TEST_TMUX_SOCKET" list-sessions -F "#{session_name}"
    [ "$status" -eq 0 ]
    [[ "$output" == *"proj-testorg-testproject"* ]]
}

@test "proj-tmux session list shows created sessions" {
    cd "$TEST_PROJECT_DIR/testorg/testproject"
    
    # Set tmux socket for proj-tmux to use isolated environment
    export TMUX_SOCKET="$TEST_TMUX_SOCKET"
    
    # Create session
    env TMUX="" PROJECT_ROOT="$TEST_PROJECT_DIR" "$PROJ_TMUX_BINARY" session create --switch=false testorg/testproject
    
    # List sessions
    run env TMUX="" PROJECT_ROOT="$TEST_PROJECT_DIR" "$PROJ_TMUX_BINARY" session list
    [ "$status" -eq 0 ]
    [[ "$output" == *"proj-testorg-testproject"* ]]
    [[ "$output" == *"testorg/testproject"* ]]
}

@test "proj-tmux session current detects project from directory" {
    cd "$TEST_PROJECT_DIR/testorg/testproject"
    
    # Set tmux socket and root directory for proj-tmux to use isolated environment
    export TMUX_SOCKET="$TEST_TMUX_SOCKET"
    
    run env TMUX="" PROJECT_ROOT="$TEST_PROJECT_DIR" "$PROJ_TMUX_BINARY" session current
    [ "$status" -eq 0 ]
    [[ "$output" == *"testorg/testproject"* ]]
}

@test "session naming handles special characters" {
    mkdir -p "$TEST_PROJECT_DIR/test.org/test.project"
    cd "$TEST_PROJECT_DIR/test.org/test.project"
    git init -q
    
    # Set tmux socket for proj-tmux to use isolated environment
    export TMUX_SOCKET="$TEST_TMUX_SOCKET"
    
    run env TMUX="" PROJECT_ROOT="$TEST_PROJECT_DIR" "$PROJ_TMUX_BINARY" session create --switch=false test.org/test.project
    [ "$status" -eq 0 ]
    
    # Check that dots are replaced with dashes in session name
    run tmux -S "$TEST_TMUX_SOCKET" list-sessions -F "#{session_name}"
    [ "$status" -eq 0 ]
    [[ "$output" == *"proj-test-org-test-project"* ]]
}

@test "session creation is idempotent" {
    cd "$TEST_PROJECT_DIR/testorg/testproject"
    
    # Set tmux socket for proj-tmux to use isolated environment
    export TMUX_SOCKET="$TEST_TMUX_SOCKET"
    
    # Create session twice
    env TMUX="" PROJECT_ROOT="$TEST_PROJECT_DIR" "$PROJ_TMUX_BINARY" session create --switch=false testorg/testproject
    run env TMUX="" PROJECT_ROOT="$TEST_PROJECT_DIR" "$PROJ_TMUX_BINARY" session create --switch=false testorg/testproject
    [ "$status" -eq 0 ]
    
    # Should still only have one session
    sessions=$(tmux -S "$TEST_TMUX_SOCKET" list-sessions -F "#{session_name}" | grep "proj-testorg-testproject" | wc -l)
    [ "$sessions" -eq 1 ]
}

@test "session creation with proper working directory" {
    cd "$TEST_PROJECT_DIR/testorg/testproject"
    
    # Set tmux socket for proj-tmux to use isolated environment
    export TMUX_SOCKET="$TEST_TMUX_SOCKET"
    
    # Create session with proper root directory
    env TMUX="" PROJECT_ROOT="$TEST_PROJECT_DIR" "$PROJ_TMUX_BINARY" session create --switch=false testorg/testproject
    
    # Check working directory of session
    wd=$(tmux -S "$TEST_TMUX_SOCKET" display-message -t proj-testorg-testproject -p "#{pane_current_path}")
    expected_dir="$TEST_PROJECT_DIR/testorg/testproject"
    echo "Working directory: $wd"
    echo "Expected directory: $expected_dir"
    [ "$wd" = "$expected_dir" ]
}

@test "session cleanup works" {
    cd "$TEST_PROJECT_DIR/testorg/testproject"
    
    # Set tmux socket for proj-tmux to use isolated environment
    export TMUX_SOCKET="$TEST_TMUX_SOCKET"
    
    # Create session
    env TMUX="" PROJECT_ROOT="$TEST_PROJECT_DIR" "$PROJ_TMUX_BINARY" session create --switch=false testorg/testproject
    
    # Verify it exists
    tmux -S "$TEST_TMUX_SOCKET" has-session -t proj-testorg-testproject
    
    # Kill session using tmux service
    tmux -S "$TEST_TMUX_SOCKET" kill-session -t proj-testorg-testproject
    
    # Verify it's gone
    run tmux -S "$TEST_TMUX_SOCKET" has-session -t proj-testorg-testproject
    [ "$status" -ne 0 ]
}