{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.programs.clipshare;
  
  # Create wrapper script with environment variables
  clientWrapper = pkgs.writeShellScriptBin "clipshare-client" ''
    export REST_CLIPBOARD_URL="${cfg.url}"
    export REST_CLIPBOARD_DEVICE="${cfg.device}"
    exec ${cfg.package}/bin/clipshare-client "$@"
  '';
  
  # Create convenience aliases
  aliasScript = pkgs.writeShellScriptBin "clipboard-aliases" ''
    alias cb='clipshare-client'
    alias cbget='clipshare-client get'
    alias cbset='clipshare-client set'
    alias cbpush='clipshare-client set'
    alias cbpull='clipshare-client get'
    
    # Function to copy from clipboard to system clipboard (if available)
    cbcopy() {
      if command -v xclip &> /dev/null; then
        clipshare-client get | xclip -selection clipboard
      elif command -v pbcopy &> /dev/null; then
        clipshare-client get | pbcopy
      elif command -v wl-copy &> /dev/null; then
        clipshare-client get | wl-copy
      else
        echo "No system clipboard utility found (xclip, pbcopy, or wl-copy)"
        return 1
      fi
    }
    
    # Function to paste from system clipboard to clipshare
    cbpaste() {
      if command -v xclip &> /dev/null; then
        xclip -selection clipboard -o | clipshare-client set
      elif command -v pbpaste &> /dev/null; then
        pbpaste | clipshare-client set
      elif command -v wl-paste &> /dev/null; then
        wl-paste | clipshare-client set
      else
        echo "No system clipboard utility found (xclip, pbpaste, or wl-paste)"
        return 1
      fi
    }
    
    # Function to sync both ways
    cbsync() {
      case "$1" in
        push)
          cbpaste
          echo "Pushed system clipboard to clipshare"
          ;;
        pull)
          cbcopy
          echo "Pulled clipshare to system clipboard"
          ;;
        *)
          echo "Usage: cbsync [push|pull]"
          echo "  push: Copy from system clipboard to clipshare"
          echo "  pull: Copy from clipshare to system clipboard"
          return 1
          ;;
      esac
    }
  '';
in
{
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
      defaultText = literalExpression "config.networking.hostName or \"unknown\"";
      description = "Device name to use when setting clipboard content.";
    };

    enableAliases = mkOption {
      type = types.bool;
      default = true;
      description = "Whether to enable shell aliases and functions.";
    };

    enableBashIntegration = mkOption {
      type = types.bool;
      default = true;
      description = "Whether to enable bash integration.";
    };

    enableZshIntegration = mkOption {
      type = types.bool;
      default = true;
      description = "Whether to enable zsh integration.";
    };

    enableFishIntegration = mkOption {
      type = types.bool;
      default = true;
      description = "Whether to enable fish integration.";
    };

    extraConfig = mkOption {
      type = types.lines;
      default = "";
      description = "Extra shell configuration for clipshare.";
    };
  };

  config = mkIf cfg.enable {
    home.packages = [ clientWrapper ];

    home.sessionVariables = {
      REST_CLIPBOARD_URL = cfg.url;
      REST_CLIPBOARD_DEVICE = cfg.device;
    };

    programs.bash = mkIf (cfg.enableBashIntegration && config.programs.bash.enable) {
      initExtra = mkIf cfg.enableAliases ''
        # clipshare aliases and functions
        ${readFile "${aliasScript}/bin/clipboard-aliases"}
        
        ${cfg.extraConfig}
      '';
    };

    programs.zsh = mkIf (cfg.enableZshIntegration && config.programs.zsh.enable) {
      initExtra = mkIf cfg.enableAliases ''
        # clipshare aliases and functions
        ${readFile "${aliasScript}/bin/clipboard-aliases"}
        
        ${cfg.extraConfig}
      '';
    };

    programs.fish = mkIf (cfg.enableFishIntegration && config.programs.fish.enable) {
      shellInit = mkIf cfg.enableAliases ''
        # clipshare aliases and functions
        alias cb='clipshare-client'
        alias cbget='clipshare-client get'
        alias cbset='clipshare-client set'
        alias cbpush='clipshare-client set'
        alias cbpull='clipshare-client get'
        
        function cbcopy
          if command -v xclip &> /dev/null
            clipshare-client get | xclip -selection clipboard
          else if command -v pbcopy &> /dev/null
            clipshare-client get | pbcopy
          else if command -v wl-copy &> /dev/null
            clipshare-client get | wl-copy
          else
            echo "No system clipboard utility found (xclip, pbcopy, or wl-copy)"
            return 1
          end
        end
        
        function cbpaste
          if command -v xclip &> /dev/null
            xclip -selection clipboard -o | clipshare-client set
          else if command -v pbpaste &> /dev/null
            pbpaste | clipshare-client set
          else if command -v wl-paste &> /dev/null
            wl-paste | clipshare-client set
          else
            echo "No system clipboard utility found (xclip, pbpaste, or wl-paste)"
            return 1
          end
        end
        
        function cbsync
          switch $argv[1]
            case push
              cbpaste
              echo "Pushed system clipboard to clipshare"
            case pull
              cbcopy
              echo "Pulled clipshare to system clipboard"
            case '*'
              echo "Usage: cbsync [push|pull]"
              echo "  push: Copy from system clipboard to clipshare"
              echo "  pull: Copy from clipshare to system clipboard"
              return 1
          end
        end
        
        ${cfg.extraConfig}
      '';
    };
  };
}