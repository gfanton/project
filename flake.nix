{
  description = "A Git-based project management tool with zoxide-like navigation for GitHub-style directory structure";

  # For consumers: Use release branch for real vendorHash or latest tag
  # Examples:
  #   project.url = "github:gfanton/project/release";        # Always latest from release branch
  #   project.url = "github:gfanton/project?ref=latest";     # Latest release tag
  #   project.url = "github:gfanton/project?ref=v1.2.3";     # Specific version tag

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = self.packages.${system}.project;

        packages.project = pkgs.buildGo123Module rec {
          pname = "project";
          version = "0.12.0";

          src = ./.;

          # Use fakeHash for development, replaced with real hash during releases
          vendorHash = "sha256-B375AvklOVKxpIR60CatnmRgOFpqhlKyKF32isB+ncI=";

          # Override build flags to not use vendor mode
          buildFlags = [ "-mod=mod" ];

          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];

          # Build from cmd/proj
          subPackages = [ "cmd/proj" ];

          # Test dependencies for integration tests
          nativeBuildInputs = with pkgs; [
            git
            zsh
            bats
            expect
          ];

          # Skip tests in Nix build - they require interactive environment
          # Tests can be run separately with `make test` or in development shell
          doCheck = false;

          # Install documentation and examples
          postInstall = ''
            # Create examples directory
            mkdir -p $out/share/doc/project/examples

            # Copy shell.nix for development
            cp ${./shell.nix} $out/share/doc/project/examples/shell.nix
          '';

          meta = with pkgs.lib; {
            description = "A Git-based project management tool with zoxide-like navigation";
            longDescription = ''
              A tool that organizes Git repositories in a GitHub-style directory structure
              and provides fast project navigation similar to zoxide. Features include:
              - Fast fuzzy search and navigation between projects
              - Enhanced zsh completion with menu selection
              - GitHub-style organization (username/project-name)
              - Shell integration with the 'p' command
            '';
            homepage = "https://github.com/gfanton/project";
            license = licenses.mit;
            maintainers = [ "gfanton" ];
            mainProgram = "proj";
          };
        };

        # Tmux integration binary
        packages.proj-tmux = pkgs.buildGo123Module rec {
          pname = "proj-tmux";
          version = "0.12.0";

          src = ./.;

          # Disable vendoring for development (internal packages)
          vendorHash = null;

          # Override build flags to not use vendor mode
          buildFlags = [ "-mod=mod" ];

          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];

          # Build from plugins/proj-tmux
          subPackages = [ "plugins/proj-tmux" ];

          meta = with pkgs.lib; {
            description = "Tmux integration for proj";
            homepage = "https://github.com/gfanton/project";
            license = licenses.mit;
            maintainers = [ "gfanton" ];
            mainProgram = "proj-tmux";
          };
        };

        # Development shells
        devShells.default = pkgs.mkShell {
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
            echo "Project development environment loaded"
            echo "Go version: $(go version)"
            echo "Zsh version: $(zsh --version | head -1)"

            # Set up test environment variables
            export PROJECT_TEST_DIR="$PWD/test-env"
            export PROJECT_TEST_BIN="$PWD/build/proj"

            echo ""
            echo "Available commands:"
            echo "  make build              - Build the project"
            echo "  make test               - Run Go unit tests"
            echo "  make test-shell         - Run Go-based shell tests"
            echo "  make test-integration   - Run BATS/Expect tests"
            echo "  make test-nix           - Run all tests in Nix"
            echo "  ./test-completion.sh    - Test enhanced completion"
            echo "  ./test-shell.sh         - Test shell integration"
          '';
        };

        # Dedicated testing development shell for tmux integration
        devShells.testing = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain
            go_1_23
            golangci-lint
            gopls

            # Testing frameworks and tools
            tmux
            expect
            bats

            # Shell environments
            bash
            bashInteractive
            zsh

            # Development and testing utilities
            gnumake
            git
            fzf

            # Additional testing utilities
            procps # for ps, needed by tmux testing
            util-linux # for script, timeout commands
          ];

          shellHook = ''
            echo "Tmux Integration Testing Environment"
            echo "===================================="
            echo "Tmux version: $(tmux -V)"
            echo "BATS version: $(bats --version)"
            echo "Expect version: $(expect -v | head -1)"
            echo ""

            # Set up isolated tmux environment for testing
            export TEST_TMUX_DIR=$(mktemp -d -t tmux-test-XXXXXX)
            export TEST_TMUX_SOCKET="$TEST_TMUX_DIR/test-socket"
            export TMUX_TMPDIR="$TEST_TMUX_DIR"

            # Prevent interference with user's tmux sessions
            alias test-tmux='tmux -S $TEST_TMUX_SOCKET'

            # Test project directory setup
            export TEST_PROJECT_DIR="$TEST_TMUX_DIR/projects"
            mkdir -p "$TEST_PROJECT_DIR"

            # Build proj binary for testing if not exists
            if [ ! -f "./build/proj" ]; then
              echo "Building proj binary for testing..."
              make build
            fi
            export PROJ_BINARY="$PWD/build/proj"

            echo "Testing environment ready!"
            echo ""
            echo "Environment variables:"
            echo "  TEST_TMUX_DIR=$TEST_TMUX_DIR"
            echo "  TEST_TMUX_SOCKET=$TEST_TMUX_SOCKET"
            echo "  TEST_PROJECT_DIR=$TEST_PROJECT_DIR"
            echo "  PROJ_BINARY=$PROJ_BINARY"
            echo ""
            echo "Available aliases:"
            echo "  test-tmux    - Isolated tmux instance for testing"
            echo ""
            echo "Cleanup on exit will remove $TEST_TMUX_DIR"
            echo ""
            echo "Available test commands:"
            echo "  bats tests/unit/           - Run BATS unit tests"
            echo "  tests/run_tests.sh         - Run all tests"
            echo "  make test-tmux             - Run tmux integration tests"

            # Cleanup function for exit
            cleanup_test_env() {
              echo "Cleaning up test environment..."
              tmux -S "$TEST_TMUX_SOCKET" kill-server 2>/dev/null || true
              rm -rf "$TEST_TMUX_DIR"
            }

            # Register cleanup on exit
            trap cleanup_test_env EXIT
          '';
        };

        # Apps for easy running
        apps.default = {
          type = "app";
          program = "${self.packages.${system}.project}/bin/proj";
        };

        # Checks for CI
        checks = {
          project-build = self.packages.${system}.project;

          # Basic tmux testing (non-interactive)
          tmux-unit-tests = pkgs.stdenv.mkDerivation {
            name = "tmux-unit-tests";
            src = ./.;

            nativeBuildInputs = with pkgs; [
              tmux
              bats
              bash
              git
              self.packages.${system}.project
              self.packages.${system}.proj-tmux
            ];

            buildPhase = ''
              # Set up test environment
              export TEST_TMUX_DIR=$(mktemp -d)
              export TEST_TMUX_SOCKET="$TEST_TMUX_DIR/test-socket"
              export TEST_PROJECT_DIR="$TEST_TMUX_DIR/projects"
              export TMUX_TMPDIR="$TEST_TMUX_DIR"
              export PROJ_BINARY="${self.packages.${system}.project}/bin/proj"
              export PROJ_TMUX_BINARY="${self.packages.${system}.proj-tmux}/bin/proj-tmux"

              # Run BATS unit tests
              if [[ -d tests/unit && $(find tests/unit -name "*.bats" | wc -l) -gt 0 ]]; then
                bats tests/unit/ || exit 1
              fi

              # Cleanup
              tmux -S "$TEST_TMUX_SOCKET" kill-server 2>/dev/null || true
              rm -rf "$TEST_TMUX_DIR"
            '';

            installPhase = ''
              mkdir -p $out
              echo "Tmux unit tests passed" > $out/test-results.txt
            '';
          };

          # Note: Interactive shell integration tests require manual execution
          # Run them with `nix develop .#testing` and then `bats tests/unit/`
        };
      }
    )
    // {
      # Overlay for use in other flakes
      overlays.default = final: prev: {
        project = self.packages.${final.system}.project;
      };
    };
}
