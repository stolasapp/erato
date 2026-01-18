# Erato NixOS module
# https://wiki.nixos.org/wiki/NixOS_modules
flake:
{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.services.erato;
  settingsFormat = pkgs.formats.yaml { };
in
{
  options.services.erato = {
    enable = lib.mkEnableOption "Erato archive proxy service";

    package = lib.mkOption {
      type = lib.types.package;
      default = flake.packages.${pkgs.stdenv.hostPlatform.system}.erato;
      defaultText = lib.literalExpression "flake.packages.\${pkgs.stdenv.hostPlatform.system}.erato";
      description = "The Erato package to use.";
    };

    configFile = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = ''
        Path to an existing Erato configuration file.
        Mutually exclusive with `settings`.
      '';
    };

    settings = lib.mkOption {
      type = lib.types.nullOr (
        lib.types.submodule {
          freeformType = settingsFormat.type;

          options = {
            log_level = lib.mkOption {
              type = lib.types.enum [
                "DEBUG"
                "INFO"
                "WARN"
                "ERROR"
              ];
              default = "INFO";
              description = "Log level for the service.";
            };

            rpc_address = lib.mkOption {
              type = lib.types.str;
              default = "127.0.0.1:9998";
              description = "Address for the ConnectRPC server.";
            };

            web_address = lib.mkOption {
              type = lib.types.str;
              default = "127.0.0.1:9999";
              description = "Address for the web frontend server.";
            };

            db_filepath = lib.mkOption {
              type = lib.types.str;
              default = "/var/lib/erato/db.sqlite";
              description = "Path to the SQLite database file.";
            };

            root_uri = lib.mkOption {
              type = lib.types.str;
              default = "";
              description = ''
                Root URL of the upstream archive to proxy.
                Required unless `dev_mode` is enabled.
              '';
            };

            dev_mode = lib.mkOption {
              type = lib.types.bool;
              default = false;
              description = "Enable development mode with fake upstream service.";
            };
          };
        }
      );
      default = null;
      description = ''
        Erato configuration settings.
        Mutually exclusive with `configFile`.
        Requires `root_uri` to be set unless `dev_mode` is enabled.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    assertions = [
      {
        assertion = !(cfg.configFile != null && cfg.settings != null);
        message = "services.erato: `configFile` and `settings` are mutually exclusive.";
      }
      {
        assertion = cfg.configFile != null || cfg.settings != null;
        message = "services.erato: either `configFile` or `settings` must be specified.";
      }
      {
        assertion = cfg.settings == null || cfg.settings.dev_mode || cfg.settings.root_uri != "";
        message = "services.erato: `settings.root_uri` is required unless `settings.dev_mode` is enabled.";
      }
    ];

    systemd.services.erato = {
      description = "Erato Archive Proxy";
      after = [ "network.target" ];
      wantedBy = [ "multi-user.target" ];

      serviceConfig = {
        Type = "simple";
        ExecStart =
          let
            configPath =
              if cfg.configFile != null then
                cfg.configFile
              else
                settingsFormat.generate "erato.yaml" cfg.settings;
          in
          "${lib.getExe cfg.package} serve --config ${configPath}";

        Restart = "on-failure";
        RestartSec = 5;

        # User/group isolation
        DynamicUser = true;
        StateDirectory = "erato";

        # Filesystem protection
        ProtectSystem = "strict";
        ProtectHome = true;
        PrivateTmp = true;
        PrivateDevices = true;
        PrivateMounts = true;

        # Kernel protection
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectKernelLogs = true;
        ProtectControlGroups = true;
        ProtectHostname = true;
        ProtectClock = true;
        ProtectProc = "invisible";

        # Capability/privilege restrictions
        NoNewPrivileges = true;
        CapabilityBoundingSet = "";
        AmbientCapabilities = "";
        RestrictSUIDSGID = true;
        LockPersonality = true;

        # Namespace/syscall restrictions
        RestrictNamespaces = true;
        RestrictRealtime = true;
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
        ];
        SystemCallFilter = [
          "@system-service"
          "~@privileged"
        ];
        SystemCallErrorNumber = "EPERM";
        MemoryDenyWriteExecute = true;

        # Syscall architecture restriction
        SystemCallArchitectures = "native";

        # Resource controls
        DevicePolicy = "closed";
        UMask = "0077";
        ProcSubset = "pid";
      };
    };
  };
}
