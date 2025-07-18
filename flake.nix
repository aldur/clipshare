{
  description = "clipshare - A simple REST clipboard service";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # Common attributes for all packages
        commonAttrs = {
          version = "0.1.0";
          src = ./.;
          vendorHash = null;
          meta = with pkgs.lib; {
            homepage = "https://github.com/aldur/clipshare";
            license = licenses.mit;
            maintainers = [ ];
          };
        };

        # Helper function to create test derivations
        mkTest = name: testCommand:
          pkgs.stdenv.mkDerivation {
            name = "clipshare-${name}";
            src = builtins.path {
              path = ./.;
              name = "source";
            };
            buildInputs = with pkgs; [ go ];
            buildPhase = ''
              export HOME=$(mktemp -d)
              ${testCommand}
            '';
            installPhase = ''
              touch $out
            '';
          };
      in {
        packages = {
          default = self.packages.${system}.server;

          server = pkgs.buildGoModule (commonAttrs // {
            pname = "clipshare-server";
            meta = commonAttrs.meta // {
              description = "clipshare server - simple REST clipboard service";
            };
            subPackages = [ "." ];
            postInstall = ''
              mv $out/bin/clipshare $out/bin/clipshare-server
            '';
          });

          client = pkgs.buildGoModule (commonAttrs // {
            pname = "clipshare-client";
            subPackages = [ "cmd/client" ];
            meta = commonAttrs.meta // {
              description = "clipshare client - simple REST clipboard service";
            };
            postInstall = ''
              mv $out/bin/client $out/bin/clipshare-client
            '';
          });
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [ go gopls gotools go-tools ];
        };

        checks = {
          # Unit tests
          unit-tests = mkTest "unit-tests" "go test -v ./...";

          # Integration tests
          integration-tests = mkTest "integration-tests"
            ''go test -v -run "^TestClientServer" ./...'';

          # Build tests - ensure packages build successfully
          build-server = self.packages.${system}.server;
          build-client = self.packages.${system}.client;

          # Lint and format checks
          lint = mkTest "lint" ''
            go fmt ./...
            go vet ./...

            # Check if gofmt would make changes
            if [ -n "$(gofmt -l .)" ]; then
              echo "The following files need formatting:"
              gofmt -l .
              exit 1
            fi
          '';
        };
      }) // {
        nixosModules.default = { config, lib, pkgs, ... }: {
          imports = [ ./nixos-module.nix ];
          nixpkgs.overlays = [ self.overlays.default ];
        };

        homeManagerModules.default = { config, lib, pkgs, ... }: {
          imports = [ ./home-manager-module.nix ];
          nixpkgs.overlays = [ self.overlays.default ];
        };

        overlays.default = final: prev: {
          clipshare-server = self.packages.${final.system}.server;
          clipshare-client = self.packages.${final.system}.client;
        };
      };
}

