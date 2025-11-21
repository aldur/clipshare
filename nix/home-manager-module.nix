{
  config,
  lib,
  pkgs,
  ...
}:

with lib;

let
  cfg = config.programs.clipshare;

  # Build environment variable exports only for configured options
  envExports = concatStringsSep "\n" (
    optional (cfg.url != null) ''export CLIPSHARE_URL="${cfg.url}"''
    ++ optional (cfg.device != null) ''export CLIPSHARE_DEVICE="${cfg.device}"''
  );

  clientWrapper = pkgs.writeShellScriptBin "clipshare" ''
    ${envExports}
    exec ${cfg.package}/bin/clipshare "$@"
  '';

  aliases = {
    cs = "clipshare";
    cs-get = "clipshare get";
    cs-set = "clipshare set";
  };
in
{
  options.programs.clipshare = {
    enable = mkEnableOption "clipshare client";

    package = mkOption {
      type = types.package;
      default = pkgs.clipshare;
      defaultText = literalExpression "pkgs.clipshare";
      description = "The clipshare client package to use.";
    };

    url = mkOption {
      type = types.nullOr types.str;
      default = null;
      description = "URL of the clipshare server. If not set, no CLIPSHARE_URL environment variable will be exported.";
    };

    device = mkOption {
      type = types.nullOr types.str;
      default = config.networking.hostName or null;
      defaultText = literalExpression "config.networking.hostName or null";
      description = "Device name to use when setting clipboard content. Defaults to the hostname. If null, no CLIPSHARE_DEVICE environment variable will be exported.";
    };

    enableAliases = mkOption {
      type = types.bool;
      default = true;
      description = "Whether to enable shell aliases and functions.";
    };
  };

  config = mkIf cfg.enable {
    home = {
      packages = [ clientWrapper ];

      sessionVariables =
        optionalAttrs (cfg.url != null) { CLIPSHARE_URL = cfg.url; }
        // optionalAttrs (cfg.device != null) { CLIPSHARE_DEVICE = cfg.device; };

      shellAliases = mkIf cfg.enableAliases aliases;
    };
  };
}
