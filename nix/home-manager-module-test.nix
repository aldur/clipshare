# Tests for the Home Manager module
# Run with: nix-build --no-out-link -E '(import ./nix/home-manager-module-test.nix {})'
# Or use `nix flake check` to run all checks including this one

{
  pkgs ? import <nixpkgs> { },
}:

let
  lib = pkgs.lib;

  # Mock package for testing
  mockPackage = pkgs.writeShellScriptBin "clipshare" ''
    echo "Mock clipshare running"
  '';

  # Evaluate the module with given config
  evalModule =
    moduleConfig:
    lib.evalModules {
      modules = [
        # Import the module under test
        ../nix/home-manager-module.nix

        # Provide mock options that the module expects from home-manager
        {
          options = {
            home.packages = lib.mkOption {
              type = lib.types.listOf lib.types.package;
              default = [ ];
            };
            home.sessionVariables = lib.mkOption {
              type = lib.types.attrsOf lib.types.str;
              default = { };
            };
            home.shellAliases = lib.mkOption {
              type = lib.types.attrsOf lib.types.str;
              default = { };
            };
            networking.hostName = lib.mkOption {
              type = lib.types.nullOr lib.types.str;
              default = "test-hostname";
            };
          };
        }

        # Test configuration
        {
          _module.args.pkgs = pkgs // {
            clipshare = mockPackage;
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
        ${lib.optionalString (result.config.home.packages != [ ]) ''
          echo "FAIL: No packages should be added when disabled"
          exit 1
        ''}
        ${lib.optionalString (result.config.home.sessionVariables != { }) ''
          echo "FAIL: No session variables should be set when disabled"
          exit 1
        ''}
        echo "PASS: Module disabled by default"
        touch $out
      '';

    # Test 2: Basic enable adds package
    test-basic-enable =
      let
        result = evalModule {
          programs.clipshare.enable = true;
        };
      in
      pkgs.runCommand "test-basic-enable" { } ''
        ${lib.optionalString (result.config.home.packages == [ ]) ''
          echo "FAIL: Package should be added when enabled"
          exit 1
        ''}
        echo "PASS: Package added when enabled"
        touch $out
      '';

    # Test 3: URL is null by default (no env var set)
    test-url-null-by-default =
      let
        result = evalModule {
          programs.clipshare.enable = true;
          networking.hostName = null; # Also set hostname to null to avoid device env var
        };
      in
      pkgs.runCommand "test-url-null-by-default" { } ''
        ${lib.optionalString (result.config.home.sessionVariables ? CLIPSHARE_URL) ''
          echo "FAIL: CLIPSHARE_URL should not be set when url is null"
          exit 1
        ''}
        echo "PASS: URL is null by default, no env var set"
        touch $out
      '';

    # Test 4: Custom URL sets env var
    test-custom-url =
      let
        result = evalModule {
          programs.clipshare = {
            enable = true;
            url = "https://clipboard.example.com:9000";
          };
        };
      in
      pkgs.runCommand "test-custom-url" { } ''
        ${lib.optionalString (!result.config.home.sessionVariables ? CLIPSHARE_URL) ''
          echo "FAIL: CLIPSHARE_URL should be set when url is configured"
          exit 1
        ''}
        if [ "${result.config.home.sessionVariables.CLIPSHARE_URL}" != "https://clipboard.example.com:9000" ]; then
          echo "FAIL: Custom URL not set correctly"
          exit 1
        fi
        echo "PASS: Custom URL works"
        touch $out
      '';

    # Test 5: Default device from hostname
    test-default-device =
      let
        result = evalModule {
          programs.clipshare.enable = true;
          networking.hostName = "my-workstation";
        };
      in
      pkgs.runCommand "test-default-device" { } ''
        ${lib.optionalString (!result.config.home.sessionVariables ? CLIPSHARE_DEVICE) ''
          echo "FAIL: CLIPSHARE_DEVICE should be set when hostname is configured"
          exit 1
        ''}
        if [ "${result.config.home.sessionVariables.CLIPSHARE_DEVICE}" != "my-workstation" ]; then
          echo "FAIL: Device should default to hostname"
          exit 1
        fi
        echo "PASS: Default device uses hostname"
        touch $out
      '';

    # Test 6: Custom device name
    test-custom-device =
      let
        result = evalModule {
          programs.clipshare = {
            enable = true;
            device = "custom-device-name";
          };
        };
      in
      pkgs.runCommand "test-custom-device" { } ''
        ${lib.optionalString (!result.config.home.sessionVariables ? CLIPSHARE_DEVICE) ''
          echo "FAIL: CLIPSHARE_DEVICE should be set when device is configured"
          exit 1
        ''}
        if [ "${result.config.home.sessionVariables.CLIPSHARE_DEVICE}" != "custom-device-name" ]; then
          echo "FAIL: Custom device name not set correctly"
          exit 1
        fi
        echo "PASS: Custom device name works"
        touch $out
      '';

    # Test 7: Aliases enabled by default
    test-aliases-enabled-by-default =
      let
        result = evalModule {
          programs.clipshare.enable = true;
        };
      in
      pkgs.runCommand "test-aliases-enabled-by-default" { } ''
        ${lib.optionalString (!result.config.home.shellAliases ? cs) ''
          echo "FAIL: Alias 'cs' should be defined"
          exit 1
        ''}
        ${lib.optionalString (!result.config.home.shellAliases ? cs-get) ''
          echo "FAIL: Alias 'cs-get' should be defined"
          exit 1
        ''}
        ${lib.optionalString (!result.config.home.shellAliases ? cs-set) ''
          echo "FAIL: Alias 'cs-set' should be defined"
          exit 1
        ''}
        echo "PASS: Aliases enabled by default"
        touch $out
      '';

    # Test 8: Alias values are correct
    test-alias-values =
      let
        result = evalModule {
          programs.clipshare.enable = true;
        };
        aliases = result.config.home.shellAliases;
      in
      pkgs.runCommand "test-alias-values" { } ''
        if [ "${aliases.cs}" != "clipshare" ]; then
          echo "FAIL: cs alias should be 'clipshare'"
          exit 1
        fi
        if [ "${aliases.cs-get}" != "clipshare get" ]; then
          echo "FAIL: cs-get alias should be 'clipshare get'"
          exit 1
        fi
        if [ "${aliases.cs-set}" != "clipshare set" ]; then
          echo "FAIL: cs-set alias should be 'clipshare set'"
          exit 1
        fi
        echo "PASS: Alias values are correct"
        touch $out
      '';

    # Test 9: Aliases can be disabled
    test-aliases-disabled =
      let
        result = evalModule {
          programs.clipshare = {
            enable = true;
            enableAliases = false;
          };
        };
      in
      pkgs.runCommand "test-aliases-disabled" { } ''
        ${lib.optionalString (result.config.home.shellAliases ? cs) ''
          echo "FAIL: Alias 'cs' should not be defined when aliases disabled"
          exit 1
        ''}
        echo "PASS: Aliases can be disabled"
        touch $out
      '';

    # Test 10: Session variables set correctly when both configured
    test-session-variables =
      let
        result = evalModule {
          programs.clipshare = {
            enable = true;
            url = "http://myserver:8080";
            device = "my-laptop";
          };
        };
        vars = result.config.home.sessionVariables;
      in
      pkgs.runCommand "test-session-variables" { } ''
        ${lib.optionalString (!vars ? CLIPSHARE_URL) ''
          echo "FAIL: CLIPSHARE_URL should be set"
          exit 1
        ''}
        ${lib.optionalString (!vars ? CLIPSHARE_DEVICE) ''
          echo "FAIL: CLIPSHARE_DEVICE should be set"
          exit 1
        ''}
        if [ "${vars.CLIPSHARE_URL}" != "http://myserver:8080" ]; then
          echo "FAIL: CLIPSHARE_URL value incorrect"
          exit 1
        fi
        if [ "${vars.CLIPSHARE_DEVICE}" != "my-laptop" ]; then
          echo "FAIL: CLIPSHARE_DEVICE value incorrect"
          exit 1
        fi
        echo "PASS: Session variables set correctly"
        touch $out
      '';

    # Test 11: Device is null when hostname is null (no env var set)
    test-device-null-when-hostname-null =
      let
        result = evalModule {
          programs.clipshare.enable = true;
          networking.hostName = null;
        };
      in
      pkgs.runCommand "test-device-null-when-hostname-null" { } ''
        ${lib.optionalString (result.config.home.sessionVariables ? CLIPSHARE_DEVICE) ''
          echo "FAIL: CLIPSHARE_DEVICE should not be set when hostname is null"
          exit 1
        ''}
        echo "PASS: Device env var not set when hostname is null"
        touch $out
      '';

    # Test 12: Full configuration with both values set
    test-full-configuration =
      let
        result = evalModule {
          programs.clipshare = {
            enable = true;
            url = "https://secure-clipboard.internal:443";
            device = "workstation-01";
            enableAliases = true;
          };
        };
      in
      pkgs.runCommand "test-full-configuration" { } ''
        # Check packages are installed
        ${lib.optionalString (result.config.home.packages == [ ]) ''
          echo "FAIL: Package should be installed"
          exit 1
        ''}

        # Check session variables
        if [ "${result.config.home.sessionVariables.CLIPSHARE_URL}" != "https://secure-clipboard.internal:443" ]; then
          echo "FAIL: URL not set correctly"
          exit 1
        fi
        if [ "${result.config.home.sessionVariables.CLIPSHARE_DEVICE}" != "workstation-01" ]; then
          echo "FAIL: Device not set correctly"
          exit 1
        fi

        # Check aliases
        ${lib.optionalString (!result.config.home.shellAliases ? cs) ''
          echo "FAIL: Aliases should be enabled"
          exit 1
        ''}

        echo "PASS: Full configuration works"
        touch $out
      '';

    # Test 13: Only URL set, device null
    test-only-url-set =
      let
        result = evalModule {
          programs.clipshare = {
            enable = true;
            url = "http://myserver:8080";
          };
          networking.hostName = null;
        };
        vars = result.config.home.sessionVariables;
      in
      pkgs.runCommand "test-only-url-set" { } ''
        ${lib.optionalString (!vars ? CLIPSHARE_URL) ''
          echo "FAIL: CLIPSHARE_URL should be set"
          exit 1
        ''}
        ${lib.optionalString (vars ? CLIPSHARE_DEVICE) ''
          echo "FAIL: CLIPSHARE_DEVICE should not be set when hostname is null"
          exit 1
        ''}
        echo "PASS: Only URL env var set when device is null"
        touch $out
      '';

    # Test 14: Only device set, URL null
    test-only-device-set =
      let
        result = evalModule {
          programs.clipshare = {
            enable = true;
            device = "my-device";
          };
        };
        vars = result.config.home.sessionVariables;
      in
      pkgs.runCommand "test-only-device-set" { } ''
        ${lib.optionalString (vars ? CLIPSHARE_URL) ''
          echo "FAIL: CLIPSHARE_URL should not be set when url is null"
          exit 1
        ''}
        ${lib.optionalString (!vars ? CLIPSHARE_DEVICE) ''
          echo "FAIL: CLIPSHARE_DEVICE should be set"
          exit 1
        ''}
        if [ "${vars.CLIPSHARE_DEVICE}" != "my-device" ]; then
          echo "FAIL: CLIPSHARE_DEVICE value incorrect"
          exit 1
        fi
        echo "PASS: Only device env var set when URL is null"
        touch $out
      '';

    # Test 15: No env vars when both null
    test-no-env-vars-when-both-null =
      let
        result = evalModule {
          programs.clipshare.enable = true;
          networking.hostName = null;
        };
        vars = result.config.home.sessionVariables;
      in
      pkgs.runCommand "test-no-env-vars-when-both-null" { } ''
        ${lib.optionalString (vars ? CLIPSHARE_URL) ''
          echo "FAIL: CLIPSHARE_URL should not be set"
          exit 1
        ''}
        ${lib.optionalString (vars ? CLIPSHARE_DEVICE) ''
          echo "FAIL: CLIPSHARE_DEVICE should not be set"
          exit 1
        ''}
        echo "PASS: No env vars when both url and device are null"
        touch $out
      '';
  };

  # Combine all tests - use runCommand to aggregate results
  allTests = pkgs.runCommand "home-manager-module-tests" { } ''
    mkdir -p $out
    ${lib.concatMapStringsSep "\n" (
      test: "ln -s ${test} $out/${test.name}"
    ) (builtins.attrValues tests)}
    echo "All Home Manager module tests passed"
  '';
in
{
  inherit tests allTests;
  # Allow running with `nix-build -A allTests`
  __functor = self: self.allTests;
}
