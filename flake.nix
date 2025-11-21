{
  description = "clipshare - A simple REST clipboard service";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
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

        # Common attributes for all packages
        commonAttrs = {
          version = "0.1.0";
          src = self;
          vendorHash = null; # Look ma, no deps!
          meta = with pkgs.lib; {
            homepage = "https://github.com/aldur/clipshare";
            license = licenses.mit;
            maintainers = [ ];
          };
        };

        # Helper function to create test derivations
        mkTest =
          name: testCommand:
          pkgs.stdenv.mkDerivation {
            name = "clipshare-${name}";
            src = builtins.path {
              path = self;
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

        server = pkgs.buildGoModule (
          commonAttrs
          // {
            pname = "clipshare-server";
            meta = commonAttrs.meta // {
              description = "clipshare server - simple REST clipboard service";
            };
            subPackages = [ "." ];
            postInstall = ''
              mv $out/bin/clipshare $out/bin/clipshare-server
            '';
          }
        );

        client = pkgs.buildGoModule (
          commonAttrs
          // {
            pname = "clipshare";
            subPackages = [ "cmd/client" ];
            meta = commonAttrs.meta // {
              description = "clipshare client - simple REST clipboard service";
            };
            postInstall = ''
              mv $out/bin/client $out/bin/clipshare
            '';
          }
        );
      in
      {
        packages = {
          inherit client server;
          default = client;

          dockerImage = pkgs.dockerTools.buildImage {
            name = "clipshare-server";
            tag = "latest";
            copyToRoot = server;
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
          ];
        };

        checks = {
          # Unit tests
          unit-tests = mkTest "unit-tests" "go test -v ./...";

          # Integration tests
          integration-tests = mkTest "integration-tests" ''go test -v -run "^TestClientServer" ./...'';

          # Build tests - ensure packages build successfully
          build-server = self.packages.${system}.server;
          build-client = self.packages.${system}.client;

          # NixOS module tests
          nixos-module-tests = (import ./nix/nixos-module-test.nix { inherit pkgs; }).allTests;

          # Home Manager module tests
          home-manager-module-tests = (import ./nix/home-manager-module-test.nix { inherit pkgs; }).allTests;

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
      }
    )
    // {
      nixosModules.default = _: {
        imports = [ ./nix/nixos-module.nix ];
        nixpkgs.overlays = [ self.overlays.default ];
      };

      homeManagerModules.default =
        { pkgs, ... }:
        {
          _module.args.clipsharePackage = self.packages.${pkgs.stdenv.hostPlatform.system}.client;
          imports = [ ./nix/home-manager-module.nix ];
        };

      overlays.default =
        final: prev:
        let
          inherit (final.stdenv.hostPlatform) system;
        in
        {
          clipshare-server = self.packages.${system}.server;
          clipshare = self.packages.${system}.client;
        };
    };
}
