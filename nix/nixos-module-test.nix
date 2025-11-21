# Tests for the NixOS module
# Run with: nix-build --no-out-link -E '(import ./nix/nixos-module-test.nix {})'
# Or use `nix flake check` to run all checks including this one

{
  pkgs ? import <nixpkgs> { },
}:

let
  lib = pkgs.lib;

  # Mock package for testing
  mockPackage = pkgs.writeShellScriptBin "clipshare-server" ''
    echo "Mock clipshare-server running"
  '';

  # Evaluate the module with given config
  evalModule =
    moduleConfig:
    lib.evalModules {
      modules = [
        # Import the module under test
        ../nix/nixos-module.nix

        # Provide mock options that the module expects from NixOS
        {
          options = {
            users.users = lib.mkOption {
              type = lib.types.attrsOf (
                lib.types.submodule {
                  options = {
                    description = lib.mkOption { type = lib.types.str; };
                    group = lib.mkOption { type = lib.types.str; };
                    isSystemUser = lib.mkOption { type = lib.types.bool; };
                  };
                }
              );
              default = { };
            };
            users.groups = lib.mkOption {
              type = lib.types.attrsOf (lib.types.submodule { options = { }; });
              default = { };
            };
            systemd.services = lib.mkOption {
              type = lib.types.attrsOf lib.types.anything;
              default = { };
            };
            networking.firewall.allowedTCPPorts = lib.mkOption {
              type = lib.types.listOf lib.types.port;
              default = [ ];
            };
          };
        }

        # Test configuration
        {
          _module.args.pkgs = pkgs // {
            clipshare-server = mockPackage;
          };
        }

        moduleConfig
      ];
    };

  # Test cases
  tests = {
    # Test 1: Module disabled by default
    test-disabled-by-default =
      let
        result = evalModule { };
      in
      pkgs.runCommand "test-disabled-by-default" { } ''
        ${lib.optionalString (result.config.systemd.services ? clipshare) ''
          echo "FAIL: Service should not exist when disabled"
          exit 1
        ''}
        echo "PASS: Service not created when disabled"
        touch $out
      '';

    # Test 2: Basic enable creates service
    test-basic-enable =
      let
        result = evalModule {
          services.clipshare.enable = true;
        };
      in
      pkgs.runCommand "test-basic-enable" { } ''
        ${lib.optionalString (!result.config.systemd.services ? clipshare) ''
          echo "FAIL: Service should exist when enabled"
          exit 1
        ''}
        echo "PASS: Service created when enabled"
        touch $out
      '';

    # Test 3: Default values
    test-default-values =
      let
        result = evalModule {
          services.clipshare.enable = true;
        };
        svc = result.config.systemd.services.clipshare;
      in
      pkgs.runCommand "test-default-values" { } ''
        # Check default host
        if [ "${svc.environment.HOST}" != "localhost" ]; then
          echo "FAIL: Default host should be localhost, got ${svc.environment.HOST}"
          exit 1
        fi

        # Check default port
        if [ "${svc.environment.PORT}" != "8080" ]; then
          echo "FAIL: Default port should be 8080, got ${svc.environment.PORT}"
          exit 1
        fi

        echo "PASS: Default values are correct"
        touch $out
      '';

    # Test 4: Custom host and port
    test-custom-host-port =
      let
        result = evalModule {
          services.clipshare = {
            enable = true;
            host = "0.0.0.0";
            port = 9000;
          };
        };
        svc = result.config.systemd.services.clipshare;
      in
      pkgs.runCommand "test-custom-host-port" { } ''
        if [ "${svc.environment.HOST}" != "0.0.0.0" ]; then
          echo "FAIL: Custom host not set correctly"
          exit 1
        fi

        if [ "${svc.environment.PORT}" != "9000" ]; then
          echo "FAIL: Custom port not set correctly"
          exit 1
        fi

        echo "PASS: Custom host and port work"
        touch $out
      '';

    # Test 5: User and group creation
    test-user-group-creation =
      let
        result = evalModule {
          services.clipshare.enable = true;
        };
      in
      pkgs.runCommand "test-user-group-creation" { } ''
        ${lib.optionalString (!result.config.users.users ? clipshare) ''
          echo "FAIL: User 'clipshare' should be created"
          exit 1
        ''}
        ${lib.optionalString (!result.config.users.groups ? clipshare) ''
          echo "FAIL: Group 'clipshare' should be created"
          exit 1
        ''}
        ${lib.optionalString (!result.config.users.users.clipshare.isSystemUser) ''
          echo "FAIL: User should be a system user"
          exit 1
        ''}
        echo "PASS: User and group created correctly"
        touch $out
      '';

    # Test 6: Custom user and group
    test-custom-user-group =
      let
        result = evalModule {
          services.clipshare = {
            enable = true;
            user = "myuser";
            group = "mygroup";
          };
        };
      in
      pkgs.runCommand "test-custom-user-group" { } ''
        ${lib.optionalString (!result.config.users.users ? myuser) ''
          echo "FAIL: Custom user 'myuser' should be created"
          exit 1
        ''}
        ${lib.optionalString (!result.config.users.groups ? mygroup) ''
          echo "FAIL: Custom group 'mygroup' should be created"
          exit 1
        ''}
        echo "PASS: Custom user and group work"
        touch $out
      '';

    # Test 7: Firewall closed by default
    test-firewall-closed-by-default =
      let
        result = evalModule {
          services.clipshare.enable = true;
        };
      in
      pkgs.runCommand "test-firewall-closed-by-default" { } ''
        ${lib.optionalString (builtins.elem 8080 result.config.networking.firewall.allowedTCPPorts) ''
          echo "FAIL: Firewall should be closed by default"
          exit 1
        ''}
        echo "PASS: Firewall closed by default"
        touch $out
      '';

    # Test 8: Firewall opens when requested
    test-firewall-opens =
      let
        result = evalModule {
          services.clipshare = {
            enable = true;
            openFirewall = true;
          };
        };
      in
      pkgs.runCommand "test-firewall-opens" { } ''
        ${lib.optionalString (!builtins.elem 8080 result.config.networking.firewall.allowedTCPPorts) ''
          echo "FAIL: Firewall should open port 8080"
          exit 1
        ''}
        echo "PASS: Firewall opens correctly"
        touch $out
      '';

    # Test 9: Firewall opens custom port
    test-firewall-custom-port =
      let
        result = evalModule {
          services.clipshare = {
            enable = true;
            port = 9999;
            openFirewall = true;
          };
        };
      in
      pkgs.runCommand "test-firewall-custom-port" { } ''
        ${lib.optionalString (!builtins.elem 9999 result.config.networking.firewall.allowedTCPPorts) ''
          echo "FAIL: Firewall should open custom port 9999"
          exit 1
        ''}
        echo "PASS: Firewall opens custom port"
        touch $out
      '';

    # Test 10: Service configuration
    test-service-config =
      let
        result = evalModule {
          services.clipshare.enable = true;
        };
        svc = result.config.systemd.services.clipshare;
      in
      pkgs.runCommand "test-service-config" { } ''
        # Check service description
        if [ "${svc.description}" != "Clipshare Server" ]; then
          echo "FAIL: Wrong service description"
          exit 1
        fi

        # Check restart policy
        if [ "${svc.serviceConfig.Restart}" != "on-failure" ]; then
          echo "FAIL: Restart policy should be on-failure"
          exit 1
        fi

        echo "PASS: Service configuration correct"
        touch $out
      '';

    # Test 11: Security hardening settings
    test-security-hardening =
      let
        result = evalModule {
          services.clipshare.enable = true;
        };
        svc = result.config.systemd.services.clipshare;
      in
      pkgs.runCommand "test-security-hardening" { } ''
        ${lib.optionalString (!svc.serviceConfig.NoNewPrivileges) ''
          echo "FAIL: NoNewPrivileges should be true"
          exit 1
        ''}
        ${lib.optionalString (!svc.serviceConfig.PrivateTmp) ''
          echo "FAIL: PrivateTmp should be true"
          exit 1
        ''}
        ${lib.optionalString (svc.serviceConfig.ProtectSystem != "strict") ''
          echo "FAIL: ProtectSystem should be strict"
          exit 1
        ''}
        ${lib.optionalString (!svc.serviceConfig.ProtectHome) ''
          echo "FAIL: ProtectHome should be true"
          exit 1
        ''}
        ${lib.optionalString (!svc.serviceConfig.PrivateDevices) ''
          echo "FAIL: PrivateDevices should be true"
          exit 1
        ''}
        echo "PASS: Security hardening settings correct"
        touch $out
      '';

    # Test 12: Service ordering
    test-service-ordering =
      let
        result = evalModule {
          services.clipshare.enable = true;
        };
        svc = result.config.systemd.services.clipshare;
      in
      pkgs.runCommand "test-service-ordering" { } ''
        ${lib.optionalString (!builtins.elem "multi-user.target" svc.wantedBy) ''
          echo "FAIL: Service should be wanted by multi-user.target"
          exit 1
        ''}
        ${lib.optionalString (!builtins.elem "network.target" svc.after) ''
          echo "FAIL: Service should start after network.target"
          exit 1
        ''}
        echo "PASS: Service ordering correct"
        touch $out
      '';
  };

  # Combine all tests - use runCommand to aggregate results
  allTests = pkgs.runCommand "nixos-module-tests" { } ''
    mkdir -p $out
    ${lib.concatMapStringsSep "\n" (
      test: "ln -s ${test} $out/${test.name}"
    ) (builtins.attrValues tests)}
    echo "All NixOS module tests passed"
  '';
in
{
  inherit tests allTests;
  # Allow running with `nix-build -A allTests`
  __functor = self: self.allTests;
}
