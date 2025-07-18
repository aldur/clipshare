{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.services.clipshare;
in
{
  options.services.clipshare = {
    enable = mkEnableOption "clipshare server";

    package = mkOption {
      type = types.package;
      default = pkgs.clipshare-server;
      defaultText = literalExpression "pkgs.clipshare-server";
      description = "The clipshare server package to use.";
    };

    host = mkOption {
      type = types.str;
      default = "localhost";
      description = "Host to bind the server to.";
    };

    port = mkOption {
      type = types.port;
      default = 8080;
      description = "Port to bind the server to.";
    };

    user = mkOption {
      type = types.str;
      default = "clipshare";
      description = "User to run the clipshare server as.";
    };

    group = mkOption {
      type = types.str;
      default = "clipshare";
      description = "Group to run the clipshare server as.";
    };

    openFirewall = mkOption {
      type = types.bool;
      default = false;
      description = "Whether to open the firewall for the clipshare server.";
    };
  };

  config = mkIf cfg.enable {
    users.users.${cfg.user} = {
      description = "clipshare server user";
      group = cfg.group;
      isSystemUser = true;
    };

    users.groups.${cfg.group} = {};

    systemd.services.clipshare = {
      description = "Clipshare Server";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];
      
      serviceConfig = {
        Type = "simple";
        User = cfg.user;
        Group = cfg.group;
        Restart = "on-failure";
        RestartSec = "5s";
        ExecStart = "${cfg.package}/bin/clipshare-server";
        
        # Security settings
        NoNewPrivileges = true;
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        ReadWritePaths = [];
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
        RestrictAddressFamilies = [ "AF_INET" "AF_INET6" ];
        RestrictNamespaces = true;
        LockPersonality = true;
        MemoryDenyWriteExecute = true;
        RestrictRealtime = true;
        RestrictSUIDSGID = true;
        RemoveIPC = true;
        PrivateDevices = true;
      };
      
      environment = {
        HOST = cfg.host;
        PORT = toString cfg.port;
      };
    };

    networking.firewall = mkIf cfg.openFirewall {
      allowedTCPPorts = [ cfg.port ];
    };
  };
}