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

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = self.packages.${system}.project;
        
        packages.project = pkgs.buildGo123Module rec {
          pname = "project";
          version = "0.1.0";
          
          src = ./.;
          
          # Use fakeHash for development, replaced with real hash during releases
          vendorHash = pkgs.lib.fakeHash;

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
            
            # Copy test scripts
            cp ${./test-completion.sh} $out/share/doc/project/examples/test-completion.sh
            cp ${./test-shell.sh} $out/share/doc/project/examples/test-shell.sh
            chmod +x $out/share/doc/project/examples/*.sh
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

        # Development shell
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

        # Apps for easy running
        apps.default = {
          type = "app";
          program = "${self.packages.${system}.project}/bin/proj";
        };

        # Checks for CI
        checks = {
          project-build = self.packages.${system}.project;
          
          # Note: Shell integration tests require interactive environment
          # Run them manually with `make test-integration` or `make test-nix`
        };
      }) // {
        # Overlay for use in other flakes
        overlays.default = final: prev: {
          project = self.packages.${final.system}.project;
        };
      };
}