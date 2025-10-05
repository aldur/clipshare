{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.programs.clipshare;

  clientWrapper = pkgs.writeShellScriptBin "clipshare-client" ''
    export REST_CLIPBOARD_URL="${cfg.url}"
    export REST_CLIPBOARD_DEVICE="${cfg.device}"
    exec ${cfg.package}/bin/clipshare-client "$@"
  '';

  aliases = {
    cb = "clipshare-client";
    cb-get = "clipshare-client get";
    cb-set = "clipshare-client set";
  };
in {
  options.programs.clipshare = {
    enable = mkEnableOption "clipshare client";

    package = mkOption {
      type = types.package;
      default = pkgs.clipshare-client;
      defaultText = literalExpression "pkgs.clipshare-client";
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
      REST_CLIPBOARD_URL = cfg.url;
      REST_CLIPBOARD_DEVICE = cfg.device;
    };

    home.shellAliases = mkIf enableAliases.enable aliases;
  };
}

