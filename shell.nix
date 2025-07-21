{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    # Go toolchain
    go_1_23
    golangci-lint
    gopls
    
    # Shells for testing
    bash
    zsh
    
    # Testing tools
    bats
    expect
    tmux
    
    # Development tools
    gnumake
    git
    
    # For shell completion testing
    fzf
  ];
  
  shellHook = ''
    # Set up a clean zsh environment for completion testing
    export ZDOTDIR="$PWD/.zsh-test"
    export TEST_HOME="$PWD/.zsh-test"
    
    # Create test directory structure
    mkdir -p "$ZDOTDIR"
    
    # Create minimal zshrc for testing
    cat > "$ZDOTDIR/.zshrc" << 'EOF'
# Minimal zshrc for completion testing
autoload -Uz compinit
compinit -d "$ZDOTDIR/.zcompdump"

# Enable completion menu
zstyle ':completion:*' menu select
zstyle ':completion:*' list-colors ""

# Basic prompt
PS1='test-zsh %~ $ '

# Source project completion if available
if [[ -f "./build/proj" ]]; then
  eval "$(./build/proj init zsh)"
fi
EOF

    # Create test project structure
    mkdir -p "$ZDOTDIR/code/gfanton"
    mkdir -p "$ZDOTDIR/code/example"
    
    # Add some test projects
    mkdir -p "$ZDOTDIR/code/gfanton/test-project"
    mkdir -p "$ZDOTDIR/code/gfanton/another-project" 
    mkdir -p "$ZDOTDIR/code/example/demo-app"
    
    # Initialize as git repos
    (cd "$ZDOTDIR/code/gfanton/test-project" && git init --quiet 2>/dev/null || true)
    (cd "$ZDOTDIR/code/gfanton/another-project" && git init --quiet 2>/dev/null || true)
    (cd "$ZDOTDIR/code/example/demo-app" && git init --quiet 2>/dev/null || true)
    
    # Build the project first
    make build
    
    echo "========================================="
    echo "Clean zsh environment ready for testing!"
    echo "========================================="
    echo "Test directory: $ZDOTDIR"
    echo "Project root: $ZDOTDIR/code"
    echo ""
    echo "To start testing completion:"
    echo "1. cd \$ZDOTDIR"
    echo "2. HOME=\$TEST_HOME zsh"
    echo "3. Try: p <TAB> or p test<TAB>"
    echo ""
    echo "Available test projects:"
    echo "- gfanton/test-project"
    echo "- gfanton/another-project" 
    echo "- example/demo-app"
    echo ""
    echo "Quick test command:"
    echo "  cd \$ZDOTDIR && HOME=\$TEST_HOME zsh"
    echo "========================================="
  '';
}