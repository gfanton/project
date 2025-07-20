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
    echo "Project testing environment loaded"
    echo "Go version: $(go version)"
    echo "Zsh version: $(zsh --version | head -1)"
    
    # Set up test environment variables
    export PROJECT_TEST_DIR="$PWD/test-env"
    export PROJECT_TEST_BIN="$PWD/build/proj"
    
    # Ensure we have a clean test binary
    make build
  '';
}