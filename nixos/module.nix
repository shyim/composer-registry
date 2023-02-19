{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.services.composer-registry;

  settingsFormat = pkgs.formats.json { };

  configFile = settingsFormat.generate "config.json" cfg.settings;
in
{
  options.services.composer-registry = {
    enable = mkEnableOption (lib.mdDoc "Composer Registry");

    package = mkOption {
      type = types.package;
      default = pkgs.composer-registry;
      defaultText = "pkgs.composer-registry";
      description = "The composer-registry package to use.";
    };

    settings = lib.mkOption {
      type = lib.types.submodule {
        freeformType = settingsFormat.type;

        options."base_url" = lib.mkOption {
          type = lib.types.str;
          default = "http://localhost:8000";
          description = lib.mdDoc ''
            Base URL
          '';
        };

        options."storage_path" = lib.mkOption {
          type = lib.types.str;
          default = "/var/lib/composer-registry";
          description = lib.mdDoc ''
            Storage Path
          '';
        };
      };

      default = { };

      description = lib.mdDoc ''
        Composer Registry configuration.
      '';
    };
  };

  config = mkIf cfg.enable {
    users.users.composer-registry = {
      description = "The composer-registry service user";
      group = "composer-registry";
      isSystemUser = true;
    };

    users.groups.composer-registry = { };

    systemd.services.composer-registry = {
      wantedBy = [ "multi-user.target" ];
      environment = {
        COMPOSER_REGISTRY_CONFIG_PATH = configFile;
      };
      serviceConfig = {
        ExecStart = "${cfg.package}/bin/composer-registry";
        User = "composer-registry";
        Group = "composer-registry";
        StateDirectory = "composer-registry";
        RuntimeDirectory = "composer-registry";
        StateDirectoryMode = "0770";
        RuntimeDirectoryMode = "0770";
        UMask = "0002";
      };
    };
  };
}
