{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.programs.clipshare;

  clientWrapper = pkgs.writeShellScriptBin "clipshare" ''
    export CLIPSHARE_DEVICE="${cfg.url}"
    export CLIPSHARE_URL="${cfg.device}"
    exec ${cfg.package}/bin/clipshare "$@"
  '';

  aliases = {
    cs = "clipshare";
    cs-get = "clipshare get";
    cs-set = "clipshare set";
  };
in {
  options.programs.clipshare = {
    enable = mkEnableOption "clipshare client";

    package = mkOption {
      type = types.package;
      default = pkgs.clipshare;
      defaultText = literalExpression "pkgs.clipshare";
      description = "The clipshare client package to use.";
    };

    url = mkOption {
      type = types.str;
      default = "http://localhost:8080";
      description = "URL of the clipshare server.";
    };

    device = mkOption {
      type = types.str;
      default = config.networking.hostName or "unknown";
      defaultText =
        literalExpression ''config.networking.hostName or "unknown"'';
      description = "Device name to use when setting clipboard content.";
    };

    enableAliases = mkOption {
      type = types.bool;
      default = true;
      description = "Whether to enable shell aliases and functions.";
    };
  };

  config = mkIf cfg.enable {
    home.packages = [ clientWrapper ];

    home.sessionVariables = {
      CLIPSHARE_URL = cfg.url;
      CLIPSHARE_DEVICE = cfg.device;
    };

    home.shellAliases = mkIf enableAliases.enable aliases;
  };
}

