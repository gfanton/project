{
  description = "A Git-based project management tool with zoxide-like navigation for GitHub-style directory structure";

  # For consumers: Use specific version tags or latest from master
  # Examples:
  #   project.url = "github:gfanton/project";                # Latest from master
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
    let
      # Global shared version for all packages and systems
      # This version is automatically updated by the release script
      projectVersion = "0.16.5";
    in
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # Define project package
        projectPackage = pkgs.buildGoModule rec {
          pname = "project";
          version = projectVersion;

          src = ./.;

          # Vendor hash - updated by release script or manually during development
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
            # bats  # Temporarily disabled due to nokogiri build issue in nixpkgs-unstable
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

        # Define proj-tmux package
        projTmuxPackage = pkgs.buildGoModule rec {
          pname = "proj-tmux";
          version = projectVersion;

          src = ./.;

          # Same vendorHash as main project since they share go.mod
          vendorHash = "sha256-B375AvklOVKxpIR60CatnmRgOFpqhlKyKF32isB+ncI=";

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

        # Define tmux plugin package
        tmuxProjPackage = pkgs.tmuxPlugins.mkTmuxPlugin rec {
          pluginName = "tmux-proj";
          version = projectVersion;
          rtpFilePath = "proj-tmux.tmux";

          src = ./plugins/proj-tmux/plugin;

          # Dependencies - the plugin needs proj and proj-tmux binaries
          buildInputs = [ projectPackage projTmuxPackage ];

          # Wrap the scripts to have access to the binaries
          nativeBuildInputs = [ pkgs.makeWrapper ];

          postInstall = ''
            # Wrap the main plugin script
            wrapProgram $out/share/tmux-plugins/tmux-proj/proj-tmux.tmux \
              --prefix PATH : ${pkgs.lib.makeBinPath [ projectPackage projTmuxPackage ]}

            # Wrap all scripts in the scripts directory
            for script in $out/share/tmux-plugins/tmux-proj/scripts/*.sh; do
              wrapProgram "$script" \
                --prefix PATH : ${pkgs.lib.makeBinPath [ projectPackage projTmuxPackage ]}
            done
          '';

          meta = with pkgs.lib; {
            description = "A tmux plugin for proj - Git-based project management with session integration";
            homepage = "https://github.com/gfanton/project";
            license = licenses.mit;
            platforms = platforms.unix;
            maintainers = [ "gfanton" ];
          };
        };
      in
      {
        # Packages - no self-references to avoid circular dependencies
        packages = {
          default = projectPackage;
          project = projectPackage;
          proj-tmux = projTmuxPackage;
          tmux-proj = tmuxProjPackage;
        };

        # Development shells
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain
            go_1_25

            # Shells for testing
            bash
            zsh

            # Testing tools
            # bats  # Temporarily disabled due to nokogiri build issue in nixpkgs-unstable
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
            go_1_25

            # Testing frameworks and tools
            tmux
            expect
            # bats  # Temporarily disabled due to nokogiri build issue in nixpkgs-unstable

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
          program = "${projectPackage}/bin/proj";
        };

        # Checks for CI
        checks = {
          # Simple build test without circular dependencies
          project-build = projectPackage;
          proj-tmux-build = projTmuxPackage;
          tmux-proj-build = tmuxProjPackage;

          # Note: Advanced integration tests with package dependencies
          # should be run manually with `nix develop .#testing` and then `bats tests/unit/`
        };
      }
    )
    // {
      # Overlay for use in other flakes - allows importing tmux-proj easily
      overlays.default = final: prev: {
        # Make project packages available in nixpkgs
        project = final.callPackage (
          { buildGoModule }:
          buildGoModule rec {
            pname = "project";
            version = projectVersion;
            src = ./.;
            vendorHash = "sha256-B375AvklOVKxpIR60CatnmRgOFpqhlKyKF32isB+ncI=";
            buildFlags = [ "-mod=mod" ];
            ldflags = [
              "-s"
              "-w"
              "-X main.version=${version}"
            ];
            subPackages = [ "cmd/proj" ];
            meta = with final.lib; {
              description = "A Git-based project management tool with zoxide-like navigation";
              homepage = "https://github.com/gfanton/project";
              license = licenses.mit;
              maintainers = [ "gfanton" ];
              mainProgram = "proj";
            };
          }
        ) { };

        # Make tmux plugin easily available
        tmux-proj = final.tmuxPlugins.mkTmuxPlugin rec {
          pluginName = "tmux-proj";
          version = projectVersion;
          rtpFilePath = "proj-tmux.tmux";
          src = ./plugins/proj-tmux/plugin;

          # Dependencies - the plugin needs proj and proj-tmux binaries
          buildInputs = [ final.project ];
          nativeBuildInputs = [ final.makeWrapper ];

          postInstall = ''
            # Note: The consuming nixpkgs config should ensure proj is in PATH
            # This plugin expects proj and proj-tmux to be available
          '';

          meta = with final.lib; {
            description = "A tmux plugin for proj - Git-based project management with session integration";
            homepage = "https://github.com/gfanton/project";
            license = licenses.mit;
            platforms = platforms.unix;
            maintainers = [ "gfanton" ];
          };
        };
      };
    };
}
